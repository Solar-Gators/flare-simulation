package main

// physics.go
// NOTE: This file assumes you already have `package main` (or your existing package) elsewhere,
// and that your Segment/Track types are already defined exactly once in your codebase.
// Do NOT add another package declaration here.

import (
	"fmt"
	"math"
)

// calcCurveSpeed returns the max steady-state curve speed given a lateral accel cap.
// v_max = sqrt((gmax*g) * R)
func calcCurveSpeed(seg Segment, gravity, gmax float64) float64 {
    if seg.Radius <= 0 || gravity <= 0 || gmax <= 0 {
        return 0
    }
    return math.Sqrt(gmax * gravity * seg.Radius)
}

// PowerRequired returns wheel mechanical power required to maintain speed v (W).
func PowerRequired(v, m, g, crr, rho, cd, area, theta float64) float64 {
    return (crr*m*g+m*g*math.Sin(theta))*v + 0.5*rho*cd*area*v*v*v
}

// WheelPowerEV returns available wheel mechanical power (W) given torque + power caps.
func WheelPowerEV(v, tMax, pMax, rWheel, eta float64) float64 {
    if v <= 0 || rWheel <= 0 || eta <= 0 || tMax <= 0 || pMax <= 0 {
        return 0
    }
    omegaWheel := v / rWheel
    pTorque := tMax * omegaWheel
    pCap := eta * pMax
    if pTorque < pCap {
        return pTorque
    }
    return pCap
}

// DistanceForSpeedEV computes the maximum distance (m) achievable at constant speed v,
// given battery + solar, and checks feasibility vs wheel power available.
//
// Units:
// batteryWh [Wh], solarWhPerMin [Wh/min], raceDayMin [min], Pmax [W], etc.
func DistanceForSpeedEV(
    v float64,
    batteryWh, solarWhPerMin, etaDrive, raceDayMin float64,
    rWheel, tMax, pMax float64,
    m, g, crr, rho, cd, area, theta float64,
) (float64, bool) {
    if v <= 0 || raceDayMin <= 0 || etaDrive <= 0 {
        return 0, false
    }

    preq := PowerRequired(v, m, g, crr, rho, cd, area, theta) // wheel W
    if preq <= 0 || math.IsNaN(preq) || math.IsInf(preq, 0) {
        return 0, false
    }

    // Feasibility check: can the EV supply the required wheel power at this speed?
    if WheelPowerEV(v, tMax, pMax, rWheel, etaDrive)+1e-9 < preq {
        return 0, false
    }

    tSec := raceDayMin * 60.0

    // Battery energy at wheel in Joules:
    // Wh * 3600 = J, then multiply by etaDrive to get wheel-available energy.
    eBattWheelJ := batteryWh * 3600.0 * etaDrive

    // Solar power at wheel in Watts:
    // (Wh/min) * (3600 J/Wh) / (60 s/min) = (Wh/min) * 60 = W, then * etaDrive.
    pSolarWheel := solarWhPerMin * 60.0 * etaDrive

    // If solar alone covers demand, can run full race duration.
    if preq <= pSolarWheel {
        return v * tSec, true
    }

    drain := preq - pSolarWheel // W = J/s
    tEnd := eBattWheelJ / drain
    if tEnd > tSec {
        tEnd = tSec
    }
    if tEnd < 0 {
        tEnd = 0
    }

    return v * tEnd, true
}

// calcCoastDistance estimates distance (m) needed to coast from currentSpeed down to the
// curve target speed using a simple drag+rolling model.
func calcCoastDistance(
    currentSpeed float64,
    segment Segment,
    cd float64,
    crr float64,
    gravity float64,
    gmax float64,
    mass float64,
    area float64,
    rho float64,
) float64 {
    targetSpeed := calcCurveSpeed(segment, gravity, gmax)

    if targetSpeed <= 0 || currentSpeed <= 0 || currentSpeed <= targetSpeed {
        return 0
    }
    if cd <= 0 || area <= 0 || rho <= 0 || mass <= 0 || gravity <= 0 {
        return 0
    }

    vi2 := currentSpeed * currentSpeed
    vf2 := targetSpeed * targetSpeed

    startTerm := (rho*cd*area)/(2*mass)*vi2 + (crr * gravity)
    endTerm := (rho*cd*area)/(2*mass)*vf2 + (crr * gravity)
    if startTerm <= 0 || endTerm <= 0 {
        return 0
    }

    lnTerm := math.Log(startTerm / endTerm)
    return (mass / (rho * cd * area)) * lnTerm
}

