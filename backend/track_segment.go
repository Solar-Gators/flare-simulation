package main

import (
	"math"
)

type Segment struct {
	Length float64
	Radius float64
	Angle  float64
}

func (s Segment) getArcLength() float64 {
	if s.Radius != 0 {
		return s.Radius * s.Angle * math.Pi / 180
	} else {
		return s.Length
	}
}

type Track struct {
	Segments []Segment
	length   float64
}

func getTotalLength(t Track) float64 {

	for _, seg := range t.Segments {
		t.length += seg.getArcLength()
	}
	return t.length

}
