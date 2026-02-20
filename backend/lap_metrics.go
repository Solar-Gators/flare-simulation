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
        gmax   = 0.8
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

    constraints := buildConstraints(segments, gmax, g)
    lapLength := totalLapLengthM(segments)

    getNext := func(distInLap float64) (float64, float64, bool) {
        if req.Wraparound {
            return nextConstraintWrap(constraints, distInLap, lapLength)
        }
        return nextConstraint(constraints, distInLap)
    }

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
                vLim, dTo, ok := getNext(distInLap)

                var aCmd float64

                if ok && dTo > 0 && vLim > 0 {
                    entryV := vLim * entrySafety
                    if v > entryV && entryV > 0 {
                        dNeedCoast := coastDistanceToSpeed(v, entryV, Cd, Crr, g, m, A, rho)

                        aCruise := cruiseAccelCmd(v, aLongMax)

                        aCoast := coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                        aCoast = math.Max(aCoast, -aLongMax)

                        if dTo > dNeedCoast+leadInM {
                            aCmd = aCruise
                        } else if dTo > dNeedCoast {
                            alpha := clamp01((dNeedCoast + leadInM - dTo) / leadInM)
                            aCmd = (1-alpha)*aCruise + alpha*aCoast
                        } else {
                            aCmd = aCoast
                            if dNeedCoast > dTo {
                                aReq := (entryV*entryV - v*v) / (2 * dTo)
                                aCmd = math.Max(aReq, aCmd)
                                aCmd = math.Max(aCmd, -aLongMax)
                            }
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

            vCap := math.Sqrt(gmax * g * seg.Radius)
            curveTarget := math.Min(vCap*entrySafety, baseTarget)

            arcLen := seg.Radius * math.Abs(seg.Angle) * math.Pi / 180.0
            remaining := arcLen

            for remaining > 0 {
                ds := math.Min(stepM, remaining)

                aLat := (v * v) / seg.Radius
                aTotalMax := muTire * g
                aLongMax := math.Sqrt(math.Max(0, aTotalMax*aTotalMax-aLat*aLat))

                distInLap := distInLapFn(totalDist)
                nextVLim, _, ok := getNext(distInLap)

                targetSpeed := curveTarget
                if ok && nextVLim > 0 {
                    targetSpeed = math.Min(targetSpeed, nextVLim*entrySafety)
                }

                var aCmd float64

                if v > targetSpeed {
                    aHold := 0.0
                    aCoast := coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                    aCoast = math.Max(aCoast, -aLongMax)

                    dNeedCoast := coastDistanceToSpeed(v, targetSpeed, Cd, Crr, g, m, A, rho)

                    if dNeedCoast <= 0 {
                        aCmd = aCoast
                    } else if remaining > dNeedCoast+leadInM {
                        aCmd = aHold
                    } else if remaining > dNeedCoast {
                        alpha := clamp01((dNeedCoast + leadInM - remaining) / leadInM)
                        aCmd = (1-alpha)*aHold + alpha*aCoast
                    } else {
                        aCmd = aCoast
                    }

                    aReq := (targetSpeed*targetSpeed - v*v) / (2 * remaining)
                    aCmd = math.Max(aReq, aCmd)
                    aCmd = math.Max(aCmd, -aLongMax)
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
                if vNext > targetSpeed {
                    vNext = targetSpeed
                }

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