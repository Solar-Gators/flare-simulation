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

// buildApexTargets places one target at the midpoint of every curve segment.
// For the current segment model, the midpoint acts as the local apex.
func buildApexTargets(segments []trackSegment, gmax, g float64) []speedConstraint {
	targets := make([]speedConstraint, 0, len(segments))
	s := 0.0

	for _, seg := range segments {
		segLen := segLengthM(seg)
		if seg.Type == "curve" && seg.Radius > 0 && segLen > 0 {
			targets = append(targets, speedConstraint{
				dist:   s + 0.5*segLen,
				vLimit: math.Sqrt(gmax * g * seg.Radius),
			})
		}
		s += segLen
	}

	return targets
}

// buildConstraints adds a constraint at every segment start where the local lateral cap drops.
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

// nextConstraint finds the next constraint strictly ahead (no wrap).
func nextConstraint(constraints []speedConstraint, curDist float64) (float64, float64, bool) {
	for _, c := range constraints {
		if c.dist > curDist+1e-9 {
			return c.vLimit, c.dist - curDist, true
		}
	}
	return 0, 0, false
}

// nextConstraintWrap finds the next constraint treating the lap as circular.
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

func forwardDistanceToTarget(curDist, targetDist, lapLength float64, wraparound bool) (float64, bool) {
	if wraparound {
		if lapLength <= 0 {
			return 0, false
		}
		d := math.Mod(curDist, lapLength)
		if d < 0 {
			d += lapLength
		}
		dist := targetDist - d
		if dist <= 1e-9 {
			dist += lapLength
		}
		return dist, true
	}

	if targetDist <= curDist+1e-9 {
		return 0, false
	}
	return targetDist - curDist, true
}

// sharpestApexWithinHorizon scans upcoming curve apices and returns the sharpest
// one within the requested lookahead distance.
func sharpestApexWithinHorizon(
	targets []speedConstraint,
	curDist float64,
	lapLength float64,
	horizon float64,
	wraparound bool,
) (float64, float64, bool) {
	if len(targets) == 0 || horizon <= 0 {
		return 0, 0, false
	}

	bestVLimit := 0.0
	bestDist := 0.0
	found := false

	for _, target := range targets {
		dist, ok := forwardDistanceToTarget(curDist, target.dist, lapLength, wraparound)
		if !ok || dist > horizon {
			continue
		}
		if !found || target.vLimit < bestVLimit-1e-9 || (math.Abs(target.vLimit-bestVLimit) <= 1e-9 && dist < bestDist) {
			bestVLimit = target.vLimit
			bestDist = dist
			found = true
		}
	}

	if !found {
		return 0, 0, false
	}
	return bestVLimit, bestDist, true
}

// mostRestrictiveConstraintAhead returns the future constraint that imposes the
// lowest currently-feasible speed after accounting for available braking.
func mostRestrictiveConstraintAhead(
	constraints []speedConstraint,
	curDist float64,
	lapLength float64,
	aBrake float64,
	wraparound bool,
) (float64, float64, bool) {
	if len(constraints) == 0 || aBrake <= 0 {
		return 0, 0, false
	}

	bestNowCap := math.Inf(1)
	bestVLimit := 0.0
	bestDist := 0.0

	consider := func(vLimit, dist float64) {
		if dist <= 0 || vLimit <= 0 {
			return
		}
		nowCap := math.Sqrt(vLimit*vLimit + 2*aBrake*dist)
		if nowCap < bestNowCap {
			bestNowCap = nowCap
			bestVLimit = vLimit
			bestDist = dist
		}
	}

	if wraparound {
		if lapLength <= 0 {
			return 0, 0, false
		}
		d := math.Mod(curDist, lapLength)
		if d < 0 {
			d += lapLength
		}
		for _, c := range constraints {
			dist := c.dist - d
			if dist <= 1e-9 {
				dist += lapLength
			}
			consider(c.vLimit, dist)
		}
	} else {
		for _, c := range constraints {
			if c.dist <= curDist+1e-9 {
				continue
			}
			consider(c.vLimit, c.dist-curDist)
		}
	}

	if math.IsNaN(bestNowCap) || math.IsInf(bestNowCap, 0) {
		return 0, 0, false
	}
	return bestVLimit, bestDist, true
}
