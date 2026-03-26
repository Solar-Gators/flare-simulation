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
	got, err := buildTelemetry(segments)
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
