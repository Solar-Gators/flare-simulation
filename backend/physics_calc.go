package main

/*
**Important Vars.**

- Cd = track coeff
- A = frontal area
- Crr = Rolling Resistance

*/

import(
	"fmt"
	"math"
)

//speed going into curve and throughout it
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



//Power required to maintaining coasting speed
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

//
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

// Calculates the max speed of upcoming curve

//determines distance to let off accelerator before a curve to hit target speed
//taking in the same params as CalcGForce to be passed into it
func calcCoastDistance(
	currentSpeed float64,
	segment Segment,
	aDrag float64,
	rRes float64,
	gravity float64,
	gmax float64,
	mass float64,
	fArea float64, 
	rho float64) float64 {
	//segment: upcoming curve segment
	//aDrag: aerodynamic drag
	//rRes: rolling resistance
	//fArea: frontal area
	//rho: air density

	//target speed obtained from CalcGforce func
	targetSpeed := calcCurveSpeed(segment, gravity, gmax) //m/s
	//squaring initial and end speeds
	initSpeedSquared := math.Pow(currentSpeed, 2)
	endSpeedSquared := math.Pow(targetSpeed, 2)

	//quotient of coasting distance formula
	startTerm := (((rho*aDrag*fArea)/(2*mass)*initSpeedSquared + (rRes*gravity)))
	endTerm := (((rho*aDrag*fArea)/(2*mass)*endSpeedSquared + (rRes*gravity)))
	
	lnTerm := math.Log(startTerm/endTerm)

	coastDistance := ((mass/(rho*aDrag*fArea))*lnTerm)
	return coastDistance
}

/*
Next funcs all connect with one another together 
**Coast Conservation - Curve Accel Energy = Net Curve Losses**
*/

// How much energy is saved by letting off the gas before a curve
func coastConservation(cruiseEnergy, bottomEnergy, distance, curveSpeed, cruiseSpeed float64) float64 {
	//use time = distance / speed (because we need to energy * time (wh))
	avgSpeed := math.Abs(curveSpeed - cruiseSpeed) / 2 // find average
	time := distance / avgSpeed // in seconds
	//bottomEnergy is arbitrary constant 
	energySavedWh := (cruiseEnergy - bottomEnergy) * time / 3600.0
	return energySavedWh
}


// curveAccelEnergy calculates the total energy (in Wh) required to accelerate
// a vehicle from initSpeed to cruiseSpeed under constant accel,
// including aerodynamic drag and rolling resistance losses.
func curveAccelEnergy(
    mass float64,           // Vehicle mass (kg)
    fArea float64,    // Frontal area (m^2)
    aDrag float64,// Aerodynamic drag coefficient (Cd)
    rRes float64, // Rolling resistance coefficient (Cr)
    initSpeed float64,      // Initial speed (m/s)
    cruiseSpeed float64,     // cruise speed (m/s)
    accel float64,   // Constant accel (m/s^2)
    rho float64,     // Air density (kg/m^3)
    gravity float64,        // Gravitational accel (m/s^2)
) (float64) {

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

//finding net loss by combining curveAccelEnergy and coastConservation
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
	gmax float64  ) float64 {
	curveSpeed := calcCurveSpeed(upcomingCurve, gravity, gmax)
	energyUsed := curveAccelEnergy(mass, fArea, aDrag, rRes, curveSpeed, cruiseSpeed, accel, rho, gravity)
	energySaved := coastConservation(cruiseE, bottomE, distance, curveSpeed, cruiseSpeed)
	return energyUsed - energySaved
}

//unused for now...
func simulateCoast(race_track Track){
	//iterating through track and finding coast distance prior to a curve to hit target speed
	//turn loop into function?
	//implement coastConservation function?
	//the car hits target speed and goes at target speed throughout curve (ends the curve at target speed)
	// --> accelerate from current speed to coast speed 
	// --> iterate through constant accelerate options to find optimal constant accel rate

	for i := 0; i < len(race_track.Segments); i++ {
		if ((i+1) == len(race_track.Segments)){
			break
		}
		speed := 27.0 //in meters per second
		upcomingCurve := race_track.Segments[i+1]
		aDrag := 0.21 //Cd
		rRes := 0.0015 //Crr
		gravity := 9.81
		gmax := 0.8
		mass := 285.0
		fArea := 0.456 //frontal area
		rho := 1.225 //air density
		bottomE := 0.006 //Watt hours per meter (wh/m)
		accel := 0.5//m/s^2
		if (upcomingCurve.Radius > 0) {
			DistanceToCoast := calcCoastDistance(speed, upcomingCurve, aDrag, rRes, gravity, gmax, mass, fArea, rho)
			if (DistanceToCoast < 0){
				fmt.Println("Already below current speed")
			} else if (DistanceToCoast > race_track.Segments[i].Length){
				fmt.Println("Not enough track to slow down: ", DistanceToCoast, "meters needed")
			} else {fmt.Println("Coast to target with: ", DistanceToCoast, "meters")

			//theta (representing elevation) is temporarily 0
			//Cruise energy used in wh/m
			cruiseWhPerM := PowerRequired(speed, mass, gravity, rRes, rho, aDrag, fArea, 0) / speed / 3600.0 
			fmt.Println("Cruise Energy in Wh/m: ", cruiseWhPerM)

			//finds the net losses between conserved energy (from coasting) and used energy (from accel)
			netE := netCurveLosses(mass, fArea, aDrag, rRes, upcomingCurve, speed, accel, rho, gravity, cruiseWhPerM, bottomE, DistanceToCoast, gmax)
			fmt.Println("Net loss of energy is: ", netE, " wh")
			}
			fmt.Println("--------------------------------------------------")
		} 
	}
}

