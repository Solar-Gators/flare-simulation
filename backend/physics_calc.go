package main

/*
**Important Vars.**

- Cd = track coeff
- A = frontal area
- Crr = Rolling Resistance

*/

/*
**Important Vars.**

- Cd = track coeff
- A = frontal area
- Crr = Rolling Resistance

*/

import (
	"math"
)

// trackSample captures a single distance sample along the track.
// Intended for precomputing per-meter constraints (for example, curve speed caps)
// before running a forward/backward speed pass.
type trackSample struct {
	SegmentIndex     int
	SegmentOffsetM   float64
	TrackDistanceM   float64
	StepLengthM      float64
	RadiusM          float64
	CurveSpeedCapMPS float64
}

// speedProfile is a speed limit array: max allowed speed at each sampled meter.
type speedProfile []float64

type profileSet struct {
	Base  speedProfile
	Brake speedProfile
	Coast speedProfile
}

// sampleTrackMeters builds fixed-distance samples for every segment in track.
func sampleTrackMeters(track Track, stepM, gravity, gmax float64) []trackSample {
	if stepM <= 0 {
		stepM = 1.0
	}

	// Small default capacity to limit reallocations for typical tracks.
	samples := make([]trackSample, 0, 1024)
	trackDistanceM := 0.0

	for segIdx, seg := range track.Segments {
		segLen := seg.getArcLength()
		if segLen <= 0 {
			continue
		}

		radius := 0.0
		curveCap := math.Inf(1)
		if seg.Radius > 0 {
			radius = seg.Radius
			curveCap = calcCurveSpeed(seg, gravity, gmax)
		}

		for segOffset := 0.0; segOffset < segLen; segOffset += stepM {
			ds := math.Min(stepM, segLen-segOffset)
			samples = append(samples, trackSample{
				SegmentIndex:     segIdx,
				SegmentOffsetM:   segOffset,
				TrackDistanceM:   trackDistanceM,
				StepLengthM:      ds,
				RadiusM:          radius,
				CurveSpeedCapMPS: curveCap,
			})
			trackDistanceM += ds
		}
	}

	return samples
}

// buildSpeedProfile converts meter samples into a speed limit array.
// Each point is capped by cruiseCapMPS, then further limited by local curve cap.
func buildSpeedProfile(samples []trackSample, cruiseCapMPS float64) speedProfile {
	profile := make(speedProfile, len(samples))
	for i, s := range samples {
		limit := s.CurveSpeedCapMPS
		if cruiseCapMPS > 0 && limit > cruiseCapMPS {
			limit = cruiseCapMPS
		}
		profile[i] = limit
	}
	return profile
}

// backwardFeasibilityPass performs a backward pass on the speed profile.
// It ensures each sample's speed is low enough that the car can still brake
// to the next sample's feasible speed over the available distance.
func backwardFeasibilityPass(limits speedProfile, samples []trackSample, maxBrakeMPS2 float64) speedProfile {
	feasible := make(speedProfile, len(limits))
	copy(feasible, limits)

	if len(feasible) == 0 || maxBrakeMPS2 <= 0 {
		return feasible
	}
	if len(samples) != len(feasible) {
		return feasible
	}

	for i := len(feasible) - 2; i >= 0; i-- {
		ds := samples[i].StepLengthM
		if ds <= 0 {
			ds = 1.0
		}

		next := feasible[i+1]
		maxHereForNext := math.Sqrt(math.Max(0, next*next+2*maxBrakeMPS2*ds))
		if feasible[i] > maxHereForNext {
			feasible[i] = maxHereForNext
		}
	}

	return feasible
}

// backwardCoastFeasibilityPass performs a backward pass using only coasting decel.
// This is stricter than brake feasibility and answers:
// "what speed here is still slow enough to make the next point by just lifting?"
func backwardCoastFeasibilityPass(
	limits speedProfile,
	samples []trackSample,
	vMin float64,
	m float64,
	g float64,
	Crr float64,
	rho float64,
	Cd float64,
	A float64,
	theta float64,
) speedProfile {
	feasible := make(speedProfile, len(limits))
	copy(feasible, limits)

	if len(feasible) == 0 {
		return feasible
	}
	if len(samples) != len(feasible) {
		return feasible
	}

	for i := len(feasible) - 2; i >= 0; i-- {
		ds := samples[i].StepLengthM
		if ds <= 0 {
			ds = 1.0
		}

		next := feasible[i+1]
		coastA := -coastDecelFromPower(next, vMin, m, g, Crr, rho, Cd, A, theta)
		if coastA <= 0 {
			continue
		}

		maxHereForNext := math.Sqrt(math.Max(0, next*next+2*coastA*ds))
		if feasible[i] > maxHereForNext {
			feasible[i] = maxHereForNext
		}
	}

	return feasible
}