// coastDistanceToSpeed returns distance (m) needed to coast from v0 down to v1 (v1 < v0)
// using the same drag+rolling closed-form used in calcCoastDistance.
func coastDistanceToSpeed(v0, v1, cd, crr, gravity, mass, area, rho float64) float64 {
    if v1 <= 0 || v0 <= 0 || v1 >= v0 {
        return 0
    }
    if cd <= 0 || area <= 0 || rho <= 0 || mass <= 0 || gravity <= 0 {
        return 0
    }

    vi2 := v0 * v0
    vf2 := v1 * v1

    startTerm := (rho*cd*area)/(2*mass)*vi2 + (crr * gravity)
    endTerm := (rho*cd*area)/(2*mass)*vf2 + (crr * gravity)
    if startTerm <= 0 || endTerm <= 0 {
        return 0
    }

    return (mass / (rho * cd * area)) * math.Log(startTerm/endTerm)
}

// coastConservation computes energy saved (Wh) by coasting for distanceM,
// assuming you otherwise would have spent cruiseWhPerM, but instead spend bottomWhPerM.
// Since these are Wh/m, savedWh = (Wh/m)*m.
func coastConservation(cruiseWhPerM, bottomWhPerM, distanceM float64) float64 {
    if distanceM <= 0 {
        return 0
    }
    return (cruiseWhPerM - bottomWhPerM) * distanceM
}

// curveAccelEnergy computes total energy (Wh) to accelerate from initSpeed to cruiseSpeed
// under constant accel, including aero drag and rolling resistance.
func curveAccelEnergy(
    mass float64,
    area float64,
    cd float64,
    crr float64,
    initSpeed float64,
    cruiseSpeed float64,
    accel float64,
    rho float64,
    gravity float64,
) float64 {
    if cruiseSpeed <= initSpeed || accel <= 0 || mass <= 0 || gravity <= 0 {
        return 0
    }
    if cd < 0 || crr < 0 || area <= 0 || rho <= 0 {
        return 0
    }

    vi := initSpeed
    vf := cruiseSpeed

    deltaKE := 0.5 * mass * (vf*vf - vi*vi)          // J
    distance := (vf*vf - vi*vi) / (2 * accel)        // m
    eDrag := (0.5 * rho * cd * area / accel) * (math.Pow(vf, 4)-math.Pow(vi, 4)) / 4.0 // J
    eRoll := crr * mass * gravity * distance          // J

    return (deltaKE + eDrag + eRoll) / 3600.0 // Wh
}

// netCurveLosses returns (energyUsed - energySaved) in Wh for a “coast into curve then accel out”.
func netCurveLosses(
    mass float64,
    area float64,
    cd float64,
    crr float64,
    upcomingCurve Segment,
    cruiseSpeed float64,
    accel float64,
    rho float64,
    gravity float64,
    cruiseWhPerM float64,
    bottomWhPerM float64,
    distanceToCoast float64,
    gmax float64,
) float64 {
    curveSpeed := calcCurveSpeed(upcomingCurve, gravity, gmax)
    energyUsedWh := curveAccelEnergy(mass, area, cd, crr, curveSpeed, cruiseSpeed, accel, rho, gravity)
    energySavedWh := coastConservation(cruiseWhPerM, bottomWhPerM, distanceToCoast)
    return energyUsedWh - energySavedWh
}

// simulateCoast is a simple console test of the coast + curve-loss model.
// It uses your existing Track/Segment definitions.
func simulateCoast(raceTrack Track) {
    for i := 0; i < len(raceTrack.Segments); i++ {
        if i+1 == len(raceTrack.Segments) {
            break
        }

        cruiseSpeed := 27.0 // m/s
        upcoming := raceTrack.Segments[i+1]
        if upcoming.Radius <= 0 {
            continue
        }

        cd := 0.21
        crr := 0.0015
        gravity := 9.81
        gmax := 0.8
        mass := 285.0
        area := 0.456
        rho := 1.225
        bottomWhPerM := 0.006
        accel := 0.5

        dToCoast := calcCoastDistance(cruiseSpeed, upcoming, cd, crr, gravity, gmax, mass, area, rho)

        if dToCoast <= 0 {
            fmt.Println("Already below target curve speed (or invalid coast model).")
            continue
        }

        // If current segment is a straight with known length, sanity-check available distance.
        if raceTrack.Segments[i].Angle == 0 && raceTrack.Segments[i].Length > 0 && dToCoast > raceTrack.Segments[i].Length {
            fmt.Println("Not enough track to slow down:", dToCoast, "m needed")
            continue
        }

        fmt.Println("Coast to target with:", dToCoast, "m")

        // Cruise energy at constant speed (Wh/m). theta assumed 0 for now.
        cruiseWhPerM := PowerRequired(cruiseSpeed, mass, gravity, crr, rho, cd, area, 0) / cruiseSpeed / 3600.0
        fmt.Println("Cruise Energy in Wh/m:", cruiseWhPerM)

        netE := netCurveLosses(mass, area, cd, crr, upcoming, cruiseSpeed, accel, rho, gravity, cruiseWhPerM, bottomWhPerM, dToCoast, gmax)
        fmt.Println("Net loss of energy is:", netE, "Wh")
        fmt.Println("--------------------------------------------------")
    }
}