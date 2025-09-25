package main

import "math"

func totalEnergy(solarYield float64, /* watt hours / minute */
	raceDayTime float64, /* in minutes */
	batterySize float64 /* in watt hours */) float64 {
	totalBattery := (solarYield * raceDayTime) + batterySize
	return totalBattery
}

func tractionForce(engineTorque float64,
	gearRatio float64,
	differentialRatio float64,
	drivetrainEfficiency float64,
	rollingRadius float64) float64 {
	return (engineTorque * gearRatio * differentialRatio * drivetrainEfficiency) / rollingRadius
}

func WheelPowerEV(v, Tmax, Pmax, rWheel, eta float64) float64 {
	if v <= 0 || Tmax <= 0 || Pmax <= 0 || rWheel <= 0 || eta <= 0 {
		return 0
	}
	vb := (Pmax / eta) / (Tmax / rWheel)
	if v <= vb {
		return eta * Tmax * (v / rWheel)
	}
	return eta * Pmax
}

func WheelPowerICE(
	v, gearRatio, finalDrive, rWheel, eta float64,
	torqueAtRPM func(rpm float64) float64,
) float64 {
	if v <= 0 || gearRatio <= 0 || finalDrive <= 0 || rWheel <= 0 || eta <= 0 || torqueAtRPM == nil {
		return 0
	}
	rpm := (v / rWheel) * gearRatio * finalDrive * 60.0 / (2.0 * math.Pi)
	Te := torqueAtRPM(rpm)

	if Te <= 0 {
		return 0
	}

	Twheel := Te * gearRatio * finalDrive * eta
	omegaWheel := v / rWheel
	return Twheel * omegaWheel
}

// usable energy is distributed across average tractive energy per m
// func expectedSpeed(time float64, energy float64, rollingResistance float64,
// 	drivetrainLosses float64, drag float64, mass float64, frontalArea float64, velocity float64) float64 {

// 	resistance_force := rollingResistance*mass*9.8 + 0.5*1.225*drag*frontalArea*velocity*velocity
// 	return (energy * 3600 / resistance_force) / drivetrainLosses
// }

func expectedDistance(
	energy, rollingResistance, drivetrainLosses, drag, mass, frontalArea, velocity float64,
) float64 {
	// Convert pack Wh to wheel Joules.
	// If "drivetrainLosses" actually means efficiency (η), use it directly.
	// If it means fractional loss (e.g., 0.08 for 8% loss), change to: etaDrive := 1.0 - drivetrainLosses
	etaDrive := drivetrainLosses // treat as efficiency 0..1

	EJwheel := energy * 3600.0 * etaDrive

	rho := 1.225 // kg/m^3 (sea level, ~15°C); OK to keep constant for a simplified model
	g := 9.81

	// Tractive force at constant speed on flat road
	F := rollingResistance*mass*g + 0.5*rho*drag*frontalArea*velocity*velocity
	if F <= 0 {
		return 0
	}
	return EJwheel / F // meters
}

// func distributedEnergy(energy float64, distanceInMeters float64) float64 {
// 	// hi
// 	return 0.0
// }

func EnergyPerMeter_J(
	vAir, m, g, Crr, rho, Cd, A, theta, etaDrive float64,
) float64 {
	Froll := Crr * m * g
	Fgrade := m * g * math.Sin(theta)
	Faero := 0.5 * rho * Cd * A * vAir * vAir
	return (Froll + Fgrade + Faero) / etaDrive // J/m
}

func WhPerKmAt(
	vAir, m, g, Crr, rho, Cd, A, theta, etaDrive float64,
) float64 {
	eJm := EnergyPerMeter_J(vAir, m, g, Crr, rho, Cd, A, theta, etaDrive)
	return eJm * 0.27778 // Wh/km
}
