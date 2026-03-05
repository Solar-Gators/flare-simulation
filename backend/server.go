package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
)

type distanceRequest struct {
    BatteryWh     float64 `json:"batteryWh"`
    SolarWhPerMin float64 `json:"solarWhPerMin"`
    EtaDrive      float64 `json:"etaDrive"`
    RaceDayMin    float64 `json:"raceDayMin"`

    RWheel float64 `json:"rWheel"`
    Tmax   float64 `json:"tMax"`
    Pmax   float64 `json:"pMax"`

    M     float64 `json:"m"`
    G     float64 `json:"g"`
    Crr   float64 `json:"cRr"`
    Rho   float64 `json:"rho"`
    Cd    float64 `json:"cD"`
    A     float64 `json:"a"`
    Theta float64 `json:"theta"`

    Gmax       float64 `json:"gmax"`
    Wraparound bool `json:"wraparound"`
}

type distanceResponse struct {
    DistanceM   float64 `json:"distanceM"`
    OptimalV    float64 `json:"optimalV"`
    RemainingWh float64 `json:"remainingWh"`
    OK          bool    `json:"ok"`
    Message     string  `json:"message,omitempty"`
}

type trackSegment struct {
    Type      string  `json:"type"`
    Length    float64 `json:"length,omitempty"`
    Radius    float64 `json:"radius,omitempty"`
    Angle     float64 `json:"angle,omitempty"`
    Direction string  `json:"direction,omitempty"`
}

type trackResponse struct {
    Segments []trackSegment `json:"segments"`
}

type telemetryPoint struct {
    X        float64 `json:"x"`
    Y        float64 `json:"y"`
    Speed    float64 `json:"speed"`
    Accel    float64 `json:"accel"`
    Distance float64 `json:"distance"`
    VCap     float64 `json:"vCap"`
}

type telemetryRequest struct {
    // same inputs as /distance
    BatteryWh     float64 `json:"batteryWh"`
    SolarWhPerMin float64 `json:"solarWhPerMin"`
    EtaDrive      float64 `json:"etaDrive"`
    RaceDayMin    float64 `json:"raceDayMin"`
    RWheel        float64 `json:"rWheel"`
    Tmax          float64 `json:"tMax"`
    Pmax          float64 `json:"pMax"`
    M             float64 `json:"m"`
    G             float64 `json:"g"`
    Crr           float64 `json:"cRr"`
    Rho           float64 `json:"rho"`
    Cd            float64 `json:"cD"`
    A             float64 `json:"a"`
    Theta         float64 `json:"theta"`
    Gmax       float64 `json:"gmax"`

    Wraparound    bool    `json:"wraparound"`

    // provided by frontend from /distance result
    BaseTarget float64 `json:"baseTarget"`
}

type telemetryResponse struct {
    Points  []telemetryPoint `json:"points"`
    OK      bool             `json:"ok"`
    Message string           `json:"message,omitempty"`
}