// enforceBrakeLookahead tightens a coast-first/brake-only-as-needed command so the
// target speed is still reachable by the lookahead distance.
func enforceBrakeLookahead(aCmd, vNow, vTarget, distToTarget, aBrakeMax float64) float64 {
	if aBrakeMax <= 0 {
		if aCmd < 0 {
			return 0
		}
		return aCmd
	}
	if distToTarget > 0 {
		aReq := (vTarget*vTarget - vNow*vNow) / (2 * distToTarget)
		if aCmd > aReq {
			aCmd = aReq
		}
	}
	if aCmd < -aBrakeMax {
		aCmd = -aBrakeMax
	}
	return aCmd
}

// jerkBrakeBufferDistance estimates the extra distance needed to ramp from the
// current longitudinal acceleration to a more negative braking command.
func jerkBrakeBufferDistance(aPrev, aTarget, vNow, vMin, jerkMax float64) float64 {
	if jerkMax <= 0 || aTarget >= aPrev {
		return 0
	}
	tRamp := (aPrev - aTarget) / jerkMax
	return math.Max(vNow, vMin) * tRamp
}

// apexLookaheadHorizon estimates how far ahead to search for an upcoming apex.
func apexLookaheadHorizon(vNow, vMin, aBrakeMax, leadInM float64) float64 {
	const minLookaheadM = 80.0

	if aBrakeMax <= 0 {
		return minLookaheadM
	}

	vEff := math.Max(vNow, vMin)
	brakeDistance := (vEff * vEff) / (2 * aBrakeMax)
	return math.Max(minLookaheadM, brakeDistance+leadInM+vEff)
}

// buildProfiles constructs base, brake-feasible, and coast-feasible profiles together.
func buildProfiles(
	samples []trackSample,
	cruiseCapMPS float64,
	maxBrakeMPS2 float64,
	vMin float64,
	m float64,
	g float64,
	Crr float64,
	rho float64,
	Cd float64,
	A float64,
	theta float64,
) profileSet {
	base := buildSpeedProfile(samples, cruiseCapMPS)
	brake := backwardFeasibilityPass(base, samples, maxBrakeMPS2)
	coast := backwardCoastFeasibilityPass(base, samples, vMin, m, g, Crr, rho, Cd, A, theta)
	return profileSet{
		Base:  base,
		Brake: brake,
		Coast: coast,
	}
}

// speed going into curve and throughout it
func calcCurveSpeed(segments Segment, gravity float64, gmax float64) float64 {
	radius := segments.Radius
	maxVelocity := math.Sqrt(gmax * radius * gravity)
	return maxVelocity
}

// func newTotalEnergy(solarYield float64, /* watt hours / minute */
// 	raceDayTime float64, /* in minutes */
// 	batterySize float64 /* in watt hours */) float64 {
// 	totalBattery := (solarYield * raceDayTime) + batterySize
// 	return totalBattery
// }

// Power required to maintaining coasting speed
func PowerRequired(v, m, g, Crr, rho, Cd, A, theta float64) float64 {
	return (Crr*m*g+m*g*math.Sin(theta))*v + 0.5*rho*Cd*A*v*v*v
}

//Calculates wheel mechanical power
//Useful for finding distance
//At this speed, can the car even produce the wheel power required to overcome resistances?
//Useful as a feasability check

// v: vehicle speed [m/s]
// Tmax: max wheel torque [N·m] (post-gearing)
// Pmax: electrical power cap at battery/inverter [W]
// rWheel: wheel effective radius [m]
// eta: battery→wheel efficiency [0..1]
// Returns: available wheel mechanical power [W]

func WheelPowerEV(v, Tmax, Pmax, rWheel, eta float64) float64 {

	if v <= 0 || rWheel <= 0 || eta <= 0 {
		return 0
	}
	// Wheel angular speed [rad/s]
	omegaWheel := v / rWheel

	// Torque-limited wheel power (mechanical)
	P_torque := Tmax * omegaWheel // [W]

	// Power cap at the wheel after drivetrain losses
	P_cap := eta * Pmax // [W]

	if P_torque < P_cap {
		return P_torque
	}
	return P_cap
}

