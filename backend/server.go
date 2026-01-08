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
	"strings"
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
		{Type: "straight", Length: 100},
		{Type: "straight", Length: 100},
		{Type: "straight", Length: 100},
		{Type: "curve", Radius: 90, Angle: 90, Direction: "right"},
		{Type: "straight", Length: 100},
		{Type: "curve", Radius: 90, Angle: 90, Direction: "right"},
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

//returns a list of points (x,y,speed,accel,distance)
func buildTelemetry(segments []trackSegment) []telemetryPoint {
	const (
		stepM    = 10.0
		gmax     = 0.8
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
	v := 0.0
	distance := 0.0
	points = append(points, telemetryPoint{X: x, Y: y, Speed: v, Accel: 0, Distance: distance})

	for _, seg := range segments {
		switch seg.Type {
		//when we are dealing with a straight segment
		case "straight":
			remaining := seg.Length
			for remaining > 0 {
				ds := math.Min(stepM, remaining) //going thru every stepM meters (10m)
				a := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
				vNext := updateSpeed(v, a, ds)
				//update position
				x += ds * math.Cos(heading)
				y += ds * math.Sin(heading)
				distance += ds
				points = append(points, telemetryPoint{X: x, Y: y, Speed: vNext, Accel: a, Distance: distance})
				v = vNext
				remaining -= ds
			}
		case "curve":
			if seg.Radius <= 0 || seg.Angle == 0 {
				continue
			}
			vCap := calcCurveSpeed(Segment{Radius: seg.Radius}, g, gmax) //compute max allowed curve speed
			if v > vCap {
				v = vCap
			}
			arcLength := seg.Radius * seg.Angle * math.Pi / 180.0
			remaining := arcLength
			isRight := strings.ToLower(seg.Direction) == "right"
			//computes data points for every 10 m over curve segment
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
				points = append(points, telemetryPoint{X: x, Y: y, Speed: v, Accel: 0, Distance: distance})
				remaining -= ds
			}
		}
	}

	return points
}

//calculates accel at a given v
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
	pRes := PowerRequired(v, m, g, Crr, rho, Cd, A, theta)
	fRes := pRes / vEff
	return (fDrive - fRes) / m
}

//updates new speed given curent v and constant a
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