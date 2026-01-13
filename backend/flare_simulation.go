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
	// --- NCM Motorsports Park ---

	seg0_s := Segment{Length: 637.1769912}

	seg1_c := Segment{Radius: 15.81415929, Angle: -17.4}
	seg2_c := Segment{Radius: 12.30088496, Angle: -19.79}
	seg3_c := Segment{Radius: 28.07079646, Angle: -10.89}
	seg4_c := Segment{Radius: 21.88495575, Angle: 7.45}
	seg5_c := Segment{Radius: 15.18584071, Angle: 22.74}
	seg6_c := Segment{Radius: 14.7699115, Angle: 8.72}
	seg7_c := Segment{Radius: 15.96460177, Angle: 23.36}

	seg8_c := Segment{Radius: 0, Angle: 12.76}
	seg9_s := Segment{Length: 100.9646018}

	seg10_c := Segment{Radius: 39.30973451, Angle: -23.95}

	seg11_c := Segment{Radius: 0, Angle: -16.92}
	seg12_s := Segment{Length: 138.460177}

	seg13_c := Segment{Radius: 9.159292035, Angle: -21.3}
	seg14_c := Segment{Radius: 11.30088496, Angle: -23.08}
	seg15_c := Segment{Radius: 11.49557522, Angle: -33.67}
	seg16_c := Segment{Radius: 24.34513274, Angle: -14.11}
	seg17_c := Segment{Radius: 22.22123894, Angle: -4.36}
	seg18_c := Segment{Radius: 20.59292035, Angle: -13.53}
	seg19_c := Segment{Radius: 23.53097345, Angle: -14.95}

	seg20_c := Segment{Radius: 0, Angle: -19.1}
	seg21_s := Segment{Length: 260.2920354}

	seg22_c := Segment{Radius: 31.74336283, Angle: -7.41}

	seg23_c := Segment{Radius: 0, Angle: -9.15}
	seg24_s := Segment{Length: 55.48672566}

	seg25_c := Segment{Radius: 0, Angle: 1.93}
	seg26_s := Segment{Length: 73.62831858}

	seg27_c := Segment{Radius: 0, Angle: 8.65}
	seg28_s := Segment{Length: 65.7079646}

	seg29_c := Segment{Radius: 0, Angle: 10.82}
	seg30_s := Segment{Length: 86.44247788}

	seg31_c := Segment{Radius: 0, Angle: -8.69}
	seg32_s := Segment{Length: 71.31858407}

	seg33_c := Segment{Radius: 49.55752212, Angle: -19.78}

	seg34_c := Segment{Radius: 0, Angle: -18.02}
	seg35_s := Segment{Length: 53.30973451}

	seg36_c := Segment{Radius: 37.69026549, Angle: -17.61}
	seg37_c := Segment{Radius: 36.83185841, Angle: -7.12}
	seg38_c := Segment{Radius: 49.0619469, Angle: -9.23}
	seg39_c := Segment{Radius: 45.02654867, Angle: -1.86}
	seg40_c := Segment{Radius: 17.40707965, Angle: -37.78}
	seg41_c := Segment{Radius: 13.51327434, Angle: -25.04}
	seg42_c := Segment{Radius: 18.87610619, Angle: -13.24}
	seg43_c := Segment{Radius: 18.27433628, Angle: -27.45}
	seg44_c := Segment{Radius: 32.94690265, Angle: -6.38}
	seg45_c := Segment{Radius: 30.43362832, Angle: -15.07}
	seg46_c := Segment{Radius: 39.66371681, Angle: -4.52}
	seg47_c := Segment{Radius: 20.11504425, Angle: -33.89}
	seg48_c := Segment{Radius: 30.85840708, Angle: -22.64}

	seg49_c := Segment{Radius: 0, Angle: -2.53}
	seg50_s := Segment{Length: 64.62831858}

	seg51_c := Segment{Radius: 27.54867257, Angle: 30.49}
	seg52_c := Segment{Radius: 18.34513274, Angle: 20.1}
	seg53_c := Segment{Radius: 18.31858407, Angle: 17.54}
	seg54_c := Segment{Radius: 21.84070796, Angle: 12.57}
	seg55_c := Segment{Radius: 24.2920354, Angle: 19.01}
	seg56_c := Segment{Radius: 29.01769912, Angle: 11.44}
	seg57_c := Segment{Radius: 27.24778761, Angle: -17.92}
	seg58_c := Segment{Radius: 34.92035398, Angle: -16.36}
	seg59_c := Segment{Radius: 32.80530973, Angle: -8.94}

	seg60_c := Segment{Radius: 0, Angle: -10.59}
	seg61_s := Segment{Length: 72.51327434}

	seg62_c := Segment{Radius: 37.20353982, Angle: 9.27}
	seg63_c := Segment{Radius: 27.15044248, Angle: 11.04}
	seg64_c := Segment{Radius: 41.02654867, Angle: 7.59}
	seg65_c := Segment{Radius: 26.7699115, Angle: 26.76}
	seg66_c := Segment{Radius: 35.66371681, Angle: 16.22}
	seg67_c := Segment{Radius: 14.52212389, Angle: 60.88}
	seg68_c := Segment{Radius: 21.21238938, Angle: 35.95}

	seg69_c := Segment{Radius: 0, Angle: 18.81}
	seg70_s := Segment{Length: 87.87610619}

	seg71_c := Segment{Radius: 32.60176991, Angle: 11.02}

	seg72_c := Segment{Radius: 0, Angle: 5.8}
	seg73_s := Segment{Length: 66.18584071}

	seg74_c := Segment{Radius: 13.99115044, Angle: -31.79}

	seg75_c := Segment{Radius: 0, Angle: -16.18}
	seg76_s := Segment{Length: 116.1415929}

	seg77_c := Segment{Radius: 11.84955752, Angle: 13.52}

	seg78_c := Segment{Radius: 0, Angle: 17.62}
	seg79_s := Segment{Length: 192.8318584}

	seg80_c := Segment{Radius: 41.42477876, Angle: 3.19}

	seg81_c := Segment{Radius: 0, Angle: 12.45}
	seg82_s := Segment{Length: 79.50442478}

	seg83_c := Segment{Radius: 0, Angle: 3.09}
	seg84_s := Segment{Length: 228.2654867}

	seg85_c := Segment{Radius: 34.05309735, Angle: 10.94}

	seg86_c := Segment{Radius: 0, Angle: 8.92}
	seg87_s := Segment{Length: 98.88495575}

	seg88_c := Segment{Radius: 36.42477876, Angle: -17.64}
	seg89_c := Segment{Radius: 48.53982301, Angle: -24.99}
	seg90_c := Segment{Radius: 42.49557522, Angle: -14.52}
	seg91_c := Segment{Radius: 49.87610619, Angle: -13.58}
	seg92_c := Segment{Radius: 49.80530973, Angle: -15.14}
	seg93_c := Segment{Radius: 43.71681416, Angle: -8.58}
	seg94_c := Segment{Radius: 17.61946903, Angle: -24.65}
	seg95_c := Segment{Radius: 26.98230088, Angle: -23.13}
	seg96_c := Segment{Radius: 36.26548673, Angle: -12.81}

	seg97_c := Segment{Radius: 0, Angle: -10.92}
	seg98_s := Segment{Length: 72.16814159}

	seg99_c := Segment{Radius: 0, Angle: -6.33}
	seg100_s := Segment{Length: 57.46017699}

	seg101_c := Segment{Radius: 27.45132743, Angle: -33.22}
	seg102_c := Segment{Radius: 32.04424779, Angle: -28.76}
	seg103_c := Segment{Radius: 27.46902655, Angle: -32.22}
	seg104_c := Segment{Radius: 23.44247788, Angle: -24.13}
	seg105_c := Segment{Radius: 19.38053097, Angle: -18.12}
	seg106_c := Segment{Radius: 17.68141593, Angle: -26.56}

	seg107_c := Segment{Radius: 0, Angle: -29.07}
	seg108_s := Segment{Length: 67.75221239}

	seg109_c := Segment{Radius: 29.53097345, Angle: 33.99}
	seg110_c := Segment{Radius: 21.66371681, Angle: 23.77}
	seg111_c := Segment{Radius: 28.96460177, Angle: 27.99}
	seg112_c := Segment{Radius: 28.15044248, Angle: 21.02}
	seg113_c := Segment{Radius: 24.13274336, Angle: 33.61}
	seg114_c := Segment{Radius: 22.91150442, Angle: 25.26}

	seg115_c := Segment{Radius: 0, Angle: 37.86}
	seg116_s := Segment{Length: 56.21238938}

	seg117_c := Segment{Radius: 17.45132743, Angle: -12.67}
	seg118_c := Segment{Radius: 14.99115044, Angle: -25.63}

	seg119_c := Segment{Radius: 0, Angle: -9.99}
	seg120_s := Segment{Length: 53.59292035}

	seg121_c := Segment{Radius: 19.89380531, Angle: 9.51}

	seg122_c := Segment{Radius: 0, Angle: 29.52}
	seg123_s := Segment{Length: 75.7699115}

	seg124_c := Segment{Radius: 31.82300885, Angle: -16.17}
	seg125_c := Segment{Radius: 22.95575221, Angle: -10.85}
	NCM_Motorsports_Park := Track{
		Segments: []Segment{
			seg0_s,
			seg1_c, seg2_c, seg3_c, seg4_c, seg5_c, seg6_c, seg7_c,
			seg8_c, seg9_s, seg10_c,
			seg11_c, seg12_s,
			seg13_c, seg14_c, seg15_c, seg16_c, seg17_c, seg18_c, seg19_c,
			seg20_c, seg21_s,
			seg22_c,
			seg23_c, seg24_s,
			seg25_c, seg26_s,
			seg27_c, seg28_s,
			seg29_c, seg30_s,
			seg31_c, seg32_s,
			seg33_c,
			seg34_c, seg35_s,
			seg36_c, seg37_c, seg38_c, seg39_c, seg40_c, seg41_c, seg42_c,
			seg43_c, seg44_c, seg45_c, seg46_c, seg47_c, seg48_c,
			seg49_c, seg50_s,
			seg51_c, seg52_c, seg53_c, seg54_c, seg55_c, seg56_c, seg57_c,
			seg58_c, seg59_c,
			seg60_c, seg61_s,
			seg62_c, seg63_c, seg64_c, seg65_c, seg66_c, seg67_c, seg68_c,
			seg69_c, seg70_s,
			seg71_c,
			seg72_c, seg73_s,
			seg74_c,
			seg75_c, seg76_s,
			seg77_c,
			seg78_c, seg79_s,
			seg80_c,
			seg81_c, seg82_s,
			seg83_c, seg84_s,
			seg85_c,
			seg86_c, seg87_s,
			seg88_c, seg89_c, seg90_c, seg91_c, seg92_c, seg93_c, seg94_c,
			seg95_c, seg96_c,
			seg97_c, seg98_s,
			seg99_c, seg100_s,
			seg101_c, seg102_c, seg103_c, seg104_c, seg105_c, seg106_c,
			seg107_c, seg108_s,
			seg109_c, seg110_c, seg111_c, seg112_c, seg113_c, seg114_c,
			seg115_c, seg116_s,
			seg117_c, seg118_c,
			seg119_c, seg120_s,
			seg121_c,
			seg122_c, seg123_s,
			seg124_c, seg125_c,
		},
	}

	fullBatt := batteryWh
	battWithLosses := fullBatt

	//csv file tracking distance and speed + battery
	//reset info arleady there
	ClearStepStatstoCSV("Best Velocity (km/h), Distance (m), Battery (Wh)")

	//course sweep

	//first (OG Battery eg. 1000) -> iteration overshoot -> subtract high energy losses from orignal battery reserve (optimistic)
	//second -> use the difference from the first iteration as the battery reserve (pessimistic) then subtract energy losses from OG battery (1000) information
	//third -> use the difference of the last one (more optimistic) ...
	//until speed converges at a certain speed

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
		// This is a finer search in a narrow window around the previously found best speed
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
		//amt energy lost in lap
		cruiseE := PowerRequired(bestV, m, g, Crr, rho, Cd, A, theta)
		for j := 0; j < len(NCM_Motorsports_Park.Segments)-1; j++ {
			if NCM_Motorsports_Park.Segments[j].Radius != 0 {
				lapLoss += float64(netCurveLosses(m, A, Cd, Crr, NCM_Motorsports_Park.Segments[j+1], bestV, 0.5, rho, g, cruiseE, 0.006, 10, 0.8)) // MAKE A FUNCTION TO CHECK IF THE NEXT SEGMENT IS A CURVE
				fmt.Println(lapLoss)
				fmt.Println("total loss in segment ", j, ": ", lapLoss)
			}
		}

		fmt.Println(numLaps)
		fmt.Println("-----------------")
		battWithLosses = fullBatt - lapLoss*numLaps
		WriteStepStatstoCSV(bestV, math.Round(bestD), battWithLosses)
	}

	fmt.Println(getTotalLength(NCM_Motorsports_Park))
}
