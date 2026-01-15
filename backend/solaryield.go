package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type OpenMeteoResponse struct {
	Hourly struct {
		Time []string  `json:"time"`
		GTI  []float64 `json:"global_tilted_irradiance_instant"`
	} `json:"hourly"`
}

// FetchHourlyGTI calls Open-Meteo and returns hourly timestamps + GTI values (W/m²).
// tiltDeg: panel tilt in degrees
// azimuthDeg: panel azimuth (Open-Meteo convention: 0=south, -90=east, 90=west, ±180=north)
func FetchHourlyGTI(lat, lon, tiltDeg, azimuthDeg float64, tz string, forecastDays int) ([]time.Time, []float64, error) {
	
	// sets up base url (query parameter builder)
	base, _ := url.Parse("https://api.open-meteo.com/v1/forecast")
	q := base.Query()

	// adds parameters to url to locate car
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 6, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 6, 64))
	q.Set("hourly", "global_tilted_irradiance_instant")

	// car panel tilt
	q.Set("tilt", strconv.FormatFloat(tiltDeg, 'f', 2, 64))
	// direction panel is facing
	q.Set("azimuth", strconv.FormatFloat(azimuthDeg, 'f', 2, 64))
	// timezone
	q.Set("timezone", tz)
	if forecastDays > 0 {
		q.Set("forecast_days", strconv.Itoa(forecastDays))
	}

	// builds full url
	base.RawQuery = q.Encode()

	// connects to external weather api (built above)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(base.String())
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// check for api errors
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("open-meteo error: %s", resp.Status)
	}

	// reads JSON and converts to Open-Meteo Response struct
	var om OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&om); err != nil {
		return nil, nil, err
	}

	// Parse times as local timestamps in the provided timezone.
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, nil, err
	}

	times := make([]time.Time, 0, len(om.Hourly.Time))
	for _, ts := range om.Hourly.Time {
		// Open-Meteo hourly time strings look like: "2026-01-12T14:00"
		t, err := time.ParseInLocation("2006-01-02T15:04", ts, loc)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse time %q: %w", ts, err)
		}
		times = append(times, t)
	}

	return times, om.Hourly.GTI, nil
}

// Utilizes Open-Meteo Hourly GTI to calculate Watts
type SolarEnergyPoint struct {
	Time       time.Time
	GTI_Wm2    float64
	Energy_Wh  float64 // energy gained during this timestep
	Battery_Wh float64 // battery after adding this timestep
}

// BuildEnergySeriesWithBattery fetches GTI from Open-Meteo (via FetchHourlyGTI)
// and updates the battery over the requested span.
func BuildEnergyWithBattery(
	lat, lon float64,
	tiltDeg, azimuthDeg float64,
	timezone string,
	forecastDays int,

	panelArea float64,
	panelEff float64,
	systemEff float64,

	dt time.Duration,
	initialBatteryWh float64,

	spanStart *time.Time,
	spanEnd *time.Time,
) (totalEnergyWh float64, newBatteryWh float64, err error) {

	// 1) Fetch data
	times, gti, err := FetchHourlyGTI(lat, lon, tiltDeg, azimuthDeg, timezone, forecastDays)
	if err != nil {
		return 0, initialBatteryWh, err
	}

	dtHours := dt.Hours()
	batteryWh := initialBatteryWh
	totalEnergy := 0.0

	n := len(gti)
	if len(times) < n {
		n = len(times)
	}

	for i := 0; i < n; i++ {
		t := times[i]

		// Span filtering (8 hour window, etc.)
		if spanStart != nil && t.Before(*spanStart) {
			continue
		}
		if spanEnd != nil && !t.Before(*spanEnd) {
			break
		}

		// Power (W)
		powerW := gti[i] * panelArea * panelEff * systemEff

		// Energy (Wh)
		energyWh := powerW * dtHours

		totalEnergy += energyWh
		batteryWh += energyWh
	}

	return totalEnergy, batteryWh, nil
}
