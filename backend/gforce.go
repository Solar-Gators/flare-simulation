package main

import "math"

func CalcGforce(segments []Segment) []float64{
	gravity := 9.81
	gmax := 0.7

	curved_speed := []float64{}
	
	for i := 0; i < len(segments); i++ {
		if (segments[i].Radius > 0) {
			radius := segments[i].Radius

			maxVelocity := math.Sqrt(gmax * radius * gravity)

			curved_speed = append(curved_speed, maxVelocity)
		}
	}

	return curved_speed
}