package main

import (
	//used for decoding JSON into distanceRequest struct
	//also for encoding struct back into JSON format for HTTP response
	"encoding/json" 
	//allows for original sim to be called via terminal using flag
	"flag"
	"fmt"
	"log"
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

//relocated main bc this is new entry point
//sim now becomes function
func main() {
	mode := flag.String("mode", "server", "mode: server or simulate") //checking for user flags for sim for server
	addr := flag.String("addr", ":8080", "server listen address") //checking flag to choose different network port in cases 8080 is in use
	flag.Parse() //fills pointers (mode and addr) with values based on terminal inputs

	if *mode == "simulate" {
		runSimulation()
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/distance", distanceHandler)

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

	if req.V <= 0 || req.BatteryWh <= 0 || req.EtaDrive <= 0 || req.RaceDayMin <= 0 ||
		req.RWheel <= 0 || req.Tmax <= 0 || req.Pmax <= 0 || req.M <= 0 || req.G <= 0 ||
		req.Crr < 0 || req.Rho <= 0 || req.Cd <= 0 || req.A <= 0 {
		writeJSON(w, http.StatusBadRequest, distanceResponse{OK: false, Message: "missing or invalid input values"})
		return
	}

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

func addCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

func writeJSON(w http.ResponseWriter, status int, payload distanceResponse) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		fmt.Fprint(w, `{"ok":false,"message":"failed to encode response"}`)
	}
}
