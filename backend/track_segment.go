package main

import (
	"math"
)

type Segment struct {
	Length float64
	Radius float64
	Angle  float64

	//try to make it so that the segment appends to the track
	//appendSegment()
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
	Length   float64
}

// WIP
func appendSegment(s Segment, t Track) {
	t.Segments = append(t.Segments, s)
}

func getTotalLength(t Track) float64 {

	for _, seg := range t.Segments {
		t.Length += seg.getArcLength()
	}
	return t.Length

}
