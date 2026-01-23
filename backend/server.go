package main

import (
	//used for decoding JSON into distanceRequest struct
	//also for encoding struct back into JSON format for HTTP response
	"encoding/json"
	//allows for original sim to be called via terminal using flag
	"flag"
	"fmt"
	"log"
	"math"
	"net/http" //lets go program talk over web --> Receive requests and send responses
)

type distanceRequest struct {
	//these are all struct tags
	//tells go how struct maps to JSON for encoding and decoding

	V             float64 `json:"v"`
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
}

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
	mux := http.NewServeMux()                    //request router (empty --> no route to go), serve multiplexer -->takes http requests and routes it
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

	var req distanceRequest        // holds parsed JSON
	dec := json.NewDecoder(r.Body) //decode JSON and read
	dec.DisallowUnknownFields()    //decoding will fail if JSON has fields that are not valid
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

	segments := defaultTrackSegments()
	points := buildTelemetry(segments)
	writeJSON(w, http.StatusOK, telemetryResponse{Points: points})
}

// returns a list of points (x,y,speed,accel,distance)
// no more speed "snap" in curves as in instead of setting v to the V capacity
// the V is now ramp towards the cap given acceleration limits
// ++ included friction-circle limit bc in curves lateral acceleration uses grip which reduces how much longitudinal accel/brake you can apply
// we have target cruise speed to prevent the car from accelerating forever if V is below we accelerate and if above we brake
// fixed our issue of V approaching and reaching 0 bc of curves by removing continuous coasting decel curves and replacing by controlled approach to target curve speed
// we essentially established a baseline for the optimal speed around curve instead of always taking foot off gas when approaching curve.
func buildTelemetry(segments []trackSegment) []telemetryPoint {
	const (
		stepM    = 1.0
		gmax     = 0.8
		muTire   = 0.9
		brakePct = 0.95
		maxSpeed = 40.0
		vMin     = 0.5
		A        = 0.456
		Cd       = 0.21
		rho      = 1.225
		Crr      = 0.0015
		m        = 285.0
		g        = 9.81
		theta    = 0.0
		rWheel   = 0.2792
		Tmax     = 45.0
		Pmax     = 10000.0
		etaDrive = 0.90
	)

	points := make([]telemetryPoint, 0, 64)
	x, y, heading := 0.0, 0.0, 0.0
	v := 0.5
	distance := 0.0
	points = append(points, telemetryPoint{X: x, Y: y, Speed: v, Accel: 0, Distance: distance})

	baseTarget := maxSpeed
	if optimalCruiseSpeed > 0 {
		baseTarget = optimalCruiseSpeed
	}
	if maxSpeed > 0 && baseTarget > maxSpeed {
		baseTarget = maxSpeed
	}

	for i, seg := range segments {
		switch seg.Type {
		//when we are dealing with a straight segment
		case "straight":
			nextCurveCap := 0.0
			//if next segment is a curve find max curve speed
			if i+1 < len(segments) && segments[i+1].Type == "curve" && segments[i+1].Radius > 0 {
				nextCurveCap = math.Sqrt(gmax * g * segments[i+1].Radius)
			}
			remaining := seg.Length
			for remaining > 0 {
				ds := math.Min(stepM, remaining) //going thru every stepM meters (10m)
				targetSpeed := baseTarget
				if nextCurveCap > 0 && v > nextCurveCap*brakePct {
					targetSpeed = math.Min(targetSpeed, nextCurveCap)
				}
				aLongMax := muTire * g
				var a float64
				if v > targetSpeed {
					aReq := (targetSpeed*targetSpeed - v*v) / (2 * remaining)
					a = math.Max(aReq, -aLongMax)
				} else if v < targetSpeed {
					aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
					a = math.Min(aPower, aLongMax)
				} else {
					a = 0
				}
				vNext := updateSpeed(v, a, ds)
				if vNext > targetSpeed {
					vNext = targetSpeed
				}
				//update position
				x += ds * math.Cos(heading)
				y += ds * math.Sin(heading)
				distance += ds
				points = append(points, telemetryPoint{X: x, Y: y, Speed: vNext, Accel: a, Distance: distance})
				log.Printf("speed=%.2f accel=%.3f", v, a)
				v = vNext
				remaining -= ds
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
			aLatMax := gmax * g
			vCap := math.Sqrt(aLatMax * seg.Radius)
			targetSpeed := math.Min(vCap, baseTarget)
			if maxSpeed > 0 && targetSpeed > maxSpeed {
				targetSpeed = maxSpeed
			}
			angleDeg := seg.Angle
			arcLength := seg.Radius * math.Abs(angleDeg) * math.Pi / 180.0
			remaining := arcLength
			isRight := angleDeg < 0
			//computes data points for each step along curve segment
			for remaining > 0 {
				ds := math.Min(stepM, remaining)
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
				aTotalMax := muTire * g
				aLongMax := math.Sqrt(math.Max(0, aTotalMax*aTotalMax-aLat*aLat))
				var a float64
				if v > targetSpeed {
					aReq := (targetSpeed*targetSpeed - v*v) / (2 * remaining)
					a = math.Max(aReq, -aLongMax)
				} else if v < targetSpeed {
					aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
					a = math.Min(aPower, aLongMax)
				} else {
					a = 0
				}
				vNext := updateSpeed(v, a, ds)
				if vNext > targetSpeed {
					vNext = targetSpeed
				}
				points = append(points, telemetryPoint{X: x, Y: y, Speed: vNext, Accel: a, Distance: distance})
				v = vNext
				remaining -= ds
			}
		}
	}

	return points
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
	const (
		A             = 0.456
		Cd            = 0.21
		rho           = 1.225
		Crr           = 0.0015
		m             = 285.0
		g             = 9.81
		theta         = 0.0
		rWheel        = 0.2792
		Tmax          = 45.0
		Pmax          = 10000.0
		batteryWh     = 5000.0
		solarWhPerMin = 5.0
		etaDrive      = 0.90
		raceDayMin    = 480.0
	)

	bestV, bestD := 0.0, 0.0
	for v := 2.0; v <= 40.0; v += 0.5 {
		if d, ok := DistanceForSpeedEV(v, batteryWh, solarWhPerMin, etaDrive, raceDayMin,
			rWheel, Tmax, Pmax, m, g, Crr, rho, Cd, A, theta); ok && d > bestD {
			bestD, bestV = d, v
		}
	}

	for v := math.Max(0.5, bestV-2.0); v <= bestV+2.0; v += 0.1 {
		if d, ok := DistanceForSpeedEV(v, batteryWh, solarWhPerMin, etaDrive, raceDayMin,
			rWheel, Tmax, Pmax, m, g, Crr, rho, Cd, A, theta); ok && d > bestD {
			bestD, bestV = d, v
		}
	}

	return bestV
}