func DistanceForSpeedEV(
	v float64,
	batteryWh, solarWhPerMin, etaDrive, raceDayMin float64,
	// EV capability:
	rWheel, Tmax, Pmax float64,
	// Env/vehicle:
	m, g, Crr, rho, Cd, A, theta float64,
) (float64, bool) {
	if v <= 0 || raceDayMin <= 0 || etaDrive <= 0 {
		return 0, false
	}
	Preq := PowerRequired(v, m, g, Crr, rho, Cd, A, theta) // wheel W
	if Preq <= 0 || math.IsNaN(Preq) || math.IsInf(Preq, 0) {
		return 0, false
	}
	// Feasibility check
	if WheelPowerEV(v, Tmax, Pmax, rWheel, etaDrive)+1e-9 < Preq {
		return 0, false
	}

	Tsec := raceDayMin * 60.0
	EbattWheelJ := batteryWh * 3600.0 * etaDrive   // wheel J
	PsolarWheel := solarWhPerMin * 60.0 * etaDrive // wheel W

	// If solar alone covers demand, we can run full 8h
	if Preq <= PsolarWheel {
		return v * Tsec, true
	}
	drain := Preq - PsolarWheel
	tEnd := EbattWheelJ / drain
	if tEnd > Tsec {
		tEnd = Tsec
	}
	if tEnd < 0 {
		tEnd = 0
	}
	return v * tEnd, true
}

func coastDecelFromPower(v, vMin, m, g, Crr, rho, Cd, A, theta float64) float64 {
	vEff := math.Max(v, vMin)
	Pres := PowerRequired(v, m, g, Crr, rho, Cd, A, theta)
	Fres := Pres / vEff
	return -Fres / m
}

// Calculates the max speed of upcoming curve

// determines distance to let off accelerator before a curve to hit target speed
// taking in the same params as CalcGForce to be passed into it
func coastDistanceToSpeed(
	v0, vTarget float64,
	stepM float64,
	vMin float64,
	m, g, Crr, rho, Cd, A, theta float64,
) (dist float64) {
	if v0 <= vTarget {
		return 0
	}
	v := v0
	for v > vTarget {
		a := coastDecelFromPower(v, vMin, m, g, Crr, rho, Cd, A, theta) // negative
		if a >= 0 {
			// should never happen unless downhill overwhelms drag+rr
			break
		}
		v2 := v*v + 2*a*stepM
		if v2 <= 0 {
			// comes to stop within this step
			dist += (v * v) / (-2 * a)
			return dist
		}
		v = math.Sqrt(v2)
		dist += stepM
		// safety bound to avoid infinite loops in weird inputs
		if dist > 1e7 {
			break
		}
	}
	return dist
}

/*
Next funcs all connect with one another together
**Coast Conservation - Curve Accel Energy = Net Curve Losses**
*/

// How much energy is saved by letting off the gas before a curve
func coastConservation(cruiseEnergy, bottomEnergy, distance, curveSpeed, cruiseSpeed float64) float64 {
	//use time = distance / speed (because we need to energy * time (wh))
	avgSpeed := math.Abs(curveSpeed-cruiseSpeed) / 2 // find average
	time := distance / avgSpeed                      // in seconds
	//bottomEnergy is arbitrary constant
	energySavedWh := (cruiseEnergy - bottomEnergy) * time / 3600.0
	return energySavedWh
}

// How much energy is used given a constant accel after a curve

