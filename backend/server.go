//important coasting decel is not a constant rate
//speed profile first then sim (backwards pass first)

// quick notes:
// lets try a speed limit map for the whole track
package main

import (
	//used for decoding JSON into distanceRequest struct
	//also for encoding struct back into JSON format for HTTP response
	"encoding/json"
	"strconv"

	//allows for original sim to be called via terminal using flag
	"flag"
	"fmt"
	"log"
	"math"
	"net/http" //lets go program talk over web --> Receive requests and send responses
)

type distanceRequest = simulationInputs

type distanceResponse struct {
	DistanceM float64 `json:"distanceM"`
	OK        bool    `json:"ok"`
	Message   string  `json:"message,omitempty"`
}

type simulateRequest struct {
	Inputs     simulationInputs `json:"inputs"`
	Wraparound bool             `json:"wraparound"`
}

type simulateResponse struct {
	DistanceM float64          `json:"distanceM"`
	Points    []telemetryPoint `json:"points"`
	OK        bool             `json:"ok"`
	Message   string           `json:"message,omitempty"`
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
}

type telemetryResponse struct {
	Points []telemetryPoint `json:"points"`
}

var optimalCruiseSpeed float64

// relocated main bc this is new entry point
// sim now becomes function
func main() {
	mode := flag.String("mode", "server", "mode: server or simulate") //checking for user flags for sim for server
	addr := flag.String("addr", ":8080", "server listen address")     //checking flag to choose different network port in cases 8080 is in use
	flag.Parse()                                                      //fills pointers (mode and addr) with values based on terminal inputs

	//if flag is simulate run sim
	if *mode == "simulate" {
		runSimulation()
		return
	}
	//find cruise speed
	optimalCruiseSpeed = computeOptimalSpeed()
	//empty router (router is meant to map url to handler)
	mux := http.NewServeMux() //request router (empty --> no route to go), serve multiplexer -->takes http requests and routes it
	mux.HandleFunc("/defaults", defaultsHandler)
	mux.HandleFunc("/distance", distanceHandler) // handler that router directs oncoming requests
	mux.HandleFunc("/simulate", simulateHandler)
	mux.HandleFunc("/track", trackHandler)
	mux.HandleFunc("/track/telemetry", trackTelemetryHandler)

	log.Printf("listening on %s", *addr) //%s is replaced with dereferenced addr

	//handles errors
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

// is the HTTP handler
// w --> is outgoing http response (write)
// r --> incoming http request. pointer to struct with everything client sent (read)
func distanceHandler(w http.ResponseWriter, r *http.Request) {
	addCORSHeaders(w)
	//check to see if OPTIONS request then do nothing (this happens before API request)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	//handler is called again (two http requests are made)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := defaultSimulationInputs() // prefill with backend defaults, then let JSON override provided fields
	dec := json.NewDecoder(r.Body)   //decode JSON and read
	dec.DisallowUnknownFields()      //decoding will fail if JSON has fields that are not valid
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "invalid JSON body"})
		return
	}

	if err := validateSimulationInputs(req); err != nil {
		writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: err.Error()})
		return
	}
	//run sim if everything is valid
	distance, ok := distanceForInputs(req)
	if !ok {
		writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "inputs are not feasible for the model"})
		return
	}

	writeJSON(w, http.StatusOK, distanceResponse{DistanceM: distance, OK: true})
}

func simulateHandler(w http.ResponseWriter, r *http.Request) {
	addCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := simulateRequest{
		Inputs:     defaultSimulationInputs(),
		Wraparound: true,
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, simulateResponse{OK: false, Message: "invalid JSON body"})
		return
	}

	if err := validateSimulationInputs(req.Inputs); err != nil {
		writeJSON(w, http.StatusBadRequest, simulateResponse{OK: false, Message: err.Error()})
		return
	}

	distance, ok := distanceForInputs(req.Inputs)
	if !ok {
		writeJSON(w, http.StatusBadRequest, simulateResponse{OK: false, Message: "inputs are not feasible for the model"})
		return
	}

	points, err := buildTelemetryForInputs(defaultTrackSegments(), req.Wraparound, req.Inputs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, simulateResponse{OK: false, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, simulateResponse{DistanceM: distance, Points: points, OK: true})
}

