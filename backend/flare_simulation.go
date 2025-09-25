package main

import "fmt"


func main() {
	/*solarYield := 0.0
	maxSpeed := 50.0
	maxGforce := 0.5 */
	battCharge := 100.0 

	straight_path := Segment{Length: 100}
	straight_path1 := Segment{Length: 100}
	straight_path2 := Segment{Length: 100}
	straight_path3 := Segment{Radius: 90, Angle: 90}

	race_track := Track{Segments: []Segment{straight_path, straight_path1, straight_path2, straight_path3}}

	totalLength := 0.0
	for i := 0; i < len(race_track.Segments); i++ {
		totalLength += race_track.Segments[i].getArcLength();
	}

	for i := 1.0; i <= totalLength; i++ {
		fmt.Print(i);
		fmt.Print(": ")
		fmt.Print(battCharge / totalLength)
		fmt.Println()
	}
}
