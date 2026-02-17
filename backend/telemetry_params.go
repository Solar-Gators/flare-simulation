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

    // map cruise target (m/s)
    baseTarget float64,
) []telemetryPoint {
    const (
        stepM  = 0.25
        gmax   = 0.8
        muTire = 0.9
        vMin   = 0.5

        // jerk limit (m/s^3); applied via dt ~= ds/max(v, vMin)
        jerkMax = 1.5
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

    // reasonable guards
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

                lookV := baseTarget
                vLim, dTo, ok := getNext(distInLap)
                if ok && vLim < lookV {
                    lookV = vLim
                }

                var aCmd float64

                if v > lookV && ok && dTo > 0 {
                    aCoast := coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta) // negative
                    aReq := (lookV*lookV - v*v) / (2 * dTo)                     // negative

                    aCmd = math.Max(aReq, aCoast)   // gentlest decel that still meets constraint
                    aCmd = math.Max(aCmd, -aLongMax) // tire limit
                } else if v < baseTarget {
                    aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
                    aCmd = math.Min(aPower, aLongMax)
                } else {
                    aCmd = coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                    aCmd = math.Max(aCmd, -aLongMax)
                }

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

            vCap := math.Sqrt(gmax * g * seg.Radius)
            targetSpeed := math.Min(vCap, baseTarget)

            arcLength := seg.Radius * math.Abs(seg.Angle) * math.Pi / 180.0
            remaining := arcLength
            isRight := seg.Angle < 0

            for remaining > 0 {
                ds := math.Min(stepM, remaining)

                // geometry step along arc
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

                distInLap := distInLapFn(totalDistance)

                lookV := targetSpeed
                vLim, dTo, ok := getNext(distInLap)
                if ok && vLim < lookV {
                    lookV = vLim
                }

                var aCmd float64

                if v > lookV && ok && dTo > 0 {
                    aCoast := coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta) // negative
                    aReq := (lookV*lookV - v*v) / (2 * dTo)                     // negative

                    aCmd = math.Max(aReq, aCoast)
                    aCmd = math.Max(aCmd, -aLongMax)
                } else if v < targetSpeed {
                    aPower := accelAtSpeed(v, vMin, rWheel, Tmax, Pmax, etaDrive, m, g, Crr, rho, Cd, A, theta)
                    aCmd = math.Min(aPower, aLongMax)
                } else {
                    aCmd = coastDecel(v, vMin, m, g, Crr, rho, Cd, A, theta)
                    aCmd = math.Max(aCmd, -aLongMax)
                }

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