func validateSimulationInputs(req simulationInputs) error {
	if req.V <= 0 || req.BatteryWh <= 0 || req.EtaDrive <= 0 || req.RaceDayMin <= 0 ||
		req.RWheel <= 0 || req.Tmax <= 0 || req.Pmax <= 0 || req.M <= 0 || req.G <= 0 ||
		req.Crr < 0 || req.Rho <= 0 || req.Cd <= 0 || req.A <= 0 || req.Gmax <= 0 ||
		req.AdditionalEfficiency < -100 || req.AdditionalEfficiency > 100 {
		return fmt.Errorf("missing or invalid input values")
	}
	return nil
}

func distanceForInputs(req simulationInputs) (float64, bool) {
	return DistanceForSpeedEV(
		req.V,
		req.BatteryWh, req.SolarWhPerMin, req.EtaDrive, req.RaceDayMin,
		req.RWheel, req.Tmax, req.Pmax,
		req.M, req.G, req.Crr, req.Rho, req.Cd, req.A, req.Theta, req.AdditionalEfficiency,
	)
}

func defaultsHandler(w http.ResponseWriter, r *http.Request) {
	addCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, simulationDefaultsResponse())
}

// adds specific http response headers
// CORS = Cross Origin Resource Sharing
// Rule for controlling which websites can talk to which servers
// OPTIONS: asks for permission "what am i allowed to do?" ex. "can i send POST", "can i send JSON"
// POST --> client sends data and then server process it
func addCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")                   //any website can make request to this backend
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS") //the frontend can make POST (API call) and OPTIONS (CORS preflight)request
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")       //frontend can send content type headers
	w.Header().Set("Content-Type", "application/json")                   //response body is in JSON
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		fmt.Fprint(w, `{"ok":false,"message":"failed to encode response"}`)
	}
}

// http handler for track GET request
// http request has for major parts: request line (http method, url, verison), headers
// (format for body, how long body is), blank line, and body (optional and is a stream)
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

	resp := trackResponse{Segments: defaultTrackSegments()}
	writeJSON(w, http.StatusOK, resp)
}

