package main

import "math"

type lapMetrics struct {
	lapTimeSec  float64
	lapEnergyWh float64
	ok          bool
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func applyJerkLimit(aCmd, aPrev, vNow, ds, vMin, jerkMax float64) float64 {
	vEff := math.Max(vNow, vMin)
	dt := ds / vEff
	maxDeltaA := jerkMax * dt
	if aCmd > aPrev+maxDeltaA {
		return aPrev + maxDeltaA
	}
	if aCmd < aPrev-maxDeltaA {
		return aPrev - maxDeltaA
	}
	return aCmd
}

func simulateLapMetrics(segments []trackSegment, req distanceRequest, baseTarget float64) lapMetrics {
	const (
		stepM  = 0.25
		muTire = 0.9
		vMin   = 0.5

		jerkMax = 1.5

		entrySafety = 0.99
		leadInM     = 30.0
	)

	m := req.M
	g := req.G
	Crr := req.Crr
	rho := req.Rho
	Cd := req.Cd
	A := req.A
	theta := req.Theta

	rWheel := req.RWheel
	Tmax := req.Tmax
	Pmax := req.Pmax
	eta := req.EtaDrive

	if baseTarget <= 0 || m <= 0 || g <= 0 || rWheel <= 0 || Tmax <= 0 || Pmax <= 0 || eta <= 0 {
		return lapMetrics{ok: false}
	}

	apexTargets := buildApexTargets(segments, req.Gmax, g)
	lapLength := totalLapLengthM(segments)

	distInLapFn := func(totalDist float64) float64 {
		if req.Wraparound && lapLength > 0 {
			d := math.Mod(totalDist, lapLength)
			if d < 0 {
				d += lapLength
			}
			return d
		}
		return totalDist
	}

	// pack power used only when a>0 (driving)
	packPowerForAccel := func(vNow, aNow float64) float64 {
		if vNow <= 0 || aNow <= 0 {
			return 0
		}

		Presist := PowerRequired(vNow, m, g, Crr, rho, Cd, A, theta)
		Pinert := m * aNow * vNow
		Pwheel := Presist + Pinert
		if Pwheel < 0 {
			Pwheel = 0
		}

		PwheelAvail := WheelPowerEV(vNow, Tmax, Pmax, rWheel, eta)
		if Pwheel > PwheelAvail {
			Pwheel = PwheelAvail
		}

		Ppack := Pwheel / eta
		if Ppack > Pmax {
			Ppack = Pmax
		}
		if Ppack < 0 {
			Ppack = 0
		}
		return Ppack
	}

	cruiseAccelCmd := func(vNow, aLongMax float64) float64 {
		if vNow < baseTarget {
			aPower := accelAtSpeed(vNow, vMin, rWheel, Tmax, Pmax, eta, m, g, Crr, rho, Cd, A, theta)
			return math.Min(aPower, aLongMax)
		}
		if vNow > baseTarget {
			a := coastDecel(vNow, vMin, m, g, Crr, rho, Cd, A, theta)
			return math.Max(a, -aLongMax)
		}
		return 0
	}

	v := vMin
	totalDist := 0.0
	timeSec := 0.0
	energyWh := 0.0
	prevA := 0.0

	for _, seg := range segments {
		switch seg.Type {
		case "straight":
			remaining := seg.Length
			for remaining > 0 {
				ds := math.Min(stepM, remaining)

				aTotalMax := muTire * g
				aLongMax := aTotalMax

				distInLap := distInLapFn(totalDist)
				horizon := apexLookaheadHorizon(v, vMin, aLongMax, leadInM)
				vLim, dTo, ok := sharpestApexWithinHorizon(apexTargets, distInLap, lapLength, horizon, req.Wraparound)

				var aCmd float64

				if ok && dTo > 0 && vLim > 0 {
					entryV := vLim * entrySafety
					if v > entryV && entryV > 0 {
						dNeedCoast := coastDistanceToSpeed(v, entryV, Cd, Crr, g, m, A, rho)

						aCruise := cruiseAccelCmd(v, aLongMax)

						aCoast := coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
						aCoast = math.Max(aCoast, -aLongMax)
						aNeed := aCoast
						if dTo > 0 {
							aNeed = enforceBrakeLookahead(aCoast, v, entryV, dTo, aLongMax)
						}
						jerkBuffer := jerkBrakeBufferDistance(prevA, aNeed, v, vMin, jerkMax)
						blendStart := dNeedCoast + leadInM + jerkBuffer
						blendEnd := dNeedCoast + jerkBuffer

						if dTo > blendStart {
							aCmd = aCruise
						} else if dTo > blendEnd {
							alpha := clamp01((blendStart - dTo) / math.Max(leadInM, 1e-9))
							aCmd = (1-alpha)*aCruise + alpha*aNeed
						} else {
							aCmd = aNeed
						}
					} else {
						aCmd = cruiseAccelCmd(v, aLongMax)
					}
				} else {
					aCmd = cruiseAccelCmd(v, aLongMax)
				}

				a := applyJerkLimit(aCmd, prevA, v, ds, vMin, jerkMax)

				vEff := math.Max(v, vMin)
				dt := ds / vEff
				energyWh += (packPowerForAccel(v, a) * dt) / 3600.0

				vNext := updateSpeed(v, a, ds)
				if vNext > baseTarget {
					vNext = baseTarget
				}

				timeSec += dt
				totalDist += ds
				v = vNext
				prevA = a
				remaining -= ds
			}

		case "curve":
			if seg.Angle == 0 || seg.Radius <= 0 {
				continue
			}

			vCap := math.Sqrt(req.Gmax * g * seg.Radius)
			curveTarget := math.Min(vCap*entrySafety, baseTarget)

			arcLen := seg.Radius * math.Abs(seg.Angle) * math.Pi / 180.0
			remaining := arcLen

			for remaining > 0 {
				ds := math.Min(stepM, remaining)

				aLat := (v * v) / seg.Radius
				aTotalMax := muTire * g
				aLongMax := math.Sqrt(math.Max(0, aTotalMax*aTotalMax-aLat*aLat))

				distInLap := distInLapFn(totalDist)
				horizon := apexLookaheadHorizon(v, vMin, aLongMax, leadInM)
				nextVLim, dToEntry, ok := sharpestApexWithinHorizon(apexTargets, distInLap, lapLength, horizon, req.Wraparound)

				targetSpeed := curveTarget
				if ok && nextVLim > 0 {
					targetSpeed = math.Min(targetSpeed, nextVLim*entrySafety)
				}
				dMust := remaining
				if ok && dToEntry > 0 {
					dMust = dToEntry
				}

				var aCmd float64

				if v > targetSpeed {
					aHold := 0.0
					aCoast := coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
					aCoast = math.Max(aCoast, -aLongMax)

					dNeedCoast := coastDistanceToSpeed(v, targetSpeed, Cd, Crr, g, m, A, rho)
					aNeed := aCoast
					if dMust > 0 {
						aNeed = enforceBrakeLookahead(aCoast, v, targetSpeed, dMust, aLongMax)
					}
					jerkBuffer := jerkBrakeBufferDistance(prevA, aNeed, v, vMin, jerkMax)
					blendStart := dNeedCoast + leadInM + jerkBuffer
					blendEnd := dNeedCoast + jerkBuffer

					if dNeedCoast <= 0 {
						aCmd = aNeed
					} else if dMust > blendStart {
						aCmd = aHold
					} else if dMust > blendEnd {
						alpha := clamp01((blendStart - dMust) / math.Max(leadInM, 1e-9))
						aCmd = (1-alpha)*aHold + alpha*aNeed
					} else {
						aCmd = aNeed
					}
				} else if v < targetSpeed {
					aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, eta, m, g, Crr, rho, Cd, A, theta)
					aCmd = math.Min(aPower, aLongMax)
				} else {
					aCmd = 0
				}

				a := applyJerkLimit(aCmd, prevA, v, ds, vMin, jerkMax)

				vEff := math.Max(v, vMin)
				dt := ds / vEff
				energyWh += (packPowerForAccel(v, a) * dt) / 3600.0

				vNext := updateSpeed(v, a, ds)

				timeSec += dt
				totalDist += ds
				v = vNext
				prevA = a
				remaining -= ds
			}
		}
	}

	if timeSec <= 0 || math.IsNaN(timeSec) || math.IsInf(timeSec, 0) {
		return lapMetrics{ok: false}
	}
	if energyWh < 0 || math.IsNaN(energyWh) || math.IsInf(energyWh, 0) {
		return lapMetrics{ok: false}
	}

	return lapMetrics{lapTimeSec: timeSec, lapEnergyWh: energyWh, ok: true}
}
