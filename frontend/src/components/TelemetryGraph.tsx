import { useMemo, useState } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Label,
  Tooltip as ReTooltip,
  CartesianGrid,
  ResponsiveContainer,
} from 'recharts'

type TPoint = {
  x: number
  y: number
  speed: number
  accel: number
  distance: number
}

type Props = {
  telemetry: TPoint[]
  additionalEfficiency: number
  imperialUnits: boolean
}

type ChartPoint = {
  distance: number
  speed: number
  accel: number
  energy: number
}

const METERS_PER_MILE = 1609.344
const MPS_TO_MPH = 2.2369362920544
const MPS2_TO_FTPS2 = 3.280839895013123

function powerRequired(v: number, additionalEfficiency: number) {
  const m = 285.0
  const g = 9.81
  const Crr = 0.0015
  const rho = 1.225
  const Cd = 0.21
  const A = 0.456
  const theta = 0.0
  const fRolling = (Crr * m * g + m * g * Math.sin(theta)) * v
  const pAero = 0.5 * rho * Cd * A * v * v * v
  return (fRolling + pAero) * (1 + additionalEfficiency / 100)
}

function energyWhPerMeter(v: number, additionalEfficiency: number) {
  if (v <= 0) return 0
  const P = powerRequired(v, additionalEfficiency)
  return P / (v * 3600)
}

function metersPerSecondToMph(value: number) {
  return value * MPS_TO_MPH
}

function metersPerSecondSquaredToFeetPerSecondSquared(value: number) {
  return value * MPS2_TO_FTPS2
}

function energyWhPerMeterToWhPerMile(value: number) {
  return value * METERS_PER_MILE
}

function binTelemetry(points: TPoint[], additionalEfficiency: number, binSize = 100): ChartPoint[] {
  if (points.length === 0) return []
  const bins = new Map<
    number,
    { count: number; sumDist: number; sumSpeed: number; sumAccel: number; sumEnergy: number }
  >()
  for (const p of points) {
    const binKey = Math.floor(p.distance / binSize)
    const existing = bins.get(binKey)
    const e = energyWhPerMeter(p.speed, additionalEfficiency)
    if (existing) {
      existing.count += 1
      existing.sumDist += p.distance
      existing.sumSpeed += p.speed
      existing.sumAccel += p.accel
      existing.sumEnergy += e
    } else {
      bins.set(binKey, {
        count: 1,
        sumDist: p.distance,
        sumSpeed: p.speed,
        sumAccel: p.accel,
        sumEnergy: e,
      })
    }
  }
  const out: ChartPoint[] = []
  const keys = Array.from(bins.keys()).sort((a, b) => a - b)
  for (const k of keys) {
    const v = bins.get(k)!
    out.push({
      distance: v.sumDist / v.count,
      speed: v.sumSpeed / v.count,
      accel: v.sumAccel / v.count,
      energy: v.sumEnergy / v.count,
    })
  }
  return out
}