// setting tracks
func defaultTrackSegments() []trackSegment {
	return []trackSegment{
		{Type: "straight", Length: 1178.833711},
		{Type: "curve", Radius: 77.10287933 * 180.0 / (math.Pi * math.Abs(-12.41)), Angle: -12.41},
		{Type: "curve", Radius: 53.38887092 * 180.0 / (math.Pi * math.Abs(-12.22)), Angle: -12.22},
		{Type: "curve", Radius: 50.69556778 * 180.0 / (math.Pi * math.Abs(-9.8)), Angle: -9.8},
		{Type: "curve", Radius: 40.11646716 * 180.0 / (math.Pi * math.Abs(-14.37)), Angle: -14.37},
		{Type: "curve", Radius: 58.34681333 * 180.0 / (math.Pi * math.Abs(-10.42)), Angle: -10.42},
		{Type: "curve", Radius: 164.0812035 * 180.0 / (math.Pi * math.Abs(-4.92)), Angle: -4.92},
		{Type: "curve", Radius: 57.27919767 * 180.0 / (math.Pi * math.Abs(-3.5)), Angle: -3.5},
		{Type: "straight", Length: 140.4561631}, // -0.79 -> straight
		{Type: "curve", Radius: 37.13199612 * 180.0 / (math.Pi * math.Abs(-7.19)), Angle: -7.19},
		{Type: "curve", Radius: 27.21611129 * 180.0 / (math.Pi * math.Abs(-14.51)), Angle: -14.51},
		{Type: "curve", Radius: 34.60045293 * 180.0 / (math.Pi * math.Abs(-7.59)), Angle: -7.59},
		{Type: "curve", Radius: 32.58654157 * 180.0 / (math.Pi * math.Abs(-6.97)), Angle: -6.97},
		{Type: "curve", Radius: 57.5056616 * 180.0 / (math.Pi * math.Abs(-6.7)), Angle: -6.7},
		{Type: "curve", Radius: 56.75347784 * 180.0 / (math.Pi * math.Abs(-17.22)), Angle: -17.22},
		{Type: "curve", Radius: 126.5771595 * 180.0 / (math.Pi * math.Abs(-15.51)), Angle: -15.51},
		{Type: "straight", Length: 178.235199}, // -0.02 -> straight
		{Type: "curve", Radius: 46.2633452 * 180.0 / (math.Pi * math.Abs(-3.88)), Angle: -3.88},
		{Type: "curve", Radius: 30.45939825 * 180.0 / (math.Pi * math.Abs(-21.31)), Angle: -21.31},
		{Type: "curve", Radius: 21.49789712 * 180.0 / (math.Pi * math.Abs(-28.1)), Angle: -28.1},
		{Type: "curve", Radius: 22.12876092 * 180.0 / (math.Pi * math.Abs(-14.61)), Angle: -14.61},
		{Type: "curve", Radius: 25.51763183 * 180.0 / (math.Pi * math.Abs(-27.61)), Angle: -27.61},
		{Type: "curve", Radius: 44.83176965 * 180.0 / (math.Pi * math.Abs(-22.5)), Angle: -22.5},
		{Type: "curve", Radius: 240.480427 * 180.0 / (math.Pi * math.Abs(-9.7)), Angle: -9.7},
		{Type: "curve", Radius: 35.6357166 * 180.0 / (math.Pi * math.Abs(11.78)), Angle: 11.78},
		{Type: "curve", Radius: 23.60886445 * 180.0 / (math.Pi * math.Abs(23.99)), Angle: 23.99},
		{Type: "curve", Radius: 17.60757037 * 180.0 / (math.Pi * math.Abs(16.88)), Angle: 16.88},
		{Type: "curve", Radius: 14.76868327 * 180.0 / (math.Pi * math.Abs(21.52)), Angle: 21.52},
		{Type: "curve", Radius: 27.34551925 * 180.0 / (math.Pi * math.Abs(18.22)), Angle: 18.22},
		{Type: "curve", Radius: 28.32416694 * 180.0 / (math.Pi * math.Abs(12.98)), Angle: 12.98},
		{Type: "curve", Radius: 39.52604335 * 180.0 / (math.Pi * math.Abs(9.65)), Angle: 9.65},
		{Type: "curve", Radius: 30.42704626 * 180.0 / (math.Pi * math.Abs(6.08)), Angle: 6.08},
		{Type: "curve", Radius: 23.30152054 * 180.0 / (math.Pi * math.Abs(-20.78)), Angle: -20.78},
		{Type: "curve", Radius: 16.37010676 * 180.0 / (math.Pi * math.Abs(-22.42)), Angle: -22.42},
		{Type: "curve", Radius: 16.83921061 * 180.0 / (math.Pi * math.Abs(-18.81)), Angle: -18.81},
		{Type: "curve", Radius: 24.991912 * 180.0 / (math.Pi * math.Abs(-11.03)), Angle: -11.03},
		{Type: "curve", Radius: 31.01747007 * 180.0 / (math.Pi * math.Abs(-19.6)), Angle: -19.6},
		{Type: "curve", Radius: 133.4196053 * 180.0 / (math.Pi * math.Abs(-2.39)), Angle: -2.39},
		{Type: "curve", Radius: 32.74021352 * 180.0 / (math.Pi * math.Abs(7.71)), Angle: 7.71},
		{Type: "curve", Radius: 19.58104173 * 180.0 / (math.Pi * math.Abs(20.47)), Angle: 20.47},
		{Type: "curve", Radius: 18.71562601 * 180.0 / (math.Pi * math.Abs(23.65)), Angle: 23.65},
		{Type: "curve", Radius: 21.64348107 * 180.0 / (math.Pi * math.Abs(18.53)), Angle: 18.53},
		{Type: "curve", Radius: 23.03461663 * 180.0 / (math.Pi * math.Abs(20.62)), Angle: 20.62},
		{Type: "curve", Radius: 211.0562925 * 180.0 / (math.Pi * math.Abs(6.05)), Angle: 6.05},
		{Type: "curve", Radius: 43.4244581 * 180.0 / (math.Pi * math.Abs(11.3)), Angle: 11.3},
		{Type: "curve", Radius: 22.78388871 * 180.0 / (math.Pi * math.Abs(20.69)), Angle: 20.69},
		{Type: "curve", Radius: 35.44969266 * 180.0 / (math.Pi * math.Abs(10.49)), Angle: 10.49},
		{Type: "curve", Radius: 31.27628599 * 180.0 / (math.Pi * math.Abs(-8.83)), Angle: -8.83},
		{Type: "curve", Radius: 26.26981559 * 180.0 / (math.Pi * math.Abs(-29.27)), Angle: -29.27},
		{Type: "curve", Radius: 40.17308314 * 180.0 / (math.Pi * math.Abs(-28.45)), Angle: -28.45},
		{Type: "curve", Radius: 134.0909091 * 180.0 / (math.Pi * math.Abs(-2.4)), Angle: -2.4},
		{Type: "curve", Radius: 51.11614364 * 180.0 / (math.Pi * math.Abs(-11.27)), Angle: -11.27},
		{Type: "curve", Radius: 38.88709156 * 180.0 / (math.Pi * math.Abs(-21.34)), Angle: -21.34},
		{Type: "curve", Radius: 46.00452928 * 180.0 / (math.Pi * math.Abs(-27.22)), Angle: -27.22},
		{Type: "curve", Radius: 212.2533161 * 180.0 / (math.Pi * math.Abs(-5.71)), Angle: -5.71},
		{Type: "straight", Length: 326.918473}, // -0.86 -> straight
		{Type: "curve", Radius: 33.59754125 * 180.0 / (math.Pi * math.Abs(-16.71)), Angle: -16.71},
		{Type: "curve", Radius: 27.92785506 * 180.0 / (math.Pi * math.Abs(-25.6)), Angle: -25.6},
		{Type: "curve", Radius: 27.00582336 * 180.0 / (math.Pi * math.Abs(-30.42)), Angle: -30.42},
		{Type: "curve", Radius: 20.07440958 * 180.0 / (math.Pi * math.Abs(-22.14)), Angle: -22.14},
		{Type: "curve", Radius: 21.78906503 * 180.0 / (math.Pi * math.Abs(-8.64)), Angle: -8.64},
		{Type: "curve", Radius: 18.02005823 * 180.0 / (math.Pi * math.Abs(-11.86)), Angle: -11.86},
	}
}

