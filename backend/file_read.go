package main

import (
	"os"
	"strconv"
)

func WriteStepStatstoCSV(time float64, distance float64, energyLeft float64) {
	w, err := os.OpenFile("data/StepStats.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	timeOutput := (string(strconv.FormatFloat(time, 'f', -1, 64)))
	distanceOutput := (string(strconv.FormatFloat(distance, 'f', -1, 64)))
	energyOutput := (string(strconv.FormatFloat(energyLeft, 'f', -1, 64)))
	Output := (timeOutput + ", " + distanceOutput + ", " + energyOutput + "\n")
	w.WriteString(Output)
}

func ClearStepStatstoCSV(s string /*input 3 variables to be measured, in the format "Variable Name (units), ___, ___"*/) {
	w, err := os.OpenFile("data/StepStats.csv", os.O_WRONLY|os.O_TRUNC, 0644)

	if err != nil {
		return
	}

	w.WriteString(s + "\n")
}
