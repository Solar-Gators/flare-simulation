package main

import "fmt"

func main() {
	var solarYield int
	var maxSpeed int
	var maxGforce int

	fmt.Println("Enter the current solar panel yield (energy/min): ")
	fmt.Scan(&solarYield)

	fmt.Println("Enter the target max speed: ")
	fmt.Scan(&maxSpeed)

	fmt.Println("Enter max g-force allowed: ")
	fmt.Scan(&maxGforce)

	straight_path := Segment{Length: 50, IsCurve: false, Radius: 0}

	race_track := Track{Segments: []Segment{straight_path}}
}
