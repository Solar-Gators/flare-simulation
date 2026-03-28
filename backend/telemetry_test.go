package main

import (
	"math"
	"testing"
)

func TestBuildTelemetryOneLapStartsAtProvidedSpeed(t *testing.T) {
	segments := []trackSegment{{Type: "straight", Length: 5}}

	got, err := buildTelemetryOneLap(segments, 7.25)
	if err != nil {
		t.Fatalf("buildTelemetryOneLap returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected telemetry points")
	}

	if math.Abs(got[0].Speed-7.25) > 1e-9 {
		t.Fatalf("got initial speed %.6f, want %.6f", got[0].Speed, 7.25)
	}
}

func TestBuildTelemetryOneLapRejectsNegativeStartSpeed(t *testing.T) {
	segments := []trackSegment{{Type: "straight", Length: 5}}

	if _, err := buildTelemetryOneLap(segments, -0.1); err == nil {
		t.Fatal("expected error for negative start speed")
	}
}

func TestWarmTelemetryStartSpeedUsesPreviousLapTerminalSpeed(t *testing.T) {
	segments := []trackSegment{{Type: "straight", Length: 5}}
	firstLap, err := buildTelemetryOneLap(segments, defaultTelemetryStartSpeed)
	if err != nil {
		t.Fatalf("buildTelemetryOneLap returned error: %v", err)
	}
	if len(firstLap) == 0 {
		t.Fatal("expected telemetry points")
	}

	want := firstLap[len(firstLap)-1].Speed
	got, err := warmTelemetryStartSpeed(segments, defaultTelemetryStartSpeed, 1, 0)
	if err != nil {
		t.Fatalf("warmTelemetryStartSpeed returned error: %v", err)
	}

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got warmed start speed %.6f, want %.6f", got, want)
	}
}

func TestBuildTelemetryStartsFromWarmedSpeed(t *testing.T) {
	segments := []trackSegment{{Type: "straight", Length: 5}}
	want, err := warmTelemetryStartSpeed(
		segments,
		defaultTelemetryStartSpeed,
		telemetryWarmupMaxLaps,
		telemetryWarmupTolerance,
	)
	if err != nil {
		t.Fatalf("warmTelemetryStartSpeed returned error: %v", err)
	}
	got, err := buildTelemetry(segments, true)
	if err != nil {
		t.Fatalf("buildTelemetry returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected telemetry points")
	}

	if math.Abs(got[0].Speed-want) > 1e-9 {
		t.Fatalf("got warmed initial speed %.6f, want %.6f", got[0].Speed, want)
	}
}

func TestBuildTelemetryWithoutWraparoundStartsFromDefaultSpeed(t *testing.T) {
	segments := []trackSegment{{Type: "straight", Length: 5}}

	got, err := buildTelemetry(segments, false)
	if err != nil {
		t.Fatalf("buildTelemetry returned error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected telemetry points")
	}

	if math.Abs(got[0].Speed-defaultTelemetryStartSpeed) > 1e-9 {
		t.Fatalf("got non-wrap initial speed %.6f, want %.6f", got[0].Speed, defaultTelemetryStartSpeed)
	}
}

func TestBuildTelemetryOneLapWraparoundBrakesForNextLapCorner(t *testing.T) {
	segments := []trackSegment{
		{Type: "curve", Radius: 5, Angle: 90},
		{Type: "straight", Length: 200},
	}

	noWrap, err := buildTelemetryOneLapWithWraparound(segments, defaultTelemetryStartSpeed, false)
	if err != nil {
		t.Fatalf("buildTelemetryOneLapWithWraparound(false) returned error: %v", err)
	}
	wrap, err := buildTelemetryOneLapWithWraparound(segments, defaultTelemetryStartSpeed, true)
	if err != nil {
		t.Fatalf("buildTelemetryOneLapWithWraparound(true) returned error: %v", err)
	}
	if len(noWrap) == 0 || len(wrap) == 0 {
		t.Fatal("expected telemetry points")
	}

	noWrapTerminal := noWrap[len(noWrap)-1].Speed
	wrapTerminal := wrap[len(wrap)-1].Speed
	if wrapTerminal >= noWrapTerminal {
		t.Fatalf("expected wraparound terminal speed %.6f to be lower than non-wrap %.6f", wrapTerminal, noWrapTerminal)
	}
}

func TestBuildTelemetryOneLapWraparoundPreservesLapDistance(t *testing.T) {
	segments := []trackSegment{
		{Type: "curve", Radius: 5, Angle: 90},
		{Type: "straight", Length: 200},
	}

	wrap, err := buildTelemetryOneLapWithWraparound(segments, defaultTelemetryStartSpeed, true)
	if err != nil {
		t.Fatalf("buildTelemetryOneLapWithWraparound(true) returned error: %v", err)
	}
	if len(wrap) == 0 {
		t.Fatal("expected telemetry points")
	}

	want := getTotalLength(telemetryTrackFromSegments(segments))
	got := wrap[len(wrap)-1].Distance
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got lap distance %.6f, want %.6f", got, want)
	}
}

func TestBuildTelemetryWraparoundChangesDefaultTrack(t *testing.T) {
	segments := defaultTrackSegments()

	noWrap, err := buildTelemetry(segments, false)
	if err != nil {
		t.Fatalf("buildTelemetry(false) returned error: %v", err)
	}
	wrap, err := buildTelemetry(segments, true)
	if err != nil {
		t.Fatalf("buildTelemetry(true) returned error: %v", err)
	}
	if len(noWrap) != len(wrap) {
		t.Fatalf("telemetry length mismatch: noWrap=%d wrap=%d", len(noWrap), len(wrap))
	}

	maxDiff := 0.0
	maxIdx := 0
	for i := range wrap {
		diff := math.Abs(wrap[i].Speed - noWrap[i].Speed)
		if diff > maxDiff {
			maxDiff = diff
			maxIdx = i
		}
	}

	t.Logf(
		"start speed noWrap=%.3f wrap=%.3f; max pointwise diff=%.3f at index=%d distance=%.3f",
		noWrap[0].Speed,
		wrap[0].Speed,
		maxDiff,
		maxIdx,
		wrap[maxIdx].Distance,
	)

	if maxDiff <= 1e-6 {
		t.Fatal("expected wraparound telemetry to differ from non-wrap telemetry on default track")
	}
}
