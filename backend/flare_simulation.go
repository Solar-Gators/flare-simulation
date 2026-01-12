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
	seg23_c := Segment{Radius: 55.48672566, Angle: -9.15}
	seg24_c := Segment{Radius: 73.62831858, Angle: 1.93}
	seg25_c := Segment{Radius: 65.7079646, Angle: 8.65}
	seg26_c := Segment{Radius: 86.44247788, Angle: 10.82}
	seg27_c := Segment{Radius: 71.31858407, Angle: -8.69}
	seg28_c := Segment{Radius: 49.55752212, Angle: -19.78}
	seg29_c := Segment{Radius: 53.30973451, Angle: -18.02}
	seg30_c := Segment{Radius: 37.69026549, Angle: -17.61}
	seg31_c := Segment{Radius: 36.83185841, Angle: -7.12}
	seg32_c := Segment{Radius: 49.0619469, Angle: -9.23}
	seg33_c := Segment{Radius: 45.02654867, Angle: -1.86}
	seg34_c := Segment{Radius: 17.40707965, Angle: -37.78}
	seg35_c := Segment{Radius: 13.51327434, Angle: -25.04}
	seg36_c := Segment{Radius: 18.87610619, Angle: -13.24}
	seg37_c := Segment{Radius: 18.27433628, Angle: -27.45}
	seg38_c := Segment{Radius: 32.94690265, Angle: -6.38}
	seg39_c := Segment{Radius: 30.43362832, Angle: -15.07}
	seg40_c := Segment{Radius: 39.66371681, Angle: -4.52}
	seg41_c := Segment{Radius: 20.11504425, Angle: -33.89}
	seg42_c := Segment{Radius: 30.85840708, Angle: -22.64}
	seg43_c := Segment{Radius: 64.62831858, Angle: -2.53}
	seg44_c := Segment{Radius: 27.54867257, Angle: 30.49}
	seg45_c := Segment{Radius: 18.34513274, Angle: 20.1}
	seg46_c := Segment{Radius: 18.31858407, Angle: 17.54}
	seg47_c := Segment{Radius: 21.84070796, Angle: 12.57}
	seg48_c := Segment{Radius: 24.2920354, Angle: 19.01}
	seg49_c := Segment{Radius: 29.01769912, Angle: 11.44}
	seg50_c := Segment{Radius: 27.24778761, Angle: -17.92}
	seg51_c := Segment{Radius: 34.92035398, Angle: -16.36}
	seg52_c := Segment{Radius: 32.80530973, Angle: -8.94}
	seg53_c := Segment{Radius: 72.51327434, Angle: -10.59}
	seg54_c := Segment{Radius: 37.20353982, Angle: 9.27}
	seg55_c := Segment{Radius: 27.15044248, Angle: 11.04}
	seg56_c := Segment{Radius: 41.02654867, Angle: 7.59}
	seg57_c := Segment{Radius: 26.7699115, Angle: 26.76}
	seg58_c := Segment{Radius: 35.66371681, Angle: 16.22}
	seg59_c := Segment{Radius: 14.52212389, Angle: 60.88}
	seg60_c := Segment{Radius: 21.21238938, Angle: 35.95}
	seg61_c := Segment{Radius: 87.87610619, Angle: 18.81}
	seg62_c := Segment{Radius: 32.60176991, Angle: 11.02}
	seg63_c := Segment{Radius: 66.18584071, Angle: 5.8}
	seg64_c := Segment{Radius: 13.99115044, Angle: -31.79}

	seg65_c := Segment{Radius: 0, Angle: -16.18}
	seg66_s := Segment{Length: 116.1415929}

	seg67_c := Segment{Radius: 11.84955752, Angle: 13.52}

	seg68_c := Segment{Radius: 0, Angle: 17.62}
	seg69_s := Segment{Length: 192.8318584}

	seg70_c := Segment{Radius: 41.42477876, Angle: 3.19}
	seg71_c := Segment{Radius: 79.50442478, Angle: 12.45}

	seg72_c := Segment{Radius: 0, Angle: 3.09}
	seg73_s := Segment{Length: 228.2654867}

	seg74_c := Segment{Radius: 34.05309735, Angle: 10.94}
	seg75_c := Segment{Radius: 98.88495575, Angle: 8.92}
	seg76_c := Segment{Radius: 36.42477876, Angle: -17.64}
	seg77_c := Segment{Radius: 48.53982301, Angle: -24.99}
	seg78_c := Segment{Radius: 42.49557522, Angle: -14.52}
	seg79_c := Segment{Radius: 49.87610619, Angle: -13.58}
	seg80_c := Segment{Radius: 49.80530973, Angle: -15.14}
	seg81_c := Segment{Radius: 43.71681416, Angle: -8.58}
	seg82_c := Segment{Radius: 17.61946903, Angle: -24.65}
	seg83_c := Segment{Radius: 26.98230088, Angle: -23.13}
	seg84_c := Segment{Radius: 36.26548673, Angle: -12.81}
	seg85_c := Segment{Radius: 72.16814159, Angle: -10.92}
	seg86_c := Segment{Radius: 57.46017699, Angle: -6.33}
	seg87_c := Segment{Radius: 27.45132743, Angle: -33.22}
	seg88_c := Segment{Radius: 32.04424779, Angle: -28.76}
	seg89_c := Segment{Radius: 27.46902655, Angle: -32.22}
	seg90_c := Segment{Radius: 23.44247788, Angle: -24.13}
	seg91_c := Segment{Radius: 19.38053097, Angle: -18.12}
	seg92_c := Segment{Radius: 17.68141593, Angle: -26.56}
	seg93_c := Segment{Radius: 67.75221239, Angle: -29.07}
	seg94_c := Segment{Radius: 29.53097345, Angle: 33.99}
	seg95_c := Segment{Radius: 21.66371681, Angle: 23.77}
	seg96_c := Segment{Radius: 28.96460177, Angle: 27.99}
	seg97_c := Segment{Radius: 28.15044248, Angle: 21.02}
	seg98_c := Segment{Radius: 24.13274336, Angle: 33.61}
	seg99_c := Segment{Radius: 22.91150442, Angle: 25.26}
	seg100_c := Segment{Radius: 56.21238938, Angle: 37.86}
	seg101_c := Segment{Radius: 17.45132743, Angle: -12.67}
	seg102_c := Segment{Radius: 14.99115044, Angle: -25.63}
	seg103_c := Segment{Radius: 53.59292035, Angle: -9.99}
	seg104_c := Segment{Radius: 19.89380531, Angle: 9.51}
	seg105_c := Segment{Radius: 75.7699115, Angle: 29.52}
	seg106_c := Segment{Radius: 31.82300885, Angle: -16.17}
	seg107_c := Segment{Radius: 22.95575221, Angle: -10.85}

	NCM_Motorsports_Park := Track{
		Segments: []Segment{
			seg0_s,
			seg1_c, seg2_c, seg3_c, seg4_c, seg5_c, seg6_c, seg7_c,
			seg8_c, seg9_s, seg10_c,
			seg11_c, seg12_s,
			seg13_c, seg14_c, seg15_c, seg16_c, seg17_c, seg18_c, seg19_c,
			seg20_c, seg21_s,
			seg22_c, seg23_c, seg24_c, seg25_c, seg26_c, seg27_c, seg28_c,
			seg29_c, seg30_c, seg31_c, seg32_c, seg33_c, seg34_c, seg35_c,
			seg36_c, seg37_c, seg38_c, seg39_c, seg40_c, seg41_c, seg42_c,
			seg43_c, seg44_c, seg45_c, seg46_c, seg47_c, seg48_c, seg49_c,
			seg50_c, seg51_c, seg52_c, seg53_c, seg54_c, seg55_c, seg56_c,
			seg57_c, seg58_c, seg59_c, seg60_c, seg61_c, seg62_c, seg63_c,
			seg64_c,
			seg65_c, seg66_s,
			seg67_c,
			seg68_c, seg69_s,
			seg70_c, seg71_c,
			seg72_c, seg73_s,
			seg74_c, seg75_c, seg76_c, seg77_c, seg78_c, seg79_c, seg80_c,
			seg81_c, seg82_c, seg83_c, seg84_c, seg85_c, seg86_c, seg87_c,
			seg88_c, seg89_c, seg90_c, seg91_c, seg92_c, seg93_c, seg94_c,
			seg95_c, seg96_c, seg97_c, seg98_c, seg99_c, seg100_c, seg101_c,
			seg102_c, seg103_c, seg104_c, seg105_c, seg106_c, seg107_c,
		},
	}

	// seg0_s := Segment{Length: 637.1769912}

	// seg1_c := Segment{Radius: 52.05, Angle: -17.4}
	// seg2_c := Segment{Radius: 35.63, Angle: -19.79}
	// seg3_c := Segment{Radius: 147.60, Angle: -10.89}
	// seg4_c := Segment{Radius: 168.25, Angle: 7.45}
	// seg5_c := Segment{Radius: 38.23, Angle: 22.74}
	// seg6_c := Segment{Radius: 97.00, Angle: 8.72}
	// seg7_c := Segment{Radius: 39.14, Angle: 23.36}
	// seg8_c := Segment{Radius: 452.68, Angle: 12.76}
	// seg9_c := Segment{Radius: 94.05, Angle: -23.95}
	// seg10_c := Segment{Radius: 468.46, Angle: -16.92}
	// seg11_c := Segment{Radius: 24.64, Angle: -21.3}
	// seg12_c := Segment{Radius: 28.02, Angle: -23.08}
	// seg13_c := Segment{Radius: 19.55, Angle: -33.67}
	// seg14_c := Segment{Radius: 98.93, Angle: -14.11}
	// seg15_c := Segment{Radius: 292.02, Angle: -4.36}
	// seg16_c := Segment{Radius: 87.16, Angle: -13.53}
	// seg17_c := Segment{Radius: 90.25, Angle: -14.95}
	// seg18_c := Segment{Radius: 781.47, Angle: -19.1}
	// seg19_c := Segment{Radius: 245.16, Angle: -7.41}
	// seg20_c := Segment{Radius: 346.89, Angle: -9.15}
	// seg21_c := Segment{Radius: 2187.60, Angle: 1.93}
	// seg22_c := Segment{Radius: 435.43, Angle: 8.65}
	// seg23_c := Segment{Radius: 458.44, Angle: 10.82}
	// seg24_c := Segment{Radius: 469.46, Angle: -8.69}
	// seg25_c := Segment{Radius: 143.55, Angle: -19.78}
	// seg26_c := Segment{Radius: 169.41, Angle: -18.02}
	// seg27_c := Segment{Radius: 122.61, Angle: -17.61}
	// seg28_c := Segment{Radius: 296.16, Angle: -7.12}
	// seg29_c := Segment{Radius: 304.23, Angle: -9.23}
	// seg30_c := Segment{Radius: 1386.79, Angle: -1.86}
	// seg31_c := Segment{Radius: 26.40, Angle: -37.78}
	// seg32_c := Segment{Radius: 30.92, Angle: -25.04}
	// seg33_c := Segment{Radius: 81.61, Angle: -13.24}
	// seg34_c := Segment{Radius: 38.11, Angle: -27.45}
	// seg35_c := Segment{Radius: 295.71, Angle: -6.38}
	// seg36_c := Segment{Radius: 115.64, Angle: -15.07}
	// seg37_c := Segment{Radius: 502.76, Angle: -4.52}
	// seg38_c := Segment{Radius: 33.95, Angle: -33.89}
	// seg39_c := Segment{Radius: 78.01, Angle: -22.64}
	// seg40_c := Segment{Radius: 1464.16, Angle: -2.53}
	// seg41_c := Segment{Radius: 51.87, Angle: 30.49}
	// seg42_c := Segment{Radius: 52.33, Angle: 20.1}
	// seg43_c := Segment{Radius: 59.78, Angle: 17.54}
	// seg44_c := Segment{Radius: 99.54, Angle: 12.57}
	// seg45_c := Segment{Radius: 73.24, Angle: 19.01}
	// seg46_c := Segment{Radius: 145.47, Angle: 11.44}
	// seg47_c := Segment{Radius: 87.09, Angle: -17.92}
	// seg48_c := Segment{Radius: 122.09, Angle: -16.36}
	// seg49_c := Segment{Radius: 210.02, Angle: -8.94}
	// seg50_c := Segment{Radius: 392.55, Angle: -10.59}
	// seg51_c := Segment{Radius: 229.97, Angle: 9.27}
	// seg52_c := Segment{Radius: 140.90, Angle: 11.04}
	// seg53_c := Segment{Radius: 309.89, Angle: 7.59}
	// seg54_c := Segment{Radius: 57.28, Angle: 26.76}
	// seg55_c := Segment{Radius: 125.97, Angle: 16.22}
	// seg56_c := Segment{Radius: 13.68, Angle: 60.88}
	// seg57_c := Segment{Radius: 33.81, Angle: 35.95}
	// seg58_c := Segment{Radius: 267.78, Angle: 18.81}
	// seg59_c := Segment{Radius: 169.54, Angle: 11.02}
	// seg60_c := Segment{Radius: 653.73, Angle: 5.8}
	// seg61_c := Segment{Radius: 25.19, Angle: -31.79}
	// seg62_c := Segment{Radius: 411.26, Angle: -16.18}
	// seg63_c := Segment{Radius: 50.18, Angle: 13.52}
	// seg64_c := Segment{Radius: 626.89, Angle: 17.62}
	// seg65_c := Segment{Radius: 744.16, Angle: 3.19}
	// seg66_c := Segment{Radius: 366.05, Angle: 12.45}
	// seg67_c := Segment{Radius: 4230.52, Angle: 3.09}
	// seg68_c := Segment{Radius: 178.18, Angle: 10.94}
	// seg69_c := Segment{Radius: 636.14, Angle: 8.92}
	// seg70_c := Segment{Radius: 118.19, Angle: -17.64}
	// seg71_c := Segment{Radius: 111.32, Angle: -24.99}
	// seg72_c := Segment{Radius: 167.79, Angle: -14.52}
	// seg73_c := Segment{Radius: 210.49, Angle: -13.58}
	// seg74_c := Segment{Radius: 188.48, Angle: -15.14}
	// seg75_c := Segment{Radius: 292.09, Angle: -8.58}
	// seg76_c := Segment{Radius: 40.92, Angle: -24.65}
	// seg77_c := Segment{Radius: 66.83, Angle: -23.13}
	// seg78_c := Segment{Radius: 162.24, Angle: -12.81}
	// seg79_c := Segment{Radius: 379.05, Angle: -10.92}
	// seg80_c := Segment{Radius: 520.17, Angle: -6.33}
	// seg81_c := Segment{Radius: 47.32, Angle: -33.22}
	// seg82_c := Segment{Radius: 63.82, Angle: -28.76}
	// seg83_c := Segment{Radius: 48.87, Angle: -32.22}
	// seg84_c := Segment{Radius: 55.63, Angle: -24.13}
	// seg85_c := Segment{Radius: 61.25, Angle: -18.12}
	// seg86_c := Segment{Radius: 38.18, Angle: -26.56}
	// seg87_c := Segment{Radius: 133.64, Angle: -29.07}
	// seg88_c := Segment{Radius: 49.77, Angle: 33.99}
	// seg89_c := Segment{Radius: 52.21, Angle: 23.77}
	// seg90_c := Segment{Radius: 59.30, Angle: 27.99}
	// seg91_c := Segment{Radius: 76.80, Angle: 21.02}
	// seg92_c := Segment{Radius: 41.13, Angle: 33.61}
	// seg93_c := Segment{Radius: 52.02, Angle: 25.26}
	// seg94_c := Segment{Radius: 84.99, Angle: 37.86}
	// seg95_c := Segment{Radius: 78.85, Angle: -12.67}
	// seg96_c := Segment{Radius: 33.51, Angle: -25.63}
	// seg97_c := Segment{Radius: 307.07, Angle: -9.99}
	// seg98_c := Segment{Radius: 119.85, Angle: 9.51}
	// seg99_c := Segment{Radius: 146.78, Angle: 29.52}
	// seg100_c := Segment{Radius: 112.69, Angle: -16.17}
	// seg101_c := Segment{Radius: 121.26, Angle: -10.85}

	// NCM_Motorsports_Park := Track{
	// 	Segments: []Segment{
	// 		seg0_s,
	// 		seg1_c, seg2_c, seg3_c, seg4_c, seg5_c, seg6_c, seg7_c, seg8_c,
	// 		seg9_c, seg10_c, seg11_c, seg12_c, seg13_c, seg14_c, seg15_c,
	// 		seg16_c, seg17_c, seg18_c, seg19_c, seg20_c, seg21_c, seg22_c,
	// 		seg23_c, seg24_c, seg25_c, seg26_c, seg27_c, seg28_c, seg29_c,
	// 		seg30_c, seg31_c, seg32_c, seg33_c, seg34_c, seg35_c, seg36_c,
	// 		seg37_c, seg38_c, seg39_c, seg40_c, seg41_c, seg42_c, seg43_c,
	// 		seg44_c, seg45_c, seg46_c, seg47_c, seg48_c, seg49_c, seg50_c,
	// 		seg51_c, seg52_c, seg53_c, seg54_c, seg55_c, seg56_c, seg57_c,
	// 		seg58_c, seg59_c, seg60_c, seg61_c, seg62_c, seg63_c, seg64_c,
	// 		seg65_c, seg66_c, seg67_c, seg68_c, seg69_c, seg70_c, seg71_c,
	// 		seg72_c, seg73_c, seg74_c, seg75_c, seg76_c, seg77_c, seg78_c,
	// 		seg79_c, seg80_c, seg81_c, seg82_c, seg83_c, seg84_c, seg85_c,
	// 		seg86_c, seg87_c, seg88_c, seg89_c, seg90_c, seg91_c, seg92_c,
	// 		seg93_c, seg94_c, seg95_c, seg96_c, seg97_c, seg98_c, seg99_c,
	// 		seg100_c, seg101_c,
	// 	},
	// }
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