func trackTelemetryHandler(w http.ResponseWriter, r *http.Request) {
	addCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wraparound, err := telemetryWraparoundFromQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, struct {
			Points  []telemetryPoint `json:"points"`
			Message string           `json:"message,omitempty"`
		}{
			Points:  []telemetryPoint{},
			Message: err.Error(),
		})
		return
	}

	segments := defaultTrackSegments()
	points, err := buildTelemetry(segments, wraparound)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, struct {
			Points  []telemetryPoint `json:"points"`
			Message string           `json:"message,omitempty"`
		}{
			Points:  []telemetryPoint{},
			Message: err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, telemetryResponse{Points: points})
}

func telemetryWraparoundFromQuery(r *http.Request) (bool, error) {
	raw := r.URL.Query().Get("wraparound")
	if raw == "" {
		return true, nil
	}

	wraparound, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid wraparound query value %q", raw)
	}
	return wraparound, nil
}

// returns a list of points (x,y,speed,accel,distance)
// no more speed "snap" in curves as in instead of setting v to the V capacity
// the V is now ramp towards the cap given acceleration limits
// ++ included friction-circle limit bc in curves lateral acceleration uses grip which reduces how much longitudinal accel/brake you can apply
// we have target cruise speed to prevent the car from accelerating forever if V is below we accelerate and if above we brake
// fixed our issue of V approaching and reaching 0 bc of curves by removing continuous coasting decel curves and replacing by controlled approach to target curve speed
// we essentially established a baseline for the optimal speed around curve instead of always taking foot off gas when approaching curve.
const (
	defaultTelemetryStartSpeed = 0.5
	telemetryWarmupMaxLaps     = 5
	telemetryWrapProfileLaps   = 3
	telemetryWarmupTolerance   = 1e-3
)

func buildTelemetry(segments []trackSegment, wraparound bool) ([]telemetryPoint, error) {
	return buildTelemetryForInputs(segments, wraparound, defaultSimulationInputs())
}

