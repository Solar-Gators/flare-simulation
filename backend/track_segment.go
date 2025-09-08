package main

type Segment struct {
	Length int
	Radius int
}

type Track struct {
	Segments []Segment
}