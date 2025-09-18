package main

func totalEnergy(solarYield float64, /* watt hours / minute */
	raceDayTime float64, /* in minutes */
	batterySize float64 /* in watt hours */) float64 {
	totalBattery := (solarYield * raceDayTime) + batterySize
	return totalBattery
}

// usable energy is distributed across average tractive energy per m
func expectedSpeed(time float64, energy float64, rollingResistance float64,
	drivetrainLosses float64, drag float64, mass float64, frontalArea float64, velocity float64) float64 {

	resistance_force := rollingResistance*mass*9.8 + 0.5*1.225*drag*frontalArea*velocity*velocity
	return (energy * 3600 / resistance_force) / drivetrainLosses
}

func expectedDistance(energy float64, rollingResistance float64,
	drivetrainLosses float64, drag float64, mass float64, frontalArea float64, velocity float64) float64 {

	resistance_force := rollingResistance*mass*9.8 + 0.5*1.225*drag*frontalArea*velocity*velocity
	return (energy * 3600 / resistance_force) / drivetrainLosses
}

func distributedEnergy(energy float64, distanceInMeters float64) float64 {
	// hi
	return 0.0
}
