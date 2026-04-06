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

	if req.V <= 0 || req.BatteryWh <= 0 || req.EtaDrive <= 0 || req.RaceDayMin <= 0 ||
		req.RWheel <= 0 || req.Tmax <= 0 || req.Pmax <= 0 || req.M <= 0 || req.G <= 0 ||
		req.Crr < 0 || req.Rho <= 0 || req.Cd <= 0 || req.A <= 0 {
		writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "missing or invalid input values"})
		return
	}
	//run sim if everything is valid
	distance, ok := DistanceForSpeedEV(
		req.V,
		req.BatteryWh, req.SolarWhPerMin, req.EtaDrive, req.RaceDayMin,
		req.RWheel, req.Tmax, req.Pmax,
		req.M, req.G, req.Crr, req.Rho, req.Cd, req.A, req.Theta,
	)
	if !ok {
		writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "inputs are not feasible for the model"})
		return
	}

	writeJSON(w, http.StatusOK, distanceResponse{DistanceM: distance, OK: true})
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
        {Type: "straight", Length: 1184.185211},
        {Type: "curve", Radius: 46.01796821 * 180.0 / (math.Pi * math.Abs(-11.73)), Angle: -11.73},
        {Type: "curve", Radius: 39.31720802 * 180.0 / (math.Pi * math.Abs(-12.04)), Angle: -12.04},
        {Type: "curve", Radius: 39.78161714 * 180.0 / (math.Pi * math.Abs(-2.86)), Angle: -2.86},
        {Type: "curve", Radius: 55.01589496 * 180.0 / (math.Pi * math.Abs(-6.57)), Angle: -6.57},
        {Type: "curve", Radius: 38.23911541 * 180.0 / (math.Pi * math.Abs(-11.32)), Angle: -11.32},
        {Type: "curve", Radius: 47.35314444 * 180.0 / (math.Pi * math.Abs(-5.21)), Angle: -5.21},
        {Type: "curve", Radius: 36.01658604 * 180.0 / (math.Pi * math.Abs(-8.01)), Angle: -8.01},
        {Type: "curve", Radius: 154.5155494 * 180.0 / (math.Pi * math.Abs(-5.55)), Angle: -5.55},
        {Type: "curve", Radius: 57.66136835 * 180.0 / (math.Pi * math.Abs(-169.24)), Angle: -169.24},
        {Type: "curve", Radius: 43.30615066 * 180.0 / (math.Pi * math.Abs(156.22)), Angle: 156.22},
        {Type: "curve", Radius: 33.93503801 * 180.0 / (math.Pi * math.Abs(-10.06)), Angle: -10.06},
        {Type: "curve", Radius: 37.78299931 * 180.0 / (math.Pi * math.Abs(-8.32)), Angle: -8.32},
        {Type: "curve", Radius: 59.57705598 * 180.0 / (math.Pi * math.Abs(-8.69)), Angle: -8.69},
        {Type: "curve", Radius: 49.14443677 * 180.0 / (math.Pi * math.Abs(-11.63)), Angle: -11.63},
        {Type: "curve", Radius: 33.91015895 * 180.0 / (math.Pi * math.Abs(-14.53)), Angle: -14.53},
        {Type: "curve", Radius: 33.74429855 * 180.0 / (math.Pi * math.Abs(-3.52)), Angle: -3.52},
        {Type: "curve", Radius: 347.0048376 * 180.0 / (math.Pi * math.Abs(-11.94)), Angle: -11.94},
        {Type: "curve", Radius: 25.26883207 * 180.0 / (math.Pi * math.Abs(-8.82)), Angle: -8.82},
        {Type: "curve", Radius: 25.45127851 * 180.0 / (math.Pi * math.Abs(-21.36)), Angle: -21.36},
        {Type: "curve", Radius: 23.06288874 * 180.0 / (math.Pi * math.Abs(-30.17)), Angle: -30.17},
        {Type: "curve", Radius: 17.42363511 * 180.0 / (math.Pi * math.Abs(-20.57)), Angle: -20.57},
        {Type: "curve", Radius: 17.39875605 * 180.0 / (math.Pi * math.Abs(-10.32)), Angle: -10.32},
        {Type: "curve", Radius: 18.50172771 * 180.0 / (math.Pi * math.Abs(-16.83)), Angle: -16.83},
        {Type: "curve", Radius: 292.2128542 * 180.0 / (math.Pi * math.Abs(-18.16)), Angle: -18.16},
        {Type: "curve", Radius: 24.24049758 * 180.0 / (math.Pi * math.Abs(3.17)), Angle: 3.17},
        {Type: "curve", Radius: 18.94125777 * 180.0 / (math.Pi * math.Abs(17.37)), Angle: 17.37},
        {Type: "curve", Radius: 14.29716655 * 180.0 / (math.Pi * math.Abs(24.87)), Angle: 24.87},
        {Type: "curve", Radius: 21.96821009 * 180.0 / (math.Pi * math.Abs(8.75)), Angle: 8.75},
        {Type: "curve", Radius: 18.94125777 * 180.0 / (math.Pi * math.Abs(23.97)), Angle: 23.97},
        {Type: "curve", Radius: 17.44851417 * 180.0 / (math.Pi * math.Abs(15.28)), Angle: 15.28},
        {Type: "curve", Radius: 99.93918452 * 180.0 / (math.Pi * math.Abs(18.66)), Angle: 18.66},
        {Type: "curve", Radius: 24.22391154 * 180.0 / (math.Pi * math.Abs(-6.99)), Angle: -6.99},
        {Type: "curve", Radius: 21.78576365 * 180.0 / (math.Pi * math.Abs(-19.6)), Angle: -19.6},
        {Type: "curve", Radius: 19.51347616 * 180.0 / (math.Pi * math.Abs(-20.02)), Angle: -20.02},
        {Type: "curve", Radius: 16.64409122 * 180.0 / (math.Pi * math.Abs(-20.45)), Angle: -20.45},
        {Type: "curve", Radius: 201.2467173 * 180.0 / (math.Pi * math.Abs(-17.9)), Angle: -17.9},
        {Type: "curve", Radius: 16.08016586 * 180.0 / (math.Pi * math.Abs(12.37)), Angle: 12.37},
        {Type: "curve", Radius: 15.24257084 * 180.0 / (math.Pi * math.Abs(12.68)), Angle: 12.68},
        {Type: "curve", Radius: 15.17622668 * 180.0 / (math.Pi * math.Abs(16.25)), Angle: 16.25},
        {Type: "curve", Radius: 14.84450587 * 180.0 / (math.Pi * math.Abs(17.3)), Angle: 17.3},
        {Type: "curve", Radius: 14.57083621 * 180.0 / (math.Pi * math.Abs(9.98)), Angle: 9.98},
        {Type: "curve", Radius: 15.98064962 * 180.0 / (math.Pi * math.Abs(19.62)), Angle: 19.62},
        {Type: "curve", Radius: 107.5355909 * 180.0 / (math.Pi * math.Abs(8.61)), Angle: 8.61},
        {Type: "curve", Radius: 72.18244644 * 180.0 / (math.Pi * math.Abs(-0.32)), Angle: -0.32},
        {Type: "curve", Radius: 59.0048376 * 180.0 / (math.Pi * math.Abs(-0.7)), Angle: -0.7},
        {Type: "curve", Radius: 21.36281963 * 180.0 / (math.Pi * math.Abs(6.43)), Angle: 6.43},
        {Type: "curve", Radius: 21.71112647 * 180.0 / (math.Pi * math.Abs(11.79)), Angle: 11.79},
        {Type: "curve", Radius: 22.83897719 * 180.0 / (math.Pi * math.Abs(20.52)), Angle: 20.52},
        {Type: "curve", Radius: 27.28403594 * 180.0 / (math.Pi * math.Abs(0.95)), Angle: 0.95},
        {Type: "curve", Radius: 26.19765031 * 180.0 / (math.Pi * math.Abs(-8.43)), Angle: -8.43},
        {Type: "curve", Radius: 17.19143055 * 180.0 / (math.Pi * math.Abs(-14.28)), Angle: -14.28},
        {Type: "curve", Radius: 18.72563925 * 180.0 / (math.Pi * math.Abs(-18.92)), Angle: -18.92},
        {Type: "curve", Radius: 47.170698 * 180.0 / (math.Pi * math.Abs(-18.86)), Angle: -18.86},
        {Type: "curve", Radius: 165.4955079 * 180.0 / (math.Pi * math.Abs(-5.33)), Angle: -5.33},
        {Type: "curve", Radius: 28.3870076 * 180.0 / (math.Pi * math.Abs(-3.96)), Angle: -3.96},
        {Type: "curve", Radius: 11.86731168 * 180.0 / (math.Pi * math.Abs(-12.26)), Angle: -12.26},
        {Type: "curve", Radius: 14.47961299 * 180.0 / (math.Pi * math.Abs(-25.46)), Angle: -25.46},
        {Type: "curve", Radius: 20.11886662 * 180.0 / (math.Pi * math.Abs(-8.48)), Angle: -8.48},
        {Type: "curve", Radius: 32.93158258 * 180.0 / (math.Pi * math.Abs(-13.61)), Angle: -13.61},
        {Type: "curve", Radius: 551.1458189 * 180.0 / (math.Pi * math.Abs(-3.12)), Angle: -3.12},
        {Type: "curve", Radius: 26.47131997 * 180.0 / (math.Pi * math.Abs(-21.38)), Angle: -21.38},
        {Type: "curve", Radius: 19.02418798 * 180.0 / (math.Pi * math.Abs(-22.38)), Angle: -22.38},
        {Type: "curve", Radius: 21.52868003 * 180.0 / (math.Pi * math.Abs(-13.68)), Angle: -13.68},
        {Type: "curve", Radius: 19.78714582 * 180.0 / (math.Pi * math.Abs(-19.39)), Angle: -19.39},
        {Type: "curve", Radius: 29.14996545 * 180.0 / (math.Pi * math.Abs(-14.86)), Angle: -14.86},
        {Type: "curve", Radius: 36 * 180.0 / (math.Pi * math.Abs(-16.12)), Angle: -16.12},
        {Type: "curve", Radius: 42.00414651 * 180.0 / (math.Pi * math.Abs(-6.48)), Angle: -6.48},
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
	if !wraparound {
		return buildTelemetryOneLapWithWraparound(segments, defaultTelemetryStartSpeed, false)
	}

	startSpeed, err := warmTelemetryStartSpeed(
		segments,
		defaultTelemetryStartSpeed,
		telemetryWarmupMaxLaps,
		telemetryWarmupTolerance,
	)
	if err != nil {
		return nil, err
	}
	return buildTelemetryOneLapWithWraparound(segments, startSpeed, true)
}