func buildTelemetryForInputs(segments []trackSegment, wraparound bool, inputs simulationInputs) ([]telemetryPoint, error) {
	if !wraparound {
		return buildTelemetryOneLapWithWraparoundForInputs(segments, defaultTelemetryStartSpeed, false, inputs)
	}

	startSpeed, err := warmTelemetryStartSpeedForInputs(
		segments,
		inputs,
		defaultTelemetryStartSpeed,
		telemetryWarmupMaxLaps,
		telemetryWarmupTolerance,
	)
	if err != nil {
		return nil, err
	}
	return buildTelemetryOneLapWithWraparoundForInputs(segments, startSpeed, true, inputs)
}

// buildTelemetryOneLap simulates a single lap and starts from the provided speed.
// This is the primitive needed for later wraparound support, where one lap warms up
// the state and the next lap starts from the previous lap's terminal speed.
func buildTelemetryOneLap(segments []trackSegment, startSpeed float64) ([]telemetryPoint, error) {
	return buildTelemetryOneLapForInputs(segments, startSpeed, defaultSimulationInputs())
}

func buildTelemetryOneLapForInputs(segments []trackSegment, startSpeed float64, inputs simulationInputs) ([]telemetryPoint, error) {
	return buildTelemetryOneLapWithWraparoundForInputs(segments, startSpeed, true, inputs)
}

// buildTelemetryOneLapWithWraparound simulates one lap and chooses whether the
// brake/coast profiles should look only within the current lap or continue
// across the finish line into the next lap.
func buildTelemetryOneLapWithWraparound(
	segments []trackSegment,
	startSpeed float64,
	wraparound bool,
) ([]telemetryPoint, error) {
	return buildTelemetryOneLapWithWraparoundForInputs(
		segments,
		startSpeed,
		wraparound,
		defaultSimulationInputs(),
	)
}

