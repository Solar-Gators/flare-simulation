package main

import "math"

type speedConstraint struct {
    dist   float64 // meters from start of lap
    vLimit float64 // m/s
}

func segLengthM(seg trackSegment) float64 {
    switch seg.Type {
    case "straight":
        if seg.Length > 0 {
            return seg.Length
        }
        return 0
    case "curve":
        if seg.Radius > 0 && seg.Angle != 0 {
            return seg.Radius * math.Abs(seg.Angle) * math.Pi / 180.0
        }
        return 0
    default:
        return 0
    }
}

func totalLapLengthM(segments []trackSegment) float64 {
    sum := 0.0
    for _, seg := range segments {
        sum += segLengthM(seg)
    }
    return sum
}

// Adds a constraint at every segment start where local vCap drops.
// straight: vCap = +Inf
// curve:    vCap = sqrt(gmax*g*R)
func buildConstraints(segments []trackSegment, gmax, g float64) []speedConstraint {
    constraints := make([]speedConstraint, 0, len(segments))

    prevCap := math.Inf(1)
    s := 0.0

    for _, seg := range segments {
        localCap := math.Inf(1)
        if seg.Type == "curve" && seg.Radius > 0 {
            localCap = math.Sqrt(gmax * g * seg.Radius)
        }

        if localCap < prevCap-1e-9 {
            constraints = append(constraints, speedConstraint{dist: s, vLimit: localCap})
        }

        prevCap = localCap
        s += segLengthM(seg)
    }

    return constraints
}

// Non-wrap: next constraint strictly ahead.
func nextConstraint(constraints []speedConstraint, curDist float64) (float64, float64, bool) {
    for _, c := range constraints {
        if c.dist > curDist+1e-9 {
            return c.vLimit, c.dist - curDist, true
        }
    }
    return 0, 0, false
}

// Wraparound-aware: treat lap as circular.
func nextConstraintWrap(constraints []speedConstraint, curDist, lapLength float64) (float64, float64, bool) {
    if len(constraints) == 0 || lapLength <= 0 {
        return 0, 0, false
    }

    d := math.Mod(curDist, lapLength)
    if d < 0 {
        d += lapLength
    }

    for _, c := range constraints {
        if c.dist > d+1e-9 {
            return c.vLimit, c.dist - d, true
        }
    }

    c0 := constraints[0]
    return c0.vLimit, (lapLength - d) + c0.dist, true
}