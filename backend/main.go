package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
)
//I'm gonna end it all
type SegmentType string

const (
	Straight SegmentType = "straight"
	Turn     SegmentType = "turn"
)

type Segment struct {
	Type      SegmentType
	Length    float64 // meters
	Radius    float64 // for turns; 0 for straight
	Elevation float64 // change in elevation, optional
}

var track = []Segment{
	{Type: Straight, Length: 150},        // long straight
	{Type: Turn, Length: 40, Radius: 20}, // wide left turn
	{Type: Straight, Length: 80},         // short straight
	{Type: Turn, Length: 30, Radius: 10}, // sharp right turn
	{Type: Straight, Length: 200},        // long straight
	{Type: Turn, Length: 50, Radius: 25}, // moderate left turn
	{Type: Straight, Length: 120},        // medium straight
	{Type: Turn, Length: 35, Radius: 15}, // moderate right turn
	{Type: Straight, Length: 180},        // final straight
}

type StrategyInput struct {
	MaxSpeed  float64 // m/s
	MaxGForce float64 // in G (convert to m/s²)
	SolarRate float64 // in Wh/min
}

//this will come from real world test data

func AccelAtSpeed(speed float64) float64 {
	return 1.5 // placeholder: m/s²
}

func CoastDecelAtSpeed(speed float64) float64 {
	return -0.8 // placeholder: m/s²
}

func EnergyUsage(accel float64) float64 {
	return 5.0 * accel // placeholder: Wh/s
}

func SimulateStrategy(track []Segment, input StrategyInput) []DataPoint {
	var data []DataPoint
	currentSpeed := 0.0 // could be what ever if car a running start?
	totalDistance := 0.0
	totalEnergy := 0.0
	totalTime := 0.0

	for _, segment := range track {
		totalSegmentDistance := segment.Length
		segmentDistanceUsed := 0.0

		// ------ 1. Coast if approaching a turn ------
		var distanceCoast float64
		if segment.Type == Turn {
			maxTurnSpeed := math.Sqrt(input.MaxGForce * 9.81 * segment.Radius)
			if currentSpeed > maxTurnSpeed {
				decel := CoastDecelAtSpeed(currentSpeed)
				timeToSlow := (currentSpeed - maxTurnSpeed) / -decel
				distanceCoast = currentSpeed*timeToSlow + 0.5*decel*timeToSlow*timeToSlow

				if distanceCoast > totalSegmentDistance {
					distanceCoast = totalSegmentDistance
					timeToSlow = (-currentSpeed + math.Sqrt(currentSpeed*currentSpeed-2*decel*distanceCoast)) / decel
				}

				segmentDistanceUsed += distanceCoast
				totalDistance += distanceCoast
				totalTime += timeToSlow
				currentSpeed = maxTurnSpeed

				data = append(data, DataPoint{totalDistance, currentSpeed, totalTime, 0, segment.Type})

			}
		}

		// ------ 2. Accelerate if below MaxSpeed ------
		var distanceAccel float64
		if currentSpeed < input.MaxSpeed {
			targetSpeed := input.MaxSpeed
			accel := AccelAtSpeed(currentSpeed)
			deltaV := targetSpeed - currentSpeed
			timeToAccel := deltaV / accel
			distanceAccel = currentSpeed*timeToAccel + 0.5*accel*timeToAccel*timeToAccel

			remaining := totalSegmentDistance - segmentDistanceUsed
			if distanceAccel > remaining {
				// Not enough room to reach max speed
				distanceAccel = remaining
				// Solve v² = u² + 2as → finalSpeed
				targetSpeed = math.Sqrt(currentSpeed*currentSpeed + 2*accel*distanceAccel)
				timeToAccel = (targetSpeed - currentSpeed) / accel
			}

			energyUsed := EnergyUsage(accel) * timeToAccel
			segmentDistanceUsed += distanceAccel
			totalDistance += distanceAccel
			totalTime += timeToAccel
			totalEnergy += energyUsed

			data = append(data, DataPoint{totalDistance, targetSpeed, totalTime, energyUsed, segment.Type})

		}

		// ------ 3. Cruise if there's remaining segment ------
		remaining := totalSegmentDistance - segmentDistanceUsed
		if remaining > 0 {
			cruiseSpeed := math.Min(currentSpeed, input.MaxSpeed)
			if cruiseSpeed <= 0 {
				// Log warning or skip cruising at 0 speed
				fmt.Println("⚠️  Skipping cruise segment due to 0 speed.")
				continue
			}

			time := remaining / cruiseSpeed
			energyUsed := EnergyUsage(0) * time

			segmentDistanceUsed += remaining
			totalDistance += remaining
			totalTime += time
			totalEnergy += energyUsed
			data = append(data, DataPoint{totalDistance, cruiseSpeed, totalTime, energyUsed, segment.Type})

			currentSpeed = cruiseSpeed
		}
	}

	return data
}

type DataPoint struct {
	Distance    float64
	Speed       float64
	Time        float64
	Energy      float64
	SegmentType SegmentType
}

func WriteCSV(data []DataPoint, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Segment", "Type", "Distance (m)", "Speed (m/s)", "Energy (Wh)"})

	for i, dp := range data {
		writer.Write([]string{
			strconv.Itoa(i + 1),
			string(dp.SegmentType),
			strconv.FormatFloat(dp.Distance, 'f', 2, 64),
			strconv.FormatFloat(dp.Speed, 'f', 2, 64),
			strconv.FormatFloat(dp.Energy, 'f', 2, 64),
		})
	}

	return nil
}

func handleSimulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // for local dev
	w.Header().Set("Content-Type", "application/json")
	input := StrategyInput{
		MaxSpeed:  20.0,
		MaxGForce: 1.5,
		SolarRate: 200.0,
	}
	result := SimulateStrategy(track, input)
	fmt.Printf("%-10s %-10s %-12s %-12s %-10s %-12s\n", "Segment", "Type", "Distance (m)", "Speed (m/s)", "Energy (Wh)", "Time (T)")
	for i, dp := range result {
		fmt.Printf("%-10d %-10s %-12.2f %-12.2f %-10.2f %-12.2f\n",
			i+1, dp.SegmentType, dp.Distance, dp.Speed, dp.Energy, dp.Time)
	}

	if err := WriteCSV(result, "simulation_output.csv"); err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {

	http.HandleFunc("/simulate", handleSimulate)
	fmt.Println("🚀 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
