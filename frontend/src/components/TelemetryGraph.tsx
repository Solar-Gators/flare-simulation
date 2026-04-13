import { useMemo, useState } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
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
}

type ChartPoint = {
  distance: number
  speed: number
  accel: number
  energy: number
}

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

function binTelemetry(
  points: TPoint[],
  additionalEfficiency: number,
  binSize = 100,
): ChartPoint[] {
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

export default function TelemetryGraph({ telemetry, additionalEfficiency }: Props) {
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [showAll, setShowAll] = useState(true)
  const [normalize100, setNormalize100] = useState(false)

  const options = useMemo(() => telemetry.map((p, i) => ({ i, label: `${i}: ${p.distance.toFixed(1)} m` })), [telemetry])
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

    if (normalize100) {
      return binTelemetry(base, additionalEfficiency, 100)
    }

    // map to ChartPoint with per-point energy
    return base.map((p) => ({ distance: p.distance, speed: p.speed, accel: p.accel, energy: energyWhPerMeter(p.speed, additionalEfficiency) }))
  }, [telemetry, selectedIndex, showAll, normalize100, additionalEfficiency])

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
          Bin by 100 m
        </label>
        <div style={{ marginLeft: 'auto', fontSize: 13, color: '#666' }}>{telemetry.length} points</div>
      </div>

      <div style={{ display: 'grid', gap: 12 }}>
        <div style={{ height: 180 }}>
          <ResponsiveContainer>
            <LineChart data={windowPoints}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="distance" tickFormatter={(d: number | string) => `${Math.round(Number(d))}m`} />
              <YAxis />
              <ReTooltip {...tooltipProps} />
              <Line type="monotone" dataKey="speed" stroke="#007acc" dot={false} strokeWidth={2} activeDot={{ r: 5 }} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div style={{ height: 180 }}>
          <ResponsiveContainer>
            <LineChart data={windowPoints}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="distance" tickFormatter={(d: number | string) => `${Math.round(Number(d))}m`} />
              <YAxis />
              <ReTooltip {...tooltipProps} />
              <Line type="monotone" dataKey="accel" stroke="#e55353" dot={false} strokeWidth={2} activeDot={{ r: 5 }} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div style={{ height: 180 }}>
          <ResponsiveContainer>
            <LineChart data={windowPoints}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="distance" tickFormatter={(d: number | string) => `${Math.round(Number(d))}m`} />
              <YAxis />
              <ReTooltip {...tooltipProps} />
              <Line type="monotone" dataKey="energy" stroke="#22aa55" dot={false} strokeWidth={2} activeDot={{ r: 5 }} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
}
