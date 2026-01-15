package main

import (
	"fmt"
	"math"
	"time"
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
	// spanStart := time.Now().Truncate(time.Hour)
	// spanEnd := spanStart.Add(8 * time.Hour)

	loc, _ := time.LoadLocation("America/New_York")
	now := time.Now().In(loc)

	spanStart := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
	spanEnd   := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, loc)

	totalEnergyGained, fullBatt, err := BuildEnergyWithBattery(
	29.6516, -82.3248,        // lat, lon
	5.0, 0.0,                 // tilt, azimuth
	"America/New_York",       // timezone
	1.0,                        // forecastDays (1 day is enough for 8 hours)

	4.0,                      // panelArea (m²)
	0.22,                     // panelEff
	0.9,                      // systemEff

	time.Hour,                // dt
	batteryWh,                // initialBatteryWh

	&spanStart,               // start time
	&spanEnd,                 // end time
	)
	if err != nil {
		panic(err)
	}	

	fmt.Println("Total Energy Gained: ", totalEnergyGained, "Fullbatt after gaines: ", fullBatt)

	var battWithLosses float64 = fullBatt

	//csv file tracking distance and speed + battery
	ClearStepStatstoCSV()
	//course sweep
	for n := 0; n < 7; n += 1 {
		var lapLoss float64= 0.0
		var numLaps float64 = 0.0
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
		 
		//find total losses by taking net losses of all curves (given the calculated laps)
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
		// sets up next iteration battery (with losses) 
		battWithLosses = fullBatt - lapLoss * numLaps
		WriteStepStatstoCSV(bestV, math.Round(bestD), battWithLosses)
	}
}

func BuildEnergySeriesWithBattery(f1, f2, f3, f4 float64, s string, i int, f5, f6, f7 float64, duration time.Duration, batteryWh float64, time1 *time.Time, time2 *time.Time) (any, any, any) {
	panic("unimplemented")
}

// 1st iteration takes a full battery --> a track with no curves
// finds best velocity by iterating through speeds
// finds distance given that velocity
// uses total distance to find number of laps
// calculates the number of energy losses given the number of laps (because curves --> energy loss)
// recalculates battery for next iteration (flucating between overestimation and underestimation)
// only product we want is the velocity at the end of 7th iteration


