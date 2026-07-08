package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSimulateHandlerReturnsDistanceAndTelemetry(t *testing.T) {
	body, err := json.Marshal(simulateRequest{
		Inputs:     defaultSimulationInputs(),
		Wraparound: false,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	simulateHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got simulateResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if !got.OK {
		t.Fatalf("got ok=false with message %q", got.Message)
	}
	if got.DistanceM <= 0 {
		t.Fatalf("got distance %.6f, want positive", got.DistanceM)
	}
	if len(got.Points) == 0 {
		t.Fatal("expected telemetry points")
	}
}

func TestComputeOptimalSpeedMinimizesLeftoverEnergy(t *testing.T) {
	base := defaultSimulationInputs()
	base.BatteryWh = 100
	base.AdditionalEfficiency = 0
	base.V = computeOptimalSpeedForInputs(base)

	penalized := base
	penalized.AdditionalEfficiency = 10
	penalized.V = computeOptimalSpeedForInputs(penalized)

	baseLeftover := remainingEnergyForInputs(base)
	penalizedLeftover := remainingEnergyForInputs(penalized)

	if penalizedLeftover > baseLeftover {
		t.Fatalf(
			"got penalized leftover %.6f, want less than or equal to base leftover %.6f",
			penalizedLeftover,
			baseLeftover,
		)
	}
}