func buildTelemetryOneLapWithWraparoundForInputs(
	segments []trackSegment,
	startSpeed float64,
	wraparound bool,
	inputs simulationInputs,
) ([]telemetryPoint, error) {
	const (
		stepM    = 1.0
		muTire   = 0.9
		maxSpeed = 40.0
		vMin     = 0.5
	)

	track := telemetryTrackFromSegments(segments)
	cruiseCap := math.Min(maxSpeed, inputs.V)
	if cruiseCap <= 0 {
		cruiseCap = maxSpeed
	}
	profiles, err := buildTelemetryProfiles(
		track,
		wraparound,
		stepM,
		inputs.Gmax,
		cruiseCap,
		0.95*inputs.G,
		vMin,
		inputs.M,
		inputs.G,
		inputs.Crr,
		inputs.Rho,
		inputs.Cd,
		inputs.A,
		inputs.Theta,
		inputs.AdditionalEfficiency,
	)
	if err != nil {
		return nil, err
	}

	points := make([]telemetryPoint, 0, 64)
	x, y, heading := 0.0, 0.0, 0.0
	startSpeed, err = validateTelemetryStartSpeed(startSpeed)
	if err != nil {
		return nil, err
	}
	v := startSpeed
	distance := 0.0
	profileIdx := 0
	points = append(points, telemetryPoint{X: x, Y: y, Speed: v, Accel: 0, Distance: distance})
	// save the initial pose so we can convincingly close the lap later if
	// numerical sampling leaves a small gap between the end and the start.
	initialX, initialY := x, y

	for _, seg := range segments {
		switch seg.Type {
		//when we are dealing with a straight segment
		case "straight":
			remaining := seg.Length
			for remaining > 0 {
				ds := math.Min(stepM, remaining) //going thru every stepM meters (10m)
				brakeSpeed := cruiseCap
				coastSpeed := cruiseCap
				if profileIdx < len(profiles.Brake) {
					brakeSpeed = profiles.Brake[profileIdx]
				}
				if profileIdx < len(profiles.Coast) {
					coastSpeed = profiles.Coast[profileIdx]
				}
				aLongMax := muTire * inputs.G
				var a float64
				if v > brakeSpeed {
					aReq := (brakeSpeed*brakeSpeed - v*v) / (2 * ds)
					a = math.Max(aReq, -aLongMax)
				} else if v > coastSpeed {
					aCoast := coastDecelFromPower(
						v,
						vMin,
						inputs.M,
						inputs.G,
						inputs.Crr,
						inputs.Rho,
						inputs.Cd,
						inputs.A,
						inputs.Theta,
						inputs.AdditionalEfficiency,
					)
					a = math.Max(aCoast, -aLongMax)
				} else {
					aPower := accelAtSpeed(
						v,
						vMin,
						inputs.RWheel,
						inputs.Tmax,
						inputs.Pmax,
						inputs.EtaDrive,
						inputs.M,
						inputs.G,
						inputs.Crr,
						inputs.Rho,
						inputs.Cd,
						inputs.A,
						inputs.Theta,
						inputs.AdditionalEfficiency,
					)
					a = math.Min(aPower, aLongMax)
				}
				vNext := updateSpeed(v, a, ds)
				if vNext > brakeSpeed {
					vNext = brakeSpeed
				}
				//update position
				x += ds * math.Cos(heading)
				y += ds * math.Sin(heading)
				distance += ds
				points = append(points, telemetryPoint{X: x, Y: y, Speed: vNext, Accel: a, Distance: distance})
				v = vNext
				remaining -= ds
				profileIdx++
			}
		case "curve":
			if seg.Angle == 0 {
				continue
			}
			if seg.Radius == 0 {
				heading += seg.Angle * math.Pi / 180.0
				points = append(points, telemetryPoint{X: x, Y: y, Speed: v, Accel: 0, Distance: distance})
				continue
			}
			aLatMax := inputs.Gmax * inputs.G
			vCap := math.Sqrt(aLatMax * seg.Radius)
			angleDeg := seg.Angle
			arcLength := seg.Radius * math.Abs(angleDeg) * math.Pi / 180.0
			remaining := arcLength
			isRight := angleDeg < 0
			//computes data points for each step along curve segment
			for remaining > 0 {
				ds := math.Min(stepM, remaining)
				brakeSpeed := cruiseCap
				coastSpeed := cruiseCap
				if profileIdx < len(profiles.Brake) {
					brakeSpeed = profiles.Brake[profileIdx]
				}
				if profileIdx < len(profiles.Coast) {
					coastSpeed = profiles.Coast[profileIdx]
				}
				if brakeSpeed > vCap {
					brakeSpeed = vCap
				}
				if coastSpeed > vCap {
					coastSpeed = vCap
				}
				delta := ds / seg.Radius
				if !isRight {
					delta = -delta
				}

				normalX := -math.Sin(heading)
				normalY := math.Cos(heading)
				if !isRight {
					normalX = math.Sin(heading)
					normalY = -math.Cos(heading)
				}
				centerX := x + seg.Radius*normalX
				centerY := y + seg.Radius*normalY
				dx := x - centerX
				dy := y - centerY
				cos := math.Cos(delta)
				sin := math.Sin(delta)
				x = centerX + dx*cos - dy*sin
				y = centerY + dx*sin + dy*cos
				heading += delta
				distance += ds
				aLat := (v * v) / seg.Radius
				aTotalMax := muTire * inputs.G
				aLongMax := math.Sqrt(math.Max(0, aTotalMax*aTotalMax-aLat*aLat))
				var a float64
				if v > brakeSpeed {
					aReq := (brakeSpeed*brakeSpeed - v*v) / (2 * ds)
					a = math.Max(aReq, -aLongMax)
				} else if v > coastSpeed {
					aCoast := coastDecelFromPower(
						v,
						vMin,
						inputs.M,
						inputs.G,
						inputs.Crr,
						inputs.Rho,
						inputs.Cd,
						inputs.A,
						inputs.Theta,
						inputs.AdditionalEfficiency,
					)
					a = math.Max(aCoast, -aLongMax)
				} else {
					aPower := accelAtSpeed(
						v,
						vMin,
						inputs.RWheel,
						inputs.Tmax,
						inputs.Pmax,
						inputs.EtaDrive,
						inputs.M,
						inputs.G,
						inputs.Crr,
						inputs.Rho,
						inputs.Cd,
						inputs.A,
						inputs.Theta,
						inputs.AdditionalEfficiency,
					)
					a = math.Min(aPower, aLongMax)
				}
				vNext := updateSpeed(v, a, ds)
				if vNext > brakeSpeed {
					vNext = brakeSpeed
				}
				points = append(points, telemetryPoint{X: x, Y: y, Speed: vNext, Accel: a, Distance: distance})
				v = vNext
				remaining -= ds
				profileIdx++
			}
		}
	}

	// If the final point doesn't exactly match the starting point (small
	// numerical drift from sampling), snap the final telemetry point to the
	// starting position so the visual path closes cleanly. This avoids
	// changing track geometry and doesn't increase the point count.
	gap := math.Hypot(x-initialX, y-initialY)
	if gap > 1e-6 && len(points) > 0 {
		lastIdx := len(points) - 1
		points[lastIdx].X = initialX
		points[lastIdx].Y = initialY
		points[lastIdx].Accel = 0
		// Optionally adjust distance to reflect the chord closure; keep it
		// as the sampled distance to avoid altering physics outputs.
	}

	return points, nil
}

