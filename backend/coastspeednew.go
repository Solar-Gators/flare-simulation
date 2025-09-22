package main

import "math"

func newTotalEnergy(solarYield float64, /* watt hours / minute */
	raceDayTime float64, /* in minutes */
	batterySize float64 /* in watt hours */) float64 {
	totalBattery := (solarYield * raceDayTime) + batterySize
	return totalBattery
}

// PowerRequired returns tractive power needed at road speed v [m/s]
// Include grade theta [rad]. If you want wind, pass v as air speed (v_ground - v_wind).
func PowerRequired(v, m, g, Crr, rho, Cd, A, theta float64) float64 {
	return (Crr*m*g+m*g*math.Sin(theta))*v + 0.5*rho*Cd*A*v*v*v
}

// SolveCruise finds v where Pwheel(v) = Preq(v) using bisection.
// vMin, vMax in m/s (e.g., 0.1..120/3.6). Returns (v, ok).
func SolveCruise(
	Pwheel func(v float64) float64,
	Preq func(v float64) float64,
	vMin, vMax float64,
) (float64, bool) {
	f := func(v float64) float64 { return Pwheel(v) - Preq(v) }

	// Expand right bound if needed (up to ~400 km/h)
	for f(vMax) > 0 && vMax < 400.0/3.6 {
		vMax *= 1.5
	}
	// If no sign change, no solution in bracket
	if f(vMin)*f(vMax) > 0 {
		return math.NaN(), false
	}
	// Bisection
	for i := 0; i < 64; i++ {
		vm := 0.5 * (vMin + vMax)
		if f(vm) > 0 {
			vMin = vm
		} else {
			vMax = vm
		}
	}
	return 0.5 * (vMin + vMax), true
}

func CruiseSpeedEV(
	m, g, Crr, rho, Cd, A float64,
	rWheel, Tmax, Pmax, eta, theta float64, // theta [rad]
) (float64, bool) {
	Preq := func(v float64) float64 {
		return PowerRequired(v, m, g, Crr, rho, Cd, A, theta)
	}
	Pwheel := func(v float64) float64 {
		return WheelPowerEV(v, Tmax, Pmax, rWheel, eta)
	}
	// search 0.1 m/s .. 120 km/h; expands automatically if needed
	return SolveCruise(Pwheel, Preq, 0.1, 120.0/3.6)
}

func CruiseSpeedICEGear(
	m, g, Crr, rho, Cd, A float64,
	rWheel, gearRatio, finalDrive, eta, theta float64,
	torqueAtRPM func(rpm float64) float64,
) (float64, bool) {
	Preq := func(v float64) float64 {
		return PowerRequired(v, m, g, Crr, rho, Cd, A, theta)
	}
	Pwheel := func(v float64) float64 {
		return WheelPowerICE(v, gearRatio, finalDrive, rWheel, eta, torqueAtRPM)
	}
	return SolveCruise(Pwheel, Preq, 0.1, 120.0/3.6)
}

func BestCruiseICE(
	m, g, Crr, rho, Cd, A float64,
	rWheel, finalDrive, eta, theta float64,
	gears []float64,
	torqueAtRPM func(rpm float64) float64,
) (vBest, gearBest float64, ok bool) {
	best := math.MaxFloat64
	for _, gr := range gears {
		if v, ok := CruiseSpeedICEGear(m, g, Crr, rho, Cd, A, rWheel, gr, finalDrive, eta, theta, torqueAtRPM); ok {
			P := PowerRequired(v, m, g, Crr, rho, Cd, A, theta)
			if P < best {
				best, vBest, gearBest, ok = P, v, gr, true
			}
		}
	}
	return
}

func DistanceAtCruise(E, v, m, g, Crr, rho, Cd, A, theta float64) float64 {
	F := Crr*m*g + 0.5*rho*Cd*A*v*v + m*g*math.Sin(theta)
	if F <= 0 {
		return 0
	}
	return E / F
}
