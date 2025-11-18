package main

import (
	"fmt"
	"math"
)

func main() {

	// --- Inputs from your data ---
	const (
		// Environment / vehicle
		A     = 0.456  // frontal area [m^2]
		Cd    = 0.21   // drag coefficient [-]
		rho   = 1.225  // air density [kg/m^3]
		Crr   = 0.0015 // rolling resistance coefficient [-]
		m     = 285.0  // mass [kg]
		g     = 9.81   // gravity [m/s^2]
		theta = 0.0    // road grade [rad], 0 = flat

		// EV capability
		rWheel = 0.2792  // wheel radius [m]
		Tmax   = 45.0    // max wheel torque [N·m]  (assumed at wheel)
		Pmax   = 10000.0 // max motor/inverter power [W] (pack-side cap)

		// Energy / time
		batteryWh     = 5000.0 // battery capacity [Wh]
		solarWhPerMin = 5.0    // solar generation [Wh/min] (avg)
		etaDrive      = 0.90   // battery→wheel efficiency [0..1]
		raceDayMin    = 480.0  // race duration [min] = 8 hours
	)
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

	t := Track{Segments: []Segment{straight_path, straight_path1, straight_path2, curved_path, straight_path3, curved_path2}}
	fullBatt := batteryWh
	battWithLosses := fullBatt

	//csv file tracking distance and speed + battery
	ClearStepStatstoCSV()
	//course sweep
	for n := 0; n < 7; n += 1 {
		lapLoss := 0.0
		numLaps := 0.0
		bestV, bestD := 0.0, 0.0
		//find best speed and distace (estimate)
		for v := 2.0; v <= 40.0; v += 0.5 {
			if d, ok := DistanceForSpeedEV(v, battWithLosses, solarWhPerMin, etaDrive, raceDayMin,
				rWheel, Tmax, Pmax, m, g, Crr, rho, Cd, A, theta); ok && d > bestD {
				bestD, bestV = d, v
			}
		}
		// refine around best
		for v := math.Max(0.5, bestV-2.0); v <= bestV+2.0; v += 0.1 {
			if d, ok := DistanceForSpeedEV(v, battWithLosses, solarWhPerMin, etaDrive, raceDayMin,
				rWheel, Tmax, Pmax, m, g, Crr, rho, Cd, A, theta); ok && d > bestD {
				bestD, bestV = d, v
				numLaps = d / 5070.0
			}
		}
		fmt.Println("Distance: ", bestD)
		fmt.Println("velocity: ", bestV)
		//find total losses
		cruiseE := PowerRequired(bestV, m, g, Crr, rho, Cd, A, theta)
		for j := 0; j < len(t.Segments)-1; j++ {
			if t.Segments[j].Radius != 0 {
				lapLoss += float64(netCurveLosses(m, A, Cd, Crr, t.Segments[j+1], bestV, 0.5, rho, g, cruiseE, 0.006, 10, 0.8)) // MAKE A FUNCTION TO CHECK IF THE NEXT SEGMENT IS A CURVE
				fmt.Println(lapLoss)
				fmt.Println("total loss: ", lapLoss)
			}
		}

		fmt.Println(numLaps)
		fmt.Println("-----------------")
		battWithLosses = fullBatt - lapLoss * numLaps
		WriteStepStatstoCSV(bestV, math.Round(bestD), battWithLosses)
	}
}


