package main

import "fmt"

func main() {
	/*solarYield := 0
	maxSpeed := 50
	maxGforce := 0.5 */
	battCharge := 100 

	straight_path := Segment{Length: 100, Radius: 0}
	straight_path1 := Segment{Length: 100, Radius: 0}
	straight_path2 := Segment{Length: 100, Radius: 0}

	race_track := Track{Segments: []Segment{straight_path, straight_path1, straight_path2}}

	totalLength := 0
	for i := 0; i < len(race_track.Segments); i++ {
		if (race_track.Segments[i].Radius != 0) {
			//call arc length
		}
		totalLength += race_track.Segments[i].Length
	}

	for i := 1; i <= totalLength; i++ {
		fmt.Print(i);
		fmt.Print(": ")
		fmt.Print(float64(battCharge) / float64(totalLength))
		fmt.Println()
	}
}
