package main

import "fmt"

type Segment struct {
	length int;
	isCurve bool;
	radius int;
}

type Track struct {
	segment []Segment;
}