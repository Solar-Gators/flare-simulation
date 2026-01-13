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
		{Type: "straight", Length: 637.1769912},
		{Type: "curve", Radius: 15.81415929, Angle: -17.4},
		{Type: "curve", Radius: 12.30088496, Angle: -19.79},
		{Type: "curve", Radius: 28.07079646, Angle: -10.89},
		{Type: "curve", Radius: 21.88495575, Angle: 7.45},
		{Type: "curve", Radius: 15.18584071, Angle: 22.74},
		{Type: "curve", Radius: 14.7699115, Angle: 8.72},
		{Type: "curve", Radius: 15.96460177, Angle: 23.36},
		{Type: "curve", Radius: 0, Angle: 12.76},
		{Type: "straight", Length: 100.9646018},
		{Type: "curve", Radius: 39.30973451, Angle: -23.95},
		{Type: "curve", Radius: 0, Angle: -16.92},
		{Type: "straight", Length: 138.460177},
		{Type: "curve", Radius: 9.159292035, Angle: -21.3},
		{Type: "curve", Radius: 11.30088496, Angle: -23.08},
		{Type: "curve", Radius: 11.49557522, Angle: -33.67},
		{Type: "curve", Radius: 24.34513274, Angle: -14.11},
		{Type: "curve", Radius: 22.22123894, Angle: -4.36},
		{Type: "curve", Radius: 20.59292035, Angle: -13.53},
		{Type: "curve", Radius: 23.53097345, Angle: -14.95},
		{Type: "curve", Radius: 0, Angle: -19.1},
		{Type: "straight", Length: 260.2920354},
		{Type: "curve", Radius: 31.74336283, Angle: -7.41},
		{Type: "curve", Radius: 55.48672566, Angle: -9.15},
		{Type: "curve", Radius: 73.62831858, Angle: 1.93},
		{Type: "curve", Radius: 65.7079646, Angle: 8.65},
		{Type: "curve", Radius: 86.44247788, Angle: 10.82},
		{Type: "curve", Radius: 71.31858407, Angle: -8.69},
		{Type: "curve", Radius: 49.55752212, Angle: -19.78},
		{Type: "curve", Radius: 53.30973451, Angle: -18.02},
		{Type: "curve", Radius: 37.69026549, Angle: -17.61},
		{Type: "curve", Radius: 36.83185841, Angle: -7.12},
		{Type: "curve", Radius: 49.0619469, Angle: -9.23},
		{Type: "curve", Radius: 45.02654867, Angle: -1.86},
		{Type: "curve", Radius: 17.40707965, Angle: -37.78},
		{Type: "curve", Radius: 13.51327434, Angle: -25.04},
		{Type: "curve", Radius: 18.87610619, Angle: -13.24},
		{Type: "curve", Radius: 18.27433628, Angle: -27.45},
		{Type: "curve", Radius: 32.94690265, Angle: -6.38},
		{Type: "curve", Radius: 30.43362832, Angle: -15.07},
		{Type: "curve", Radius: 39.66371681, Angle: -4.52},
		{Type: "curve", Radius: 20.11504425, Angle: -33.89},
		{Type: "curve", Radius: 30.85840708, Angle: -22.64},
		{Type: "curve", Radius: 64.62831858, Angle: -2.53},
		{Type: "curve", Radius: 27.54867257, Angle: 30.49},
		{Type: "curve", Radius: 18.34513274, Angle: 20.1},
		{Type: "curve", Radius: 18.31858407, Angle: 17.54},
		{Type: "curve", Radius: 21.84070796, Angle: 12.57},
		{Type: "curve", Radius: 24.2920354, Angle: 19.01},
		{Type: "curve", Radius: 29.01769912, Angle: 11.44},
		{Type: "curve", Radius: 27.24778761, Angle: -17.92},
		{Type: "curve", Radius: 34.92035398, Angle: -16.36},
		{Type: "curve", Radius: 32.80530973, Angle: -8.94},
		{Type: "curve", Radius: 72.51327434, Angle: -10.59},
		{Type: "curve", Radius: 37.20353982, Angle: 9.27},
		{Type: "curve", Radius: 27.15044248, Angle: 11.04},
		{Type: "curve", Radius: 41.02654867, Angle: 7.59},
		{Type: "curve", Radius: 26.7699115, Angle: 26.76},
		{Type: "curve", Radius: 35.66371681, Angle: 16.22},
		{Type: "curve", Radius: 14.52212389, Angle: 60.88},
		{Type: "curve", Radius: 21.21238938, Angle: 35.95},
		{Type: "curve", Radius: 87.87610619, Angle: 18.81},
		{Type: "curve", Radius: 32.60176991, Angle: 11.02},
		{Type: "curve", Radius: 66.18584071, Angle: 5.8},
		{Type: "curve", Radius: 13.99115044, Angle: -31.79},
		{Type: "curve", Radius: 0, Angle: -16.18},
		{Type: "straight", Length: 116.1415929},
		{Type: "curve", Radius: 11.84955752, Angle: 13.52},
		{Type: "curve", Radius: 0, Angle: 17.62},
		{Type: "straight", Length: 192.8318584},
		{Type: "curve", Radius: 41.42477876, Angle: 3.19},
		{Type: "curve", Radius: 79.50442478, Angle: 12.45},
		{Type: "curve", Radius: 0, Angle: 3.09},
		{Type: "straight", Length: 228.2654867},
		{Type: "curve", Radius: 34.05309735, Angle: 10.94},
		{Type: "curve", Radius: 98.88495575, Angle: 8.92},
		{Type: "curve", Radius: 36.42477876, Angle: -17.64},
		{Type: "curve", Radius: 48.53982301, Angle: -24.99},
		{Type: "curve", Radius: 42.49557522, Angle: -14.52},
		{Type: "curve", Radius: 49.87610619, Angle: -13.58},
		{Type: "curve", Radius: 49.80530973, Angle: -15.14},
		{Type: "curve", Radius: 43.71681416, Angle: -8.58},
		{Type: "curve", Radius: 17.61946903, Angle: -24.65},
		{Type: "curve", Radius: 26.98230088, Angle: -23.13},
		{Type: "curve", Radius: 36.26548673, Angle: -12.81},
		{Type: "curve", Radius: 72.16814159, Angle: -10.92},
		{Type: "curve", Radius: 57.46017699, Angle: -6.33},
		{Type: "curve", Radius: 27.45132743, Angle: -33.22},
		{Type: "curve", Radius: 32.04424779, Angle: -28.76},
		{Type: "curve", Radius: 27.46902655, Angle: -32.22},
		{Type: "curve", Radius: 23.44247788, Angle: -24.13},
		{Type: "curve", Radius: 19.38053097, Angle: -18.12},
		{Type: "curve", Radius: 17.68141593, Angle: -26.56},
		{Type: "curve", Radius: 67.75221239, Angle: -29.07},
		{Type: "curve", Radius: 29.53097345, Angle: 33.99},
		{Type: "curve", Radius: 21.66371681, Angle: 23.77},
		{Type: "curve", Radius: 28.96460177, Angle: 27.99},
		{Type: "curve", Radius: 28.15044248, Angle: 21.02},
		{Type: "curve", Radius: 24.13274336, Angle: 33.61},
		{Type: "curve", Radius: 22.91150442, Angle: 25.26},
		{Type: "curve", Radius: 56.21238938, Angle: 37.86},
		{Type: "curve", Radius: 17.45132743, Angle: -12.67},
		{Type: "curve", Radius: 14.99115044, Angle: -25.63},
		{Type: "curve", Radius: 53.59292035, Angle: -9.99},
		{Type: "curve", Radius: 19.89380531, Angle: 9.51},
		{Type: "curve", Radius: 75.7699115, Angle: 29.52},
		{Type: "curve", Radius: 31.82300885, Angle: -16.17},
		{Type: "curve", Radius: 22.95575221, Angle: -10.85},
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
func buildTelemetry(segments []trackSegment) []telemetryPoint {
	const (
		stepM    = 5.0
		gmax     = 0.8
		muBrake  = 0.9
		brakePct = 0.95
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

	for i, seg := range segments {
		switch seg.Type {
		//when we are dealing with a straight segment
		case "straight":
			nextCurveCap := 0.0
			//if next segment is a curve find max curve speed
			if i+1 < len(segments) && segments[i+1].Type == "curve" && segments[i+1].Radius > 0 {
				nextCurveCap = calcCurveSpeed(Segment{Radius: segments[i+1].Radius}, g, gmax)
			}
			remaining := seg.Length
			for remaining > 0 {
				ds := math.Min(stepM, remaining) //going thru every stepM meters (10m)
				var a float64
				//if there is an upcoming curve and are faster than threshold, begin slowing
				if nextCurveCap > 0 && v > nextCurveCap*brakePct && remaining > 0 {
					a = coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)         //negative acceleratoin
					aBrake := (nextCurveCap*nextCurveCap - v*v) / (2 * remaining) //calculate braking decel needed
					//if braking is needed, clamp it to what the brakes can do
					if aBrake < 0 {
						maxBrake := -muBrake * g
						a = math.Min(a, math.Max(aBrake, maxBrake))
					}
				} else {
					a = accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
					if optimalCruiseSpeed > 0 && v >= optimalCruiseSpeed {
						a = math.Min(a, 0)
					}
				}
				vNext := updateSpeed(v, a, ds)
				if optimalCruiseSpeed > 0 && vNext > optimalCruiseSpeed {
					vNext = optimalCruiseSpeed
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
			vCap := calcCurveSpeed(Segment{Radius: seg.Radius}, g, gmax) //compute max allowed curve speed
			if v > vCap {
				v = vCap
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
				a := coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
				vNext := updateSpeed(v, a, ds)
				if vNext > vCap {
					vNext = vCap
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
