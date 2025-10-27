package main

import (
	"math"
)

// Calculates the max speed of upcoming curve
func CalcGforce(segments Segment, gravity float64, gmax float64) float64 {
	radius := segments.Radius
	maxVelocity := math.Sqrt(gmax * radius * gravity)
	return maxVelocity
}

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
	targetSpeed := CalcGforce(segment, gravity, gmax) //m/s
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

// How much energy is saved by letting off the gas before a curve
func coastConservation(cruiseEnergy, bottomEnergy, distance float64) float64 {
	//bottomEnergy is arbitrary constant 
	energyConserved := distance * (cruiseEnergy - bottomEnergy)
	return energyConserved
}

// How much energy is used given a constant accel after a curve
// func curveAccelEnergy(cruiseSpeed float64, currentSpeed float64, accel float64, mass float64) float64 {
// 	energyUsed := 0.5 * mass * ((math.Pow(cruiseSpeed, 2)) - (math.Pow(currentSpeed, 2)))
// 	return energyUsed / 3600.0 //returns wh
// }
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
	curveSpeed float64, 
	cruiseSpeed float64, 
	accel float64, 
	rho float64, 
	gravity float64, 
	cruiseE float64, 
	bottomE float64,
	distance float64  ) float64 {
	energyUsed := curveAccelEnergy(mass, fArea, aDrag, rRes, curveSpeed, cruiseSpeed, accel, rho, gravity)
	energySaved := coastConservation(cruiseE, bottomE, distance)
	return energySaved - energyUsed
}