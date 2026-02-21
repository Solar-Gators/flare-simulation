package main

import "math"

// Wrapper: if wraparound=true, do a warm-up lap and return the second lap starting
// at the previous lap’s end speed (so the preview starts “already in motion”).
func buildTelemetryWithParams(
    segments []trackSegment,
    wraparound bool,
    startFromZero bool,

    // physics
    m float64,
    g float64,
    Crr float64,
    rho float64,
    Cd float64,
    A float64,
    theta float64,

    // EV
    rWheel float64,
    Tmax float64,
    Pmax float64,
    etaDrive float64,

    // cruise target on straights (m/s)
    baseTarget float64,
) []telemetryPoint {

    // Non-wraparound: behave as before.
    if !wraparound {
        return buildTelemetryOneLapWithParams(
            segments,
            wraparound,
            startFromZero,
            m, g, Crr, rho, Cd, A, theta,
            rWheel, Tmax, Pmax, etaDrive,
            baseTarget,
            0, 0, 0,
            0,
            0, // startSpeed (0 => use startFromZero logic)
        )
    }

    // Wraparound: warm-up one lap, then generate a second lap starting from the warm-up end speed.
    warm := buildTelemetryOneLapWithParams(
        segments,
        wraparound,
        false, // don't force startFromZero for warmup
        m, g, Crr, rho, Cd, A, theta,
        rWheel, Tmax, Pmax, etaDrive,
        baseTarget,
        0, 0, 0,
        0,
        0,
    )

    if len(warm) < 2 {
        return warm
    }

    last := warm[len(warm)-1]

    // Start next lap at the previous lap’s end speed. Reset distance to 0 for clean coloring/viewBox.
    return buildTelemetryOneLapWithParams(
        segments,
        wraparound,
        false,
        m, g, Crr, rho, Cd, A, theta,
        rWheel, Tmax, Pmax, etaDrive,
        baseTarget,
        0, 0, 0, // restart position/orientation at start/finish for display
        0,
        last.Speed, // this is the key: start at previous lap end speed
    )
}