// buildTelemetryOneLap simulates a single lap and starts from the provided speed.
// This is the primitive needed for later wraparound support, where one lap warms up
// the state and the next lap starts from the previous lap's terminal speed.
func buildTelemetryOneLap(segments []trackSegment, startSpeed float64) ([]telemetryPoint, error) {
	return buildTelemetryOneLapWithWraparound(segments, startSpeed, true)
}

// buildTelemetryOneLapWithWraparound simulates one lap and chooses whether the
// brake/coast profiles should look only within the current lap or continue
// across the finish line into the next lap.
func buildTelemetryOneLapWithWraparound(
	segments []trackSegment,
	startSpeed float64,
	wraparound bool,
) ([]telemetryPoint, error) {
	const (
		stepM    = 1.0
		muTire   = 0.9
		maxSpeed = 40.0
		vMin     = 0.5
	)
	inputs := defaultSimulationInputs()

	track := Track{Segments: make([]Segment, 0, len(segments))}
	for _, seg := range segments {
		switch seg.Type {
		case "straight":
			track.Segments = append(track.Segments, Segment{Length: seg.Length})
		case "curve":
			track.Segments = append(track.Segments, Segment{Radius: seg.Radius, Angle: seg.Angle})
		}
	}
	samples := sampleTrackMeters(track, stepM, inputs.G, inputs.Gmax)
	cruiseCap := math.Min(maxSpeed, optimalCruiseSpeed)
	if cruiseCap <= 0 {
		cruiseCap = maxSpeed
	}
	profiles := buildProfiles(
		samples,
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
	)

	points := make([]telemetryPoint, 0, 64)
	x, y, heading := 0.0, 0.0, 0.0
	startSpeed, err := validateTelemetryStartSpeed(startSpeed)
	if err != nil {
		return nil, err
	}
	v := startSpeed
	distance := 0.0
	profileIdx := 0
	points = append(points, telemetryPoint{X: x, Y: y, Speed: v, Accel: 0, Distance: distance})

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
) (profileSet, error) {
	samples := sampleTrackMeters(track, stepM, g, gmax)
	if !wraparound || len(samples) == 0 {
		return buildProfiles(samples, cruiseCap, maxBrakeMPS2, vMin, m, g, Crr, rho, Cd, A, theta), nil
	}

	wrappedTrack := repeatTrack(track, telemetryWrapProfileLaps)
	wrappedSamples := sampleTrackMeters(wrappedTrack, stepM, g, gmax)
	wrappedProfiles := buildProfiles(wrappedSamples, cruiseCap, maxBrakeMPS2, vMin, m, g, Crr, rho, Cd, A, theta)

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
		points, err := buildTelemetryOneLap(segments, startSpeed)
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
) float64 {
	vEff := math.Max(v, vMin)
	pAvail := WheelPowerEV(v, Tmax, Pmax, rWheel, etaDrive)
	fDrive := pAvail / vEff
	if v < vMin && rWheel > 0 {
		fDrive = Tmax / rWheel
	}
	pRes := PowerRequired(v, m, g, Crr, rho, Cd, A, theta)
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
) float64 {
	vEff := math.Max(v, vMin)
	pRes := PowerRequired(v, m, g, Crr, rho, Cd, A, theta)
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
		); ok && d > bestD {
			bestD, bestV = d, v
		}
	}

	return bestV
}
