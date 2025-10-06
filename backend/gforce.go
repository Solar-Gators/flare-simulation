package main

import "math"

// Calculates the g force of current turn/curve
func CalcGforce(segments Segment, gravity float64, gmax float64) float64 {
	radius := segments.Radius
	maxVelocity := math.Sqrt(gmax * radius * gravity)
	return maxVelocity
}

// How much energy is used given a constant acceleration after a curve
func curveAccelEnergy(coastSpeed float64, currentSpeed float64, acceleration float64, distance float64) float64 {
	var mass float64 = 250
	energyUsed := 0.5 * mass * ((math.Pow(coastSpeed, 2)) - (math.Pow(currentSpeed, 2)))

	return energyUsed
}


//determines distance to let off accelerator before a curve to hit target speed
//taking in the same params as CalcGForce to be passed into it
func calcCoastDistance(
	currentSpeed float64, 
	segment Segment, 
	drag float64, 
	rollingRes float64, 
	gravity float64, 
	gmax float64, 
	mass float64, 
	frontalArea float64, 
	airDensity float64) float64 {

	targetSpeed := CalcGforce(segment, gravity, gmax)
	initSpeedSquared := math.Pow(currentSpeed, 2)
	endSpeedSquared := math.Pow(targetSpeed, 2)

	//quotient of coasting distance formula
	startTerm := (((airDensity*drag*frontalArea)/(2*mass)*initSpeedSquared + (rollingRes*gravity)))
	endTerm := (((airDensity*drag*frontalArea)/(2*mass)*endSpeedSquared + (rollingRes*gravity)))

	lnTerm := math.Log(startTerm/endTerm)

	coastDistance := ((mass/(airDensity*drag*frontalArea))*lnTerm)
	return coastDistance
}


// How much energy is saved by letting off the gas before a curve
func coastConservation(coastEnergy, bottomEnergy, distance float64) float64 {
	energyConserved := distance * (coastEnergy - bottomEnergy)
	return energyConserved
}