func telemetryTrackFromSegments(segments []trackSegment) Track {
	track := Track{Segments: make([]Segment, 0, len(segments))}
	for _, seg := range segments {
		switch seg.Type {
		case "straight":
			track.Segments = append(track.Segments, Segment{Length: seg.Length})
		case "curve":
			track.Segments = append(track.Segments, Segment{Radius: seg.Radius, Angle: seg.Angle})
		}
	}
	return track
}

func repeatTrack(track Track, laps int) Track {
	if laps <= 0 || len(track.Segments) == 0 {
		return Track{Segments: []Segment{}}
	}

	repeated := Track{Segments: make([]Segment, 0, len(track.Segments)*laps)}
	for lap := 0; lap < laps; lap++ {
		repeated.Segments = append(repeated.Segments, track.Segments...)
	}
	return repeated
}

func buildTelemetryProfiles(
	track Track,
	wraparound bool,
	stepM float64,
	gmax float64,
	cruiseCap float64,
	maxBrakeMPS2 float64,
	vMin float64,
	m float64,
	g float64,
	Crr float64,
	rho float64,
	Cd float64,
	A float64,
	theta float64,
	additionalEfficiency float64,
) (profileSet, error) {
	samples := sampleTrackMeters(track, stepM, g, gmax)
	if !wraparound || len(samples) == 0 {
		return buildProfiles(samples, cruiseCap, maxBrakeMPS2, vMin, m, g, Crr, rho, Cd, A, theta, additionalEfficiency), nil
	}

	wrappedTrack := repeatTrack(track, telemetryWrapProfileLaps)
	wrappedSamples := sampleTrackMeters(wrappedTrack, stepM, g, gmax)
	wrappedProfiles := buildProfiles(wrappedSamples, cruiseCap, maxBrakeMPS2, vMin, m, g, Crr, rho, Cd, A, theta, additionalEfficiency)

	oneLapCount := len(samples)
	start := oneLapCount
	end := 2 * oneLapCount
	if telemetryWrapProfileLaps < 3 || end > len(wrappedProfiles.Base) || end > len(wrappedProfiles.Brake) || end > len(wrappedProfiles.Coast) {
		return profileSet{}, fmt.Errorf("failed to build wraparound telemetry profiles")
	}

	return profileSet{
		Base:  append(speedProfile(nil), wrappedProfiles.Base[start:end]...),
		Brake: append(speedProfile(nil), wrappedProfiles.Brake[start:end]...),
		Coast: append(speedProfile(nil), wrappedProfiles.Coast[start:end]...),
	}, nil
}

func validateTelemetryStartSpeed(startSpeed float64) (float64, error) {
	if math.IsNaN(startSpeed) || math.IsInf(startSpeed, 0) || startSpeed < 0 {
		return 0, fmt.Errorf("invalid telemetry start speed: %v", startSpeed)
	}
	return startSpeed, nil
}

func telemetryTerminalSpeed(points []telemetryPoint) (float64, error) {
	if len(points) == 0 {
		return 0, fmt.Errorf("telemetry lap produced no points")
	}
	return validateTelemetryStartSpeed(points[len(points)-1].Speed)
}

// warmTelemetryStartSpeed repeatedly runs one-lap simulations and carries each
// lap's terminal speed into the next lap's start until the start/end speed
// difference is small or the iteration cap is reached.
func warmTelemetryStartSpeed(
	segments []trackSegment,
	initialStartSpeed float64,
	maxLaps int,
	tolerance float64,
) (float64, error) {
	return warmTelemetryStartSpeedForInputs(
		segments,
		defaultSimulationInputs(),
		initialStartSpeed,
		maxLaps,
		tolerance,
	)
}

