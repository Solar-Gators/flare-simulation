package main

import (
	"fmt"
	"math"
)

func main() {
	/*solarYield := 0.0
	maxSpeed := 50.0
	maxGforce := 0.5 */
	//battCharge := 100.0
	//time 19.22 for day 1
	straight_path := Segment{Length: 100}
	straight_path1 := Segment{Length: 100}
	straight_path2 := Segment{Length: 100}
	curved_path := Segment{Radius: 90, Angle: 90}

	race_track := Track{Segments: []Segment{straight_path, straight_path1, straight_path2, curved_path}}

	totalLength := 0.0
	for i := 0; i < len(race_track.Segments); i++ {
		totalLength += race_track.Segments[i].getArcLength()
	}
	ClearStepStatstoCSV()
	for i := 0.1; i <= 25.0; i = i + 0.1 {
		d, _ := TestTotalDistanceEV(5.0, 480.0, 5000.0, .92, 0.2792, 45.0, 10000.0, 285.0, 9.81, 0.0015, 1.225, 0.21, 0.456, 0, i)
		fmt.Println(math.Round(d))
		WriteStepStatstoCSV(i, math.Round(d), 0.0)
	}
}
