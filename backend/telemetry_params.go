package main

import "math"

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

    // requested cruise target (m/s) for straights
    baseTarget float64,
) []telemetryPoint {
    const (
        stepM  = 0.25
        gmax   = 0.8
        muTire = 0.9
        vMin   = 0.5

        jerkMax = 1.5

        // Small safety factor so we never exceed the flip/lat cap due to discretization.
        // 0.98–0.995 is typical.
        entrySafety = 0.99
    )

    points := make([]telemetryPoint, 0, 256)
    x, y, heading := 0.0, 0.0, 0.0

    v := 0.5
    if startFromZero {
        v = 0.0
    }
    totalDistance := 0.0
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

    for _, seg := range segments {
        switch seg.Type {
        case "straight":
            remaining := seg.Length
            for remaining > 0 {
                ds := math.Min(stepM, remaining)

                // straight: no lateral load
                aTotalMax := muTire * g
                aLongMax := aTotalMax

                distInLap := distInLapFn(totalDistance)

                // upcoming curve cap (next constraint) if any
                vLim, dTo, ok := getNext(distInLap)
                entryV := vLim * entrySafety // safe entry speed

                var aCmd float64

                // ---------- COAST-TRIGGERED APPROACH ----------
                // If a constraint is ahead and we're above its entry speed:
                // - do NOT brake yet
                // - compute how far we need to coast (drag+rolling) to reach entryV
                // - if we still have more distance than needed, keep driving/holding cruise
                // - once dTo <= dNeedCoast, lift (coastDecel)
                // - if we're too close for coasting to work, brake only as much as needed
                if ok && dTo > 0 && v > entryV && entryV > 0 {
                    dNeedCoast := coastDistanceToSpeed(v, entryV, Cd, Crr, g, m, A, rho)

                    if dTo > dNeedCoast {
                        // Too early to lift: keep cruising toward baseTarget
                        if v < baseTarget {
                            aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
                            aCmd = math.Min(aPower, aLongMax)
                        } else if v > baseTarget {
                            // If we're above cruise target, do not brake—just coast back down
                            aCmd = coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                            aCmd = math.Max(aCmd, -aLongMax)
                        } else {
                            aCmd = 0
                        }
                    } else {
                        // Time to lift: coast (no brakes)
                        aCmd = coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                        aCmd = math.Max(aCmd, -aLongMax)

                        // If we are already too close (coasting won't hit entryV), brake minimally.
                        // We detect that by checking if the needed-coast distance is larger than remaining distance.
                        // This is the "absolutely required" braking condition.
                        if dNeedCoast > dTo {
                            aReq := (entryV*entryV - v*v) / (2 * dTo) // negative
                            aCmd = math.Max(aReq, aCmd)               // minimal additional braking beyond coasting
                            aCmd = math.Max(aCmd, -aLongMax)
                        }
                    }
                } else {
                    // No upcoming constraint that forces slowing: behave normally around baseTarget
                    if v < baseTarget {
                        aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
                        aCmd = math.Min(aPower, aLongMax)
                    } else if v > baseTarget {
                        // don't brake to hit cruise; coast down
                        aCmd = coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                        aCmd = math.Max(aCmd, -aLongMax)
                    } else {
                        aCmd = 0
                    }
                }

                // JERK LIMIT
                a := applyJerkLimit(aCmd, prevA, v, ds)

                vNext := updateSpeed(v, a, ds)
                if vNext > baseTarget {
                    vNext = baseTarget
                }

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

            // curve’s own cap
            vCap := math.Sqrt(gmax * g * seg.Radius)
            targetSpeed := math.Min(vCap*entrySafety, baseTarget)

            arcLength := seg.Radius * math.Abs(seg.Angle) * math.Pi / 180.0
            remaining := arcLength
            isRight := seg.Angle < 0

            for remaining > 0 {
                ds := math.Min(stepM, remaining)

                // geometry step
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

                // friction circle
                aLat := (v * v) / seg.Radius
                aTotalMax := muTire * g
                aLongMax := math.Sqrt(math.Max(0, aTotalMax*aTotalMax-aLat*aLat))

                var aCmd float64

                // Inside curve: prefer coasting if above target (no braking unless necessary).
                if v > targetSpeed {
                    aCmd = coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                    aCmd = math.Max(aCmd, -aLongMax)

                    // If coasting is not enough to get to targetSpeed by end of remaining arc, brake minimally.
                    aReq := (targetSpeed*targetSpeed - v*v) / (2 * remaining)
                    aCmd = math.Max(aReq, aCmd)
                    aCmd = math.Max(aCmd, -aLongMax)
                } else if v < targetSpeed {
                    aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
                    aCmd = math.Min(aPower, aLongMax)
                } else {
                    aCmd = 0
                }

                // JERK LIMIT
                a := applyJerkLimit(aCmd, prevA, v, ds)

                vNext := updateSpeed(v, a, ds)
                if vNext > targetSpeed {
                    vNext = targetSpeed
                }

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