// The original implementation, parameterized with initial conditions.
// startSpeed <= 0 means “use startFromZero logic” (0 speed if startFromZero else 0.5).
func buildTelemetryOneLapWithParams(
    segments []trackSegment,
    wraparound bool,
    startFromZero bool,

    // physics
    m float64,
    g float64,
    Crr float64,
    rho float64,
    Cd float64,
    A float64,
    theta float64,

    // EV
    rWheel float64,
    Tmax float64,
    Pmax float64,
    etaDrive float64,

    // cruise target on straights (m/s)
    baseTarget float64,

    // initial conditions
    startX float64,
    startY float64,
    startHeading float64,
    startDistance float64,
    startSpeed float64,
) []telemetryPoint {
    const (
        stepM  = 0.25
        gmax   = 0.8
        muTire = 0.9
        vMin   = 0.5

        jerkMax = 1.5

        entrySafety = 0.99
        leadInM     = 30.0
    )

    points := make([]telemetryPoint, 0, 256)

    x, y, heading := startX, startY, startHeading

    v := startSpeed
    if v <= 0 {
        v = 0.5
        if startFromZero {
            v = 0.0
        }
    }

    totalDistance := startDistance
    prevA := 0.0
    points = append(points, telemetryPoint{X: x, Y: y, Speed: v, Accel: 0, Distance: totalDistance})

    if baseTarget <= 0 {
        baseTarget = 1.0
    }

    constraints := buildConstraints(segments, gmax, g)
    lapLength := totalLapLengthM(segments)

    getNext := func(distInLap float64) (float64, float64, bool) {
        if wraparound {
            return nextConstraintWrap(constraints, distInLap, lapLength)
        }
        return nextConstraint(constraints, distInLap)
    }

    distInLapFn := func(totalDist float64) float64 {
        if wraparound && lapLength > 0 {
            d := math.Mod(totalDist, lapLength)
            if d < 0 {
                d += lapLength
            }
            return d
        }
        return totalDist
    }

    clamp01 := func(x float64) float64 {
        if x < 0 {
            return 0
        }
        if x > 1 {
            return 1
        }
        return x
    }

    applyJerkLimit := func(aCmd, aPrev, vNow, ds float64) float64 {
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

    cruiseAccelCmd := func(vNow, aLongMax float64) float64 {
        if vNow < baseTarget {
            aPower := accelAtSpeed(vNow, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
            return math.Min(aPower, aLongMax)
        }
        if vNow > baseTarget {
            a := coastDecel(vNow, vMin, m, g, Crr, rho, Cd, A, theta)
            return math.Max(a, -aLongMax)
        }
        return 0
    }

    for _, seg := range segments {
        switch seg.Type {
        case "straight":
            remaining := seg.Length
            for remaining > 0 {
                ds := math.Min(stepM, remaining)

                aTotalMax := muTire * g
                aLongMax := aTotalMax

                distInLap := distInLapFn(totalDistance)
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
                        }

                        // --- NEW: enforce "must meet entryV by the entry point" ---
                        // This prevents entering the curve above cap and then "teleport clamping".
                        if dTo > 0 {
                            aReqEntry := (entryV*entryV - v*v) / (2 * dTo)
                            // aCmd must be <= aReqEntry (more negative) to be feasible.
                            if aCmd > aReqEntry {
                                aCmd = aReqEntry
                            }
                            aCmd = math.Max(aCmd, -aLongMax)
                        }
                    } else {
                        aCmd = cruiseAccelCmd(v, aLongMax)
                    }
                } else {
                    aCmd = cruiseAccelCmd(v, aLongMax)
                }

                a := applyJerkLimit(aCmd, prevA, v, ds)

                vNext := updateSpeed(v, a, ds)
                // --- REMOVED: hard clamp to baseTarget to avoid discontinuities ---

                x += ds * math.Cos(heading)
                y += ds * math.Sin(heading)
                totalDistance += ds
                points = append(points, telemetryPoint{X: x, Y: y, Speed: vNext, Accel: a, Distance: totalDistance})

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

            arcLength := seg.Radius * math.Abs(seg.Angle) * math.Pi / 180.0
            remaining := arcLength
            isRight := seg.Angle < 0

            for remaining > 0 {
                ds := math.Min(stepM, remaining)

                delta := ds / seg.Radius
                if !isRight {
                    delta = -delta
                }

                normalX := -math.Sin(heading)
                normalY := math.Cos(heading)
                if !isRight {
                    normalX = math.Sin(heading)
                    normalY = -math.Cos(heading)
                }

                centerX := x + seg.Radius*normalX
                centerY := y + seg.Radius*normalY
                dx := x - centerX
                dy := y - centerY

                c := math.Cos(delta)
                s := math.Sin(delta)
                x = centerX + dx*c - dy*s
                y = centerY + dx*s + dy*c
                heading += delta

                aLat := (v * v) / seg.Radius
                aTotalMax := muTire * g
                aLongMax := math.Sqrt(math.Max(0, aTotalMax*aTotalMax-aLat*aLat))

                // --- NEW: keep the distance to the next cap-drop entry ---
                distInLap := distInLapFn(totalDistance)
                nextVLim, dToEntry, okEntry := getNext(distInLap)

                targetSpeed := curveTarget
                if okEntry && nextVLim > 0 {
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

                    // --- NEW: enforce meeting targetSpeed BY the entry point distance, not by end of this segment ---
                    dMust := remaining
                    if okEntry && dToEntry > 0 {
                        dMust = dToEntry
                    }
                    if dMust > 0 {
                        aReq := (targetSpeed*targetSpeed - v*v) / (2 * dMust)
                        aCmd = math.Max(aReq, aCmd)
                        aCmd = math.Max(aCmd, -aLongMax)
                    }
                } else if v < targetSpeed {
                    aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
                    aCmd = math.Min(aPower, aLongMax)
                } else {
                    aCmd = 0
                }

                a := applyJerkLimit(aCmd, prevA, v, ds)

                vNext := updateSpeed(v, a, ds)
                // --- REMOVED: hard clamp to targetSpeed to avoid discontinuities ---

                totalDistance += ds
                points = append(points, telemetryPoint{X: x, Y: y, Speed: vNext, Accel: a, Distance: totalDistance})

                v = vNext
                prevA = a
                remaining -= ds
            }
        }
    }

    return points
}