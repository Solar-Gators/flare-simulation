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
	targetSpeed := CalcGforce(segment, gravity, gmax)

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
func coastConservation(coastEnergy, bottomEnergy, distance float64) float64 {
	energyConserved := distance * (coastEnergy - bottomEnergy)
	return energyConserved
}