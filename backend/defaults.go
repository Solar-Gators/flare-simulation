package main

const defaultPresetID = "flare-default"

type simulationInputs struct {
	V                    float64 `json:"v"`
	BatteryWh            float64 `json:"batteryWh"`
	SolarWhPerMin        float64 `json:"solarWhPerMin"`
	EtaDrive             float64 `json:"etaDrive"`
	RaceDayMin           float64 `json:"raceDayMin"`
	RWheel               float64 `json:"rWheel"`
	Tmax                 float64 `json:"tMax"`
	Pmax                 float64 `json:"pMax"`
	M                    float64 `json:"m"`
	G                    float64 `json:"g"`
	Crr                  float64 `json:"cRr"`
	Rho                  float64 `json:"rho"`
	Cd                   float64 `json:"cD"`
	A                    float64 `json:"a"`
	Theta                float64 `json:"theta"`
	Gmax                 float64 `json:"gmax"`
	AdditionalEfficiency float64 `json:"additionalEfficiency"`
}

type simulationPreset struct {
	ID     string           `json:"id"`
	Label  string           `json:"label"`
	Inputs simulationInputs `json:"inputs"`
}

type defaultsResponse struct {
	DefaultPresetID string             `json:"defaultPresetId"`
	Presets         []simulationPreset `json:"presets"`
}

var simulationPresets = []simulationPreset{
	{
		ID:    "flare-default",
		Label: "Flare (default)",
		Inputs: simulationInputs{
			V:                    20,
			BatteryWh:            5000,
			SolarWhPerMin:        5,
			EtaDrive:             0.90,
			RaceDayMin:           480,
			RWheel:               0.2792,
			Tmax:                 45,
			Pmax:                 10000,
			M:                    285,
			G:                    9.81,
			Crr:                  0.0015,
			Rho:                  1.225,
			Cd:                   0.21,
			A:                    0.456,
			Theta:                0,
			Gmax:                 0.8,
			AdditionalEfficiency: 0.0,
		},
	},
	{
		ID:    "lexus-gs350-awd",
		Label: "2008 Lexus GS350 AWD (215/55R17 road tires)",
		Inputs: simulationInputs{
			V:                    20,
			BatteryWh:            656000,
			SolarWhPerMin:        5,
			EtaDrive:             0.22,
			RaceDayMin:           480,
			RWheel:               0.334,
			Tmax:                 5725,
			Pmax:                 880000,
			M:                    1840,
			G:                    9.81,
			Crr:                  0.010,
			Rho:                  1.225,
			Cd:                   0.27,
			A:                    2.2,
			Theta:                0,
			Gmax:                 0.88,
			AdditionalEfficiency: 0.0,
		},
	},
}

func defaultSimulationPreset() simulationPreset {
	for _, preset := range simulationPresets {
		if preset.ID == defaultPresetID {
			return preset
		}
	}
	return simulationPresets[0]
}

func defaultSimulationInputs() simulationInputs {
	return defaultSimulationPreset().Inputs
}

func simulationDefaultsResponse() defaultsResponse {
	return defaultsResponse{
		DefaultPresetID: defaultSimulationPreset().ID,
		Presets:         simulationPresets,
	}
}