func main() {
    mode := flag.String("mode", "server", "mode: server or simulate")
    addr := flag.String("addr", ":8080", "server listen address")
    flag.Parse()

    if *mode == "simulate" {
        runSimulation()
        return
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/distance", distanceHandler)
    mux.HandleFunc("/track", trackHandler)
    mux.HandleFunc("/track/telemetry", trackTelemetryHandler)

    log.Printf("listening on %s", *addr)
    if err := http.ListenAndServe(*addr, mux); err != nil {
        log.Fatal(err)
    }
}

func distanceHandler(w http.ResponseWriter, r *http.Request) {
    addCORSHeaders(w)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req distanceRequest
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "invalid JSON body"})
        return
    }

    if req.BatteryWh <= 0 || req.EtaDrive <= 0 || req.RaceDayMin <= 0 ||
        req.RWheel <= 0 || req.Tmax <= 0 || req.Pmax <= 0 || req.M <= 0 || req.G <= 0 ||
        req.Crr < 0 || req.Rho <= 0 || req.Cd <= 0 || req.A <= 0 || req.Gmax <= 0 || req.Gmax > 2.0{
        writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "missing or invalid input values"})
        return
    }

    segments := defaultTrackSegments()
    lapLen := totalLapLengthM(segments)

    bestV, remainingWh, ok := findOptimalVForFullDepletion(segments, req, 1.0, 40.0)
    if !ok {
        writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "could not optimize speed for full depletion"})
        return
    }

    m := simulateLapMetrics(segments, req, bestV)
    if !m.ok || m.lapTimeSec <= 0 {
        writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "lap simulation failed"})
        return
    }

    raceSec := req.RaceDayMin * 60.0
    laps := raceSec / m.lapTimeSec
    distanceM := laps * lapLen

    writeJSON(w, http.StatusOK, distanceResponse{
        DistanceM:   distanceM,
        OptimalV:    bestV,
        RemainingWh: remainingWh,
        OK:          true,
    })
}

func trackHandler(w http.ResponseWriter, r *http.Request) {
    addCORSHeaders(w)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    writeJSON(w, http.StatusOK, trackResponse{Segments: defaultTrackSegments()})
}

func trackTelemetryHandler(w http.ResponseWriter, r *http.Request) {
    addCORSHeaders(w)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req telemetryRequest
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()
    if err := dec.Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, telemetryResponse{OK: false, Message: "invalid JSON body"})
        return
    }

    if req.EtaDrive <= 0 || req.RaceDayMin <= 0 ||
        req.RWheel <= 0 || req.Tmax <= 0 || req.Pmax <= 0 || req.M <= 0 || req.G <= 0 ||
        req.Crr < 0 || req.Rho <= 0 || req.Cd <= 0 || req.A <= 0 ||
        req.BaseTarget <= 0 || req.Gmax <= 0 || req.Gmax > 2.0{
        writeJSON(w, http.StatusBadRequest, telemetryResponse{OK: false, Message: "missing or invalid input values"})
        return
    }

    startFromZero := !req.Wraparound

    points := buildTelemetryWithParams(
        defaultTrackSegments(),
        req.Wraparound,
        startFromZero,
        req.M, req.G, req.Crr, req.Rho, req.Cd, req.A, req.Theta,
        req.RWheel, req.Tmax, req.Pmax, req.EtaDrive,
        req.BaseTarget, req.Gmax,
    )

    writeJSON(w, http.StatusOK, telemetryResponse{Points: points, OK: true})
}

func addCORSHeaders(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    w.Header().Set("Content-Type", "application/json")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(payload); err != nil {
        fmt.Fprint(w, `{"ok":false,"message":"failed to encode response"}`)
    }
}

