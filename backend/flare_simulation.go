package main

import "fmt"


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
			curveSpeed := CalcGforce(upcomingCurve, gravity, gmax)
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
			netE := netCurveLosses(mass, fArea, aDrag, rRes, curveSpeed, speed, accel, rho, gravity, cruiseWhPerM, bottomE, DistanceToCoast)
			fmt.Println("Net loss of energy is: ", netE, " wh")
			}
			fmt.Println("------------------")
		} 
	}
}
func main() {
	/*solarYield := 0.0
	maxSpeed := 50.0
	maxGforce := 0.5 */
	// battCharge := 100.0 
	//time 19.22 for day 1
	straight_path := Segment{Length: 100}
	straight_path1 := Segment{Length: 100}
	straight_path2 := Segment{Length: 100}
	curved_path := Segment{Radius: 90, Angle: 90}
	straight_path3 := Segment{Length: 100}
	curved_path2 := Segment{Radius: 90, Angle: 90}

	race_track := Track{Segments: []Segment{straight_path, straight_path1, straight_path2, curved_path, straight_path3, curved_path2}}

	totalLength := 0.0
	for i := 0; i < len(race_track.Segments); i++ {
		totalLength += race_track.Segments[i].getArcLength()
	}
	//pass in hardcoded track and simulate coast
	simulateCoast(race_track)
}


