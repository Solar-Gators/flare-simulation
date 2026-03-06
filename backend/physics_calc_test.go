package main

import (
	"math"
	"testing"
)

func TestEnforceBrakeLookaheadTightensCommand(t *testing.T) {
	got := enforceBrakeLookahead(-0.4, 20, 10, 50, 5)
	want := -3.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}

func TestEnforceBrakeLookaheadRespectsBrakeLimit(t *testing.T) {
	got := enforceBrakeLookahead(-0.2, 25, 5, 20, 4)
	want := -4.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}

func TestEnforceBrakeLookaheadKeepsStrongerExistingBrake(t *testing.T) {
	got := enforceBrakeLookahead(-3.5, 20, 10, 50, 5)
	want := -3.5
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}

func TestSharpestApexWithinHorizonPrefersFartherTighterCorner(t *testing.T) {
	targets := []speedConstraint{
		{dist: 50, vLimit: 20},
		{dist: 150, vLimit: 10},
	}

	vLimit, dist, ok := sharpestApexWithinHorizon(targets, 0, 0, 200, false)
	if !ok {
		t.Fatal("expected an apex target")
	}
	if math.Abs(vLimit-10) > 1e-9 || math.Abs(dist-150) > 1e-9 {
		t.Fatalf("got vLimit=%.6f dist=%.6f", vLimit, dist)
	}
}

func TestBuildApexTargetsUsesCurveMidpoints(t *testing.T) {
	segments := []trackSegment{
		{Type: "straight", Length: 40},
		{Type: "curve", Radius: 20, Angle: 90},
	}

	targets := buildApexTargets(segments, 1.0, 9.81)
	if len(targets) != 1 {
		t.Fatalf("got %d targets, want 1", len(targets))
	}

	curveLen := segLengthM(segments[1])
	wantDist := 40 + 0.5*curveLen
	if math.Abs(targets[0].dist-wantDist) > 1e-9 {
		t.Fatalf("got dist=%.6f, want %.6f", targets[0].dist, wantDist)
	}
}