export default function TelemetryGraph({ telemetry, additionalEfficiency, imperialUnits }: Props) {
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [showAll, setShowAll] = useState(true)
  const [normalize100, setNormalize100] = useState(false)

  const options = useMemo(
    () =>
      telemetry.map((p, i) => ({
        i,
        label: `${i}: ${imperialUnits ? (p.distance / METERS_PER_MILE).toFixed(2) : p.distance.toFixed(1)} ${imperialUnits ? 'mi' : 'm'}`,
      })),
    [telemetry, imperialUnits],
  )
  const tooltipProps = useMemo(
    () => ({
      shared: true,
      cursor: { stroke: '#999', strokeDasharray: '3 3' },
    }),
    [],
  )

  const windowPoints = useMemo(() => {
    if (telemetry.length === 0) return [] as ChartPoint[]

    let base: TPoint[]
    if (showAll) {
      base = telemetry
    } else {
      const half = 50
      const idx = Math.max(0, Math.min(selectedIndex, telemetry.length - 1))
      const start = Math.max(0, idx - half)
      const end = Math.min(telemetry.length, idx + half)
      base = telemetry.slice(start, end)
    }

    const points = normalize100
      ? binTelemetry(base, additionalEfficiency, 100)
      : base.map((p) => ({
          distance: p.distance,
          speed: p.speed,
          accel: p.accel,
          energy: energyWhPerMeter(p.speed, additionalEfficiency),
        }))

    if (!imperialUnits) {
      return points
    }

    return points.map((point) => ({
      distance: point.distance / METERS_PER_MILE,
      speed: metersPerSecondToMph(point.speed),
      accel: metersPerSecondSquaredToFeetPerSecondSquared(point.accel),
      energy: energyWhPerMeterToWhPerMile(point.energy),
    }))
  }, [telemetry, selectedIndex, showAll, normalize100, additionalEfficiency, imperialUnits])

  const distanceUnitLabel = imperialUnits ? 'mi' : 'm'
  const speedUnitLabel = imperialUnits ? 'mph' : 'm/s'
  const accelUnitLabel = imperialUnits ? 'ft/s²' : 'm/s²'
  const energyUnitLabel = imperialUnits ? 'Wh/mi' : 'Wh/m'
  const binLabel = imperialUnits ? 'Bin by 0.06 mi' : 'Bin by 100 m'

  return (
    <div style={{ fontFamily: 'system-ui, Arial' }}>
      <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginBottom: 8 }}>
        <label style={{ fontSize: 14 }}>Select point</label>
        <select value={selectedIndex} onChange={(e) => setSelectedIndex(Number(e.target.value))}>
          {options.map((o) => (
            <option key={o.i} value={o.i}>
              {o.label}
            </option>
          ))}
        </select>
        <label style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <input type="checkbox" checked={showAll} onChange={(e) => setShowAll(e.target.checked)} />
          Show all points
        </label>
        <label style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <input
            type="checkbox"
            checked={normalize100}
            onChange={(e) => setNormalize100(e.target.checked)}
          />
          {binLabel}
        </label>
        <div style={{ marginLeft: 'auto', fontSize: 13, color: '#666' }}>
          {telemetry.length} points
        </div>
      </div>

      <div style={{ display: 'grid', gap: 12 }}>
        <div style={{ height: 180 }}>
          <ResponsiveContainer>
            <LineChart data={windowPoints}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="distance"
                tickFormatter={(d: number | string) =>
                  imperialUnits
                    ? `${Number(d).toFixed(2)}${distanceUnitLabel}`
                    : `${Math.round(Number(d))}${distanceUnitLabel}`
                }
              >
                <Label
                  value={`Distance (${distanceUnitLabel})`}
                  position="insideBottom"
                  offset={-2}
                />
              </XAxis>
              <YAxis>
                <Label value={`Speed (${speedUnitLabel})`} angle={-90} position="insideLeft" />
              </YAxis>
              <ReTooltip {...tooltipProps} />
              <Line
                type="monotone"
                dataKey="speed"
                stroke="#007acc"
                dot={false}
                strokeWidth={2}
                activeDot={{ r: 5 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div style={{ height: 180 }}>
          <ResponsiveContainer>
            <LineChart data={windowPoints}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="distance"
                tickFormatter={(d: number | string) =>
                  imperialUnits
                    ? `${Number(d).toFixed(2)}${distanceUnitLabel}`
                    : `${Math.round(Number(d))}${distanceUnitLabel}`
                }
              >
                <Label
                  value={`Distance (${distanceUnitLabel})`}
                  position="insideBottom"
                  offset={-2}
                />
              </XAxis>
              <YAxis>
                <Label
                  value={`Acceleration (${accelUnitLabel})`}
                  angle={-90}
                  position="insideLeft"
                />
              </YAxis>
              <ReTooltip {...tooltipProps} />
              <Line
                type="monotone"
                dataKey="accel"
                stroke="#e55353"
                dot={false}
                strokeWidth={2}
                activeDot={{ r: 5 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div style={{ height: 180 }}>
          <ResponsiveContainer>
            <LineChart data={windowPoints}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="distance"
                tickFormatter={(d: number | string) =>
                  imperialUnits
                    ? `${Number(d).toFixed(2)}${distanceUnitLabel}`
                    : `${Math.round(Number(d))}${distanceUnitLabel}`
                }
              >
                <Label
                  value={`Distance (${distanceUnitLabel})`}
                  position="insideBottom"
                  offset={-2}
                />
              </XAxis>
              <YAxis>
                <Label value={`Energy (${energyUnitLabel})`} angle={-90} position="insideLeft" />
              </YAxis>
              <ReTooltip {...tooltipProps} />
              <Line
                type="monotone"
                dataKey="energy"
                stroke="#22aa55"
                dot={false}
                strokeWidth={2}
                activeDot={{ r: 5 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
}
