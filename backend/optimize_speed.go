package main

import "math"

func findOptimalVForFullDepletion(
    segments []trackSegment,
    req distanceRequest,
    vMin, vMax float64,
) (bestV float64, remainingWh float64, ok bool) {

    raceSec := req.RaceDayMin * 60.0
    energyAvail := req.BatteryWh + req.SolarWhPerMin*req.RaceDayMin

    f := func(v float64) (float64, lapMetrics) {
        m := simulateLapMetrics(segments, req, v)
        if !m.ok || m.lapTimeSec <= 0 {
            return math.NaN(), m
        }
        laps := raceSec / m.lapTimeSec
        need := laps * m.lapEnergyWh
        return energyAvail - need, m
    }

    lo, hi := vMin, vMax
    fLo, mLo := f(lo)
    fHi, mHi := f(hi)
    if !mLo.ok || !mHi.ok || math.IsNaN(fLo) || math.IsNaN(fHi) {
        return 0, 0, false
    }

    if fLo < 0 {
        return 0, fLo, false
    }
    if fHi > 0 {
        return hi, fHi, true
    }

    bestV = lo
    remainingWh = fLo

    for i := 0; i < 50; i++ {
        mid := 0.5 * (lo + hi)
        fMid, mMid := f(mid)
        if !mMid.ok || math.IsNaN(fMid) {
            hi = mid
            continue
        }

        if math.Abs(fMid) < math.Abs(remainingWh) {
            bestV = mid
            remainingWh = fMid
        }

        if fMid > 0 {
            lo = mid
        } else {
            hi = mid
        }

        if (hi - lo) < 0.001 {
            break
        }
    }

    return bestV, remainingWh, true
}