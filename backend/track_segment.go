package main

/*
OOP Section
**Track HAS-A Segment**
*/

import (
	"math"
)

// segement class with attributes
type Segment struct {
	Length float64 // used if angle == 0
	Radius float64 // used if length == 0
	Angle  float64 // as per my understanding, angle is relative to the previous segment
	// the angle described here is a "header" for direction from the current position

	//try to make it so that the segment appends to the track
	//appendSegment()
}

// calculate arc length if segment has radius
func (s Segment) getArcLength() float64 {
	if s.Radius != 0 {
		return math.Abs(s.Radius * s.Angle * math.Pi / 180)
	} else {
		return s.Length
	}
}

// Has-A relationship
type Track struct {
	Segments []Segment
	Length   float64
}

// more member functions
func appendSegment(s Segment, t Track) {
	t.Segments = append(t.Segments, s)
}

func getTotalLength(t Track) float64 {

	for _, seg := range t.Segments {
		t.Length += seg.getArcLength()
	}
	return t.Length

}
