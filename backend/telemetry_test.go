package main

import (
	"math"
	"testing"
)

func TestBuildTelemetryOneLapStartsAtProvidedSpeed(t *testing.T) {
	segments := []trackSegment{{Type: "straight", Length: 5}}

	got := buildTelemetryOneLap(segments, 7.25)
	if len(got) == 0 {
		t.Fatal("expected telemetry points")
	}

	if math.Abs(got[0].Speed-7.25) > 1e-9 {
		t.Fatalf("got initial speed %.6f, want %.6f", got[0].Speed, 7.25)
	}
}

func TestBuildTelemetryKeepsExistingDefaultStartSpeed(t *testing.T) {
	segments := []trackSegment{{Type: "straight", Length: 5}}

	got := buildTelemetry(segments)
	if len(got) == 0 {
		t.Fatal("expected telemetry points")
	}

	if math.Abs(got[0].Speed-0.5) > 1e-9 {
		t.Fatalf("got default initial speed %.6f, want %.6f", got[0].Speed, 0.5)
	}
}