// curveAccelEnergy calculates the total energy (in Wh) required to accelerate
// a vehicle from initSpeed to cruiseSpeed under constant accel,
// including aerodynamic drag and rolling resistance losses.
func curveAccelEnergy(
	mass float64, // Vehicle mass (kg)
	fArea float64, // Frontal area (m^2)
	aDrag float64, // Aerodynamic drag coefficient (Cd)
	rRes float64, // Rolling resistance coefficient (Cr)
	initSpeed float64, // Initial speed (m/s)
	cruiseSpeed float64, // cruise speed (m/s)
	accel float64, // Constant accel (m/s^2)
	rho float64, // Air density (kg/m^3)
	gravity float64, // Gravitational accel (m/s^2)
) float64 {

	if cruiseSpeed <= initSpeed {
		return 0
	}
	if accel <= 0 {
		return 0
	}

	// 1. Kinetic energy change (Joules)
	deltaKE := 0.5 * mass * (math.Pow(cruiseSpeed, 2) - math.Pow(initSpeed, 2))

	// 2. Distance traveled during accel (m)
	distance := (math.Pow(cruiseSpeed, 2) - math.Pow(initSpeed, 2)) / (2 * accel)

	// 3. Aerodynamic drag energy (Joules)
	// E_drag = (0.5 * rho * Cd * A / a) * (v_f^4 - v_i^4) / 4
	eDrag := (0.5 * rho * aDrag * fArea / accel) *
		(math.Pow(cruiseSpeed, 4) - math.Pow(initSpeed, 4)) / 4.0

	// 4. Rolling resistance energy (Joules)
	eRoll := rRes * mass * gravity * distance

	// 5. Total energy in Joules → convert to Wh (1 Wh = 3600 J)
	totalEnergyWh := (deltaKE + eDrag + eRoll) / 3600.0

	return totalEnergyWh
}

// finding net loss by combining curveAccelEnergy and coastConservation
func netCurveLosses(
	mass float64,
	fArea float64,
	aDrag float64,
	rRes float64,
	upcomingCurve Segment,
	cruiseSpeed float64,
	accel float64,
	rho float64,
	gravity float64,
	cruiseE float64,
	bottomE float64,
	distance float64,
	gmax float64) float64 {
	curveSpeed := calcCurveSpeed(upcomingCurve, gravity, gmax)
	energyUsed := curveAccelEnergy(mass, fArea, aDrag, rRes, curveSpeed, cruiseSpeed, accel, rho, gravity)
	energySaved := coastConservation(cruiseE, bottomE, distance, curveSpeed, cruiseSpeed)
	return energyUsed - energySaved
}

//unused for now...
// func simulateCoast(race_track Track){
// 	//iterating through track and finding coast distance prior to a curve to hit target speed
// 	//turn loop into function?
// 	//implement coastConservation function?
// 	//the car hits target speed and goes at target speed throughout curve (ends the curve at target speed)
// 	// --> accelerate from current speed to coast speed
// 	// --> iterate through constant accelerate options to find optimal constant accel rate

// 	for i := 0; i < len(race_track.Segments); i++ {
// 		if ((i+1) == len(race_track.Segments)){
// 			break
// 		}
// 		speed := 27.0 //in meters per second
// 		upcomingCurve := race_track.Segments[i+1]
// 		aDrag := 0.21 //Cd
// 		rRes := 0.0015 //Crr
// 		gravity := 9.81
// 		gmax := 0.8
// 		mass := 285.0
// 		fArea := 0.456 //frontal area
// 		rho := 1.225 //air density
// 		bottomE := 0.006 //Watt hours per meter (wh/m)
// 		accel := 0.5//m/s^2
// 		if (upcomingCurve.Radius > 0) {
// 			DistanceToCoast := calcCoastDistance(speed, upcomingCurve, aDrag, rRes, gravity, gmax, mass, fArea, rho)
// 			if (DistanceToCoast < 0){
// 				fmt.Println("Already below current speed")
// 			} else if (DistanceToCoast > race_track.Segments[i].Length){
// 				fmt.Println("Not enough track to slow down: ", DistanceToCoast, "meters needed")
// 			} else {fmt.Println("Coast to target with: ", DistanceToCoast, "meters")

// 			//theta (representing elevation) is temporarily 0
// 			//Cruise energy used in wh/m
// 			cruiseWhPerM := PowerRequired(speed, mass, gravity, rRes, rho, aDrag, fArea, 0) / speed / 3600.0
// 			fmt.Println("Cruise Energy in Wh/m: ", cruiseWhPerM)

// 			//finds the net losses between conserved energy (from coasting) and used energy (from accel)
// 			netE := netCurveLosses(mass, fArea, aDrag, rRes, upcomingCurve, speed, accel, rho, gravity, cruiseWhPerM, bottomE, DistanceToCoast, gmax)
// 			fmt.Println("Net loss of energy is: ", netE, " wh")
// 			}
// 			fmt.Println("--------------------------------------------------")
// 		}
// 	}
// }