func warmTelemetryStartSpeedForInputs(
	segments []trackSegment,
	inputs simulationInputs,
	initialStartSpeed float64,
	maxLaps int,
	tolerance float64,
) (float64, error) {
	startSpeed, err := validateTelemetryStartSpeed(initialStartSpeed)
	if err != nil {
		return 0, err
	}
	if maxLaps <= 0 {
		return startSpeed, nil
	}
	if tolerance < 0 || math.IsNaN(tolerance) || math.IsInf(tolerance, 0) {
		tolerance = telemetryWarmupTolerance
	}

	for lap := 0; lap < maxLaps; lap++ {
		points, err := buildTelemetryOneLapForInputs(segments, startSpeed, inputs)
		if err != nil {
			return 0, err
		}
		nextStartSpeed, err := telemetryTerminalSpeed(points)
		if err != nil {
			return 0, err
		}
		if math.Abs(nextStartSpeed-startSpeed) <= tolerance {
			return nextStartSpeed, nil
		}
		startSpeed = nextStartSpeed
	}

	return startSpeed, nil
}

// calculates accel at a given v
func accelAtSpeed(
	v float64,
	vMin float64,
	rWheel float64,
	Tmax float64,
	Pmax float64,
	etaDrive float64,
	m float64,
	g float64,
	Crr float64,
	rho float64,
	Cd float64,
	A float64,
	theta float64,
	additionalEfficiency float64,
) float64 {
	vEff := math.Max(v, vMin)
	pAvail := WheelPowerEV(v, Tmax, Pmax, rWheel, etaDrive)
	fDrive := pAvail / vEff
	if v < vMin && rWheel > 0 {
		fDrive = Tmax / rWheel
	}
	pRes := PowerRequired(v, m, g, Crr, rho, Cd, A, theta, additionalEfficiency)
	fRes := pRes / vEff
	return (fDrive - fRes) / m
}

// updates new speed given curent v and constant a
func updateSpeed(v float64, a float64, ds float64) float64 {
	if a == 0 {
		return v
	}
	v2 := v*v + 2*a*ds
	if v2 <= 0 {
		return 0
	}
	return math.Sqrt(v2)
}

// computers deceleratoin when no drive power
func coastDecel(
	v float64,
	vMin float64,
	m float64,
	g float64,
	Crr float64,
	rho float64,
	Cd float64,
	A float64,
	theta float64,
	additionalEfficiency float64,
) float64 {
	vEff := math.Max(v, vMin)
	pRes := PowerRequired(v, m, g, Crr, rho, Cd, A, theta, additionalEfficiency)
	fRes := pRes / vEff
	return -fRes / m
}

// brought the function from flare_sim file to here ...
func computeOptimalSpeed() float64 {
	inputs := defaultSimulationInputs()

	bestV, bestD := 0.0, 0.0
	for v := 2.0; v <= 40.0; v += 0.5 {
		if d, ok := DistanceForSpeedEV(
			v,
			inputs.BatteryWh,
			inputs.SolarWhPerMin,
			inputs.EtaDrive,
			inputs.RaceDayMin,
			inputs.RWheel,
			inputs.Tmax,
			inputs.Pmax,
			inputs.M,
			inputs.G,
			inputs.Crr,
			inputs.Rho,
			inputs.Cd,
			inputs.A,
			inputs.Theta,
			inputs.AdditionalEfficiency,
		); ok && d > bestD {
			bestD, bestV = d, v
		}
	}

	for v := math.Max(0.5, bestV-2.0); v <= bestV+2.0; v += 0.1 {
		if d, ok := DistanceForSpeedEV(
			v,
			inputs.BatteryWh,
			inputs.SolarWhPerMin,
			inputs.EtaDrive,
			inputs.RaceDayMin,
			inputs.RWheel,
			inputs.Tmax,
			inputs.Pmax,
			inputs.M,
			inputs.G,
			inputs.Crr,
			inputs.Rho,
			inputs.Cd,
			inputs.A,
			inputs.Theta,
			inputs.AdditionalEfficiency,
		); ok && d > bestD {
			bestD, bestV = d, v
		}
	}

	return bestV
}
