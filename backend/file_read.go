package main
import(
	"strconv"
	"os"
)
func WriteStepStatstoCSV(time float64, distance float64, energyLeft float64) {
	w, err := os.Open("StepStats.csv")
	if err != nil {
		return
	}
	timeOutput := (string(strconv.FormatFloat(time, 'f', -1, 64)))
	distanceOutput := (string(strconv.FormatFloat(distance, 'f', -1, 64)))
	energyOutput := (string(strconv.FormatFloat(energyLeft, 'f', -1, 64)))
	Output := (timeOutput + ", " + distanceOutput + ", " + energyOutput + "\n")
	w.WriteString(Output)
}

func ClearStepStatstoCSV() {
    w, err := os.OpenFile("StepStats.csv", os.O_WRONLY|os.O_TRUNC, 0644)

    if err != nil {
        return
    }

    w.WriteString("Time (s),Distance (m),Energy Left (Wh)\n")
}