func defaultTrackSegments() []trackSegment {
	return []trackSegment{
		{Type: "straight", Length: 228.0829302},
		{Type: "curve", Radius: 48.96199032 * 180.0 / (math.Pi * math.Abs(-9.5)), Angle: -9.5},
		{Type: "curve", Radius: 28.41188666 * 180.0 / (math.Pi * math.Abs(-5.25)), Angle: -5.25},
		{Type: "curve", Radius: 34.05114029 * 180.0 / (math.Pi * math.Abs(-7.52)), Angle: -7.52},
		{Type: "curve", Radius: 43.77055978 * 180.0 / (math.Pi * math.Abs(-7.57)), Angle: -7.57},
		{Type: "curve", Radius: 44.5252246 * 180.0 / (math.Pi * math.Abs(-17)), Angle: -17},
		{Type: "curve", Radius: 38.03178991 * 180.0 / (math.Pi * math.Abs(-5.57)), Angle: -5.57},
		{Type: "curve", Radius: 38.11472011 * 180.0 / (math.Pi * math.Abs(-7.81)), Angle: -7.81},
		{Type: "curve", Radius: 146.8196268 * 180.0 / (math.Pi * math.Abs(-2.72)), Angle: -2.72},
		{Type: "curve", Radius: 162.2446441 * 180.0 / (math.Pi * math.Abs(-6.24)), Angle: -6.24},
		{Type: "curve", Radius: 69.01451279 * 180.0 / (math.Pi * math.Abs(-10.57)), Angle: -10.57},
		{Type: "curve", Radius: 43.13199724 * 180.0 / (math.Pi * math.Abs(-8.28)), Angle: -8.28},
		{Type: "curve", Radius: 44.08569454 * 180.0 / (math.Pi * math.Abs(-11.3)), Angle: -11.3},
		{Type: "curve", Radius: 47.78438148 * 180.0 / (math.Pi * math.Abs(-7.7)), Angle: -7.7},
		{Type: "curve", Radius: 57.97650311 * 180.0 / (math.Pi * math.Abs(-13.12)), Angle: -13.12},
		{Type: "curve", Radius: 59.32826538 * 180.0 / (math.Pi * math.Abs(-12.37)), Angle: -12.37},
		{Type: "curve", Radius: 60.05805114 * 180.0 / (math.Pi * math.Abs(-7.74)), Angle: -7.74},
		{Type: "curve", Radius: 260.8486524 * 180.0 / (math.Pi * math.Abs(-6.71)), Angle: -6.71},
		{Type: "curve", Radius: 32.17691776 * 180.0 / (math.Pi * math.Abs(-11.09)), Angle: -11.09},
		{Type: "curve", Radius: 20.17691776 * 180.0 / (math.Pi * math.Abs(-11.85)), Angle: -11.85},
		{Type: "curve", Radius: 19.87836904 * 180.0 / (math.Pi * math.Abs(-19.53)), Angle: -19.53},
		{Type: "curve", Radius: 18.83344851 * 180.0 / (math.Pi * math.Abs(-19.36)), Angle: -19.36},
		{Type: "curve", Radius: 23.67657222 * 180.0 / (math.Pi * math.Abs(-15.6)), Angle: -15.6},
		{Type: "curve", Radius: 24.82100898 * 180.0 / (math.Pi * math.Abs(-25.25)), Angle: -25.25},
		{Type: "curve", Radius: 22.08431237 * 180.0 / (math.Pi * math.Abs(-15.17)), Angle: -15.17},
		{Type: "curve", Radius: 257.2080166 * 180.0 / (math.Pi * math.Abs(-8.78)), Angle: -8.78},
		{Type: "curve", Radius: 30.13683483 * 180.0 / (math.Pi * math.Abs(9.25)), Angle: 9.25},
		{Type: "curve", Radius: 26.27228749 * 180.0 / (math.Pi * math.Abs(16.89)), Angle: 16.89},
		{Type: "curve", Radius: 17.15825847 * 180.0 / (math.Pi * math.Abs(8.78)), Angle: 8.78},
		{Type: "curve", Radius: 15.9308915 * 180.0 / (math.Pi * math.Abs(17.39)), Angle: 17.39},
		{Type: "curve", Radius: 21.98479613 * 180.0 / (math.Pi * math.Abs(23.2)), Angle: 23.2},
		{Type: "curve", Radius: 29.44022115 * 180.0 / (math.Pi * math.Abs(24.79)), Angle: 24.79},
		{Type: "curve", Radius: 83.9170698 * 180.0 / (math.Pi * math.Abs(15.7)), Angle: 15.7},
		{Type: "curve", Radius: 30.18659295 * 180.0 / (math.Pi * math.Abs(-20.82)), Angle: -20.82},
		{Type: "curve", Radius: 15.78991016 * 180.0 / (math.Pi * math.Abs(-14.38)), Angle: -14.38},
		{Type: "curve", Radius: 16.36212854 * 180.0 / (math.Pi * math.Abs(-19.12)), Angle: -19.12},
		{Type: "curve", Radius: 16.99239806 * 180.0 / (math.Pi * math.Abs(-9.1)), Angle: -9.1},
		{Type: "curve", Radius: 22.06772633 * 180.0 / (math.Pi * math.Abs(-11.78)), Angle: -11.78},
		{Type: "curve", Radius: 167.6765722 * 180.0 / (math.Pi * math.Abs(-14.49)), Angle: -14.49},
		{Type: "curve", Radius: 24.00829302 * 180.0 / (math.Pi * math.Abs(12.63)), Angle: 12.63},
		{Type: "curve", Radius: 17.92121631 * 180.0 / (math.Pi * math.Abs(15.72)), Angle: 15.72},
		{Type: "curve", Radius: 17.19972357 * 180.0 / (math.Pi * math.Abs(16.95)), Angle: 16.95},
		{Type: "curve", Radius: 20.10228058 * 180.0 / (math.Pi * math.Abs(7.57)), Angle: 7.57},
		{Type: "curve", Radius: 15.87284036 * 180.0 / (math.Pi * math.Abs(24.01)), Angle: 24.01},
		{Type: "curve", Radius: 21.37940567 * 180.0 / (math.Pi * math.Abs(17.63)), Angle: 17.63},
		{Type: "curve", Radius: 223.8949551 * 180.0 / (math.Pi * math.Abs(3.38)), Angle: 3.38},
		{Type: "curve", Radius: 22.88873531 * 180.0 / (math.Pi * math.Abs(12.04)), Angle: 12.04},
		{Type: "curve", Radius: 29.05874223 * 180.0 / (math.Pi * math.Abs(15.09)), Angle: 15.09},
		{Type: "curve", Radius: 29.18313753 * 180.0 / (math.Pi * math.Abs(9.34)), Angle: 9.34},
		{Type: "curve", Radius: 29.18313753 * 180.0 / (math.Pi * math.Abs(-7.16)), Angle: -7.16},
		{Type: "curve", Radius: 28.02211472 * 180.0 / (math.Pi * math.Abs(-12.45)), Angle: -12.45},
		{Type: "curve", Radius: 24.10780926 * 180.0 / (math.Pi * math.Abs(-21.58)), Angle: -21.58},
		{Type: "curve", Radius: 22.15894955 * 180.0 / (math.Pi * math.Abs(-24.33)), Angle: -24.33},
		{Type: "curve", Radius: 19.6959226 * 180.0 / (math.Pi * math.Abs(-16.79)), Angle: -16.79},
		{Type: "curve", Radius: 22.07601935 * 180.0 / (math.Pi * math.Abs(-13.21)), Angle: -13.21},
		{Type: "curve", Radius: 21.42087077 * 180.0 / (math.Pi * math.Abs(-22.7)), Angle: -22.7},
		{Type: "curve", Radius: 19.27297858 * 180.0 / (math.Pi * math.Abs(-7.09)), Angle: -7.09},
		{Type: "curve", Radius: 18.09536973 * 180.0 / (math.Pi * math.Abs(-16.99)), Angle: -16.99},
		{Type: "curve", Radius: 20.39253628 * 180.0 / (math.Pi * math.Abs(-18.52)), Angle: -18.52},
		{Type: "curve", Radius: 19.68762958 * 180.0 / (math.Pi * math.Abs(-17.44)), Angle: -17.44},
		{Type: "curve", Radius: 22.13407049 * 180.0 / (math.Pi * math.Abs(-13.94)), Angle: -13.94},
		{Type: "curve", Radius: 16.85970974 * 180.0 / (math.Pi * math.Abs(-19.66)), Angle: -19.66},
		{Type: "curve", Radius: 18.06219765 * 180.0 / (math.Pi * math.Abs(-8.63)), Angle: -8.63},
		{Type: "curve", Radius: 17.15825847 * 180.0 / (math.Pi * math.Abs(-9.85)), Angle: -9.85},
		{Type: "curve", Radius: 12.31513476 * 180.0 / (math.Pi * math.Abs(-14.08)), Angle: -14.08},
		{Type: "curve", Radius: 29.02557015 * 180.0 / (math.Pi * math.Abs(-6.24)), Angle: -6.24},
		{Type: "curve", Radius: 16.54457498 * 180.0 / (math.Pi * math.Abs(11.95)), Angle: 11.95},
		{Type: "curve", Radius: 22.90532135 * 180.0 / (math.Pi * math.Abs(11.39)), Angle: 11.39},
		{Type: "curve", Radius: 70.79751209 * 180.0 / (math.Pi * math.Abs(3.04)), Angle: 3.04},
		{Type: "curve", Radius: 19.9032481 * 180.0 / (math.Pi * math.Abs(16.22)), Angle: 16.22},
		{Type: "curve", Radius: 22.45749827 * 180.0 / (math.Pi * math.Abs(17.46)), Angle: 17.46},
		{Type: "curve", Radius: 25.75812025 * 180.0 / (math.Pi * math.Abs(5.13)), Angle: 5.13},
		{Type: "curve", Radius: 33.21354527 * 180.0 / (math.Pi * math.Abs(0.11)), Angle: 0.11},
		{Type: "curve", Radius: 34.10089841 * 180.0 / (math.Pi * math.Abs(-16.19)), Angle: -16.19},
		{Type: "curve", Radius: 23.4609537 * 180.0 / (math.Pi * math.Abs(-12.13)), Angle: -12.13},
		{Type: "curve", Radius: 25.05321355 * 180.0 / (math.Pi * math.Abs(-14.44)), Angle: -14.44},
		{Type: "curve", Radius: 19.1983414 * 180.0 / (math.Pi * math.Abs(-1.86)), Angle: -1.86},
		{Type: "curve", Radius: 64.79336558 * 180.0 / (math.Pi * math.Abs(-5.65)), Angle: -5.65},
		{Type: "curve", Radius: 156.7297858 * 180.0 / (math.Pi * math.Abs(1.36)), Angle: 1.36},
		{Type: "curve", Radius: 20.64132688 * 180.0 / (math.Pi * math.Abs(11.08)), Angle: 11.08},
		{Type: "curve", Radius: 9.205252246 * 180.0 / (math.Pi * math.Abs(21.89)), Angle: 21.89},
		{Type: "curve", Radius: 18.55977885 * 180.0 / (math.Pi * math.Abs(25.97)), Angle: 25.97},
		{Type: "curve", Radius: 14.81962681 * 180.0 / (math.Pi * math.Abs(29.78)), Angle: 29.78},
		{Type: "curve", Radius: 48.0 * 180.0 / (math.Pi * math.Abs(12.36)), Angle: 12.36},
		{Type: "curve", Radius: 28.83483068 * 180.0 / (math.Pi * math.Abs(-15.86)), Angle: -15.86},
		{Type: "curve", Radius: 23.60193504 * 180.0 / (math.Pi * math.Abs(-22.95)), Angle: -22.95},
		{Type: "curve", Radius: 19.56323428 * 180.0 / (math.Pi * math.Abs(-10.39)), Angle: -10.39},
		{Type: "curve", Radius: 16.72702142 * 180.0 / (math.Pi * math.Abs(-12.79)), Angle: -12.79},
		{Type: "curve", Radius: 16.44505874 * 180.0 / (math.Pi * math.Abs(-20.69)), Angle: -20.69},
		{Type: "curve", Radius: 17.65583967 * 180.0 / (math.Pi * math.Abs(-9.97)), Angle: -9.97},
		{Type: "curve", Radius: 15.25086386 * 180.0 / (math.Pi * math.Abs(-12.54)), Angle: -12.54},
		{Type: "curve", Radius: 16.37871458 * 180.0 / (math.Pi * math.Abs(-24.75)), Angle: -24.75},
	}
}