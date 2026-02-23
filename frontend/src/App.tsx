import type { FormEvent, MouseEvent } from 'react'
import { useMemo, useState } from 'react'
import './App.css'

type TelemetryPoint = {
  x: number
  y: number
  speed: number
  accel: number
  distance: number
  vCap: number
}

type TooltipState = {
  visible: boolean
  x: number
  y: number
  speed: number
  accel: number
  distance: number
  vCap: number
}

type FieldDef = {
  name: string
  label: string
  step: string
  value: string
  min?: string
  max?: string
}

const initialFields: FieldDef[] = [
  { name: 'batteryWh', label: 'batteryWh', step: '1', value: '5000' },
  { name: 'solarWhPerMin', label: 'solarWhPerMin', step: '0.1', value: '5' },
  { name: 'etaDrive', label: 'etaDrive', step: '0.01', value: '0.9' },
  { name: 'raceDayMin', label: 'raceDayMin', step: '1', value: '480' },
  { name: 'rWheel', label: 'rWheel (m)', step: '0.0001', value: '0.2792' },
  { name: 'tMax', label: 'tMax (N·m)', step: '0.1', value: '45' },
  { name: 'pMax', label: 'pMax (W)', step: '1', value: '10000' },
  { name: 'm', label: 'm (kg)', step: '0.1', value: '285' },
  { name: 'g', label: 'g (m/s^2)', step: '0.01', value: '9.81' },
  { name: 'cRr', label: 'cRr', step: '0.0001', value: '0.0015' },
  { name: 'rho', label: 'rho', step: '0.001', value: '1.225' },
  { name: 'cD', label: 'cD', step: '0.01', value: '0.21' },
  { name: 'a', label: 'a (m^2)', step: '0.001', value: '0.456' },
  { name: 'theta', label: 'theta (rad)', step: '0.001', value: '0' },
]

function toNumber(value: string): number | null {
  const num = Number.parseFloat(value)
  return Number.isFinite(num) ? num : null
}

function speedToColor(speed: number, minSpeed: number, maxSpeed: number): string {
  const clamped = Math.max(minSpeed, Math.min(speed, maxSpeed))
  const t = (clamped - minSpeed) / Math.max(1e-6, maxSpeed - minSpeed)
  const hue = 210 - 210 * t
  return `hsl(${hue}, 80%, 48%)`
}

function lapDistanceFromTelemetry(points: TelemetryPoint[]): number | null {
  if (!points || points.length < 2) return null
  const last = points[points.length - 1]
  return Number.isFinite(last.distance) && last.distance > 0 ? last.distance : null
}

async function postDistance(payload: Record<string, number>, wraparound: boolean) {
  const response = await fetch('http://localhost:8080/distance', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...payload, wraparound }),
  })
  const data = await response.json()
  if (!response.ok || !data.ok) {
    throw new Error(data?.message || 'Request failed.')
  }
  return data as { distanceM: number; optimalV: number; remainingWh: number; ok: boolean }
}

async function postTelemetry(
  payload: Record<string, number>,
  wraparound: boolean,
  baseTarget: number,
) {
  const response = await fetch('http://localhost:8080/track/telemetry', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...payload, wraparound, baseTarget }),
  })
  const data = await response.json()
  if (!response.ok || !data.ok || !Array.isArray(data.points)) {
    throw new Error(data?.message || 'Failed to load telemetry.')
  }
  return data.points as TelemetryPoint[]
}

function App() {
  const [fields, setFields] = useState(initialFields)
  const [wraparoundLookahead, setWraparoundLookahead] = useState(true)

  const [result, setResult] = useState('--')
  const [optimalV, setOptimalV] = useState<number | null>(null)
  const [remainingWh, setRemainingWh] = useState<number | null>(null)

  const [status, setStatus] = useState('Fill inputs and press Compute.')
  const [trackStatus, setTrackStatus] = useState('Track not loaded yet.')
  const [telemetry, setTelemetry] = useState<TelemetryPoint[]>([])

  const [lapDistance, setLapDistance] = useState<number | null>(null)
  const [laps, setLaps] = useState<number | null>(null)

  const [tooltip, setTooltip] = useState<TooltipState>({
    visible: false,
    x: 0,
    y: 0,
    speed: 0,
    accel: 0,
    distance: 0,
    vCap: 0,
  })

  const { segments, viewBox, speedRange } = useMemo(() => {
    if (telemetry.length < 2) {
      return { segments: [], viewBox: '0 0 600 360', speedRange: [0, 1] as const }
    }

    let minX = telemetry[0].x
    let maxX = telemetry[0].x
    let minY = telemetry[0].y
    let maxY = telemetry[0].y
    let minSpeed = telemetry[0].speed
    let maxSpeed = telemetry[0].speed

    for (const pt of telemetry) {
      minX = Math.min(minX, pt.x)
      maxX = Math.max(maxX, pt.x)
      minY = Math.min(minY, pt.y)
      maxY = Math.max(maxY, pt.y)
      minSpeed = Math.min(minSpeed, pt.speed)
      maxSpeed = Math.max(maxSpeed, pt.speed)
    }

    const padding = 40
    const width = Math.max(1, maxX - minX)
    const height = Math.max(1, maxY - minY)
    const nextViewBox = [
      minX - padding,
      minY - padding,
      width + padding * 2,
      height + padding * 2,
    ].join(' ')

    const nextSegments = telemetry.slice(1).map((pt, i) => {
      const prev = telemetry[i]
      return {
        key: `${prev.distance}-${pt.distance}-${i}`,
        d: `M ${prev.x} ${prev.y} L ${pt.x} ${pt.y}`,
        speed: pt.speed,
        accel: pt.accel,
        distance: pt.distance,
        vCap: pt.vCap,
        color: speedToColor(pt.speed, minSpeed, maxSpeed),
      }
    })

    return {
      segments: nextSegments,
      viewBox: nextViewBox,
      speedRange: [minSpeed, maxSpeed] as const,
    }
  }, [telemetry])

  const handleInputChange = (name: string, value: string) => {
    setFields((prev) => prev.map((field) => (field.name === name ? { ...field, value } : field)))
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatus('Computing...')
    setTrackStatus('Updating track...')

    const payload: Record<string, number> = {}
    for (const field of fields) {
      const value = toNumber(field.value)
      if (value === null) {
        setStatus(`Invalid value for ${field.name}.`)
        setTrackStatus('Track not updated.')
        return
      }
      payload[field.name] = value
    }

    try {
      const distResp = await postDistance(payload, wraparoundLookahead)

      setResult(Number(distResp.distanceM).toFixed(2))
      setOptimalV(distResp.optimalV)
      setRemainingWh(distResp.remainingWh)

      const points = await postTelemetry(payload, wraparoundLookahead, distResp.optimalV)
      setTelemetry(points)

      const lapM = lapDistanceFromTelemetry(points)
      setLapDistance(lapM)

      const totalDist = distResp.distanceM
      setLaps(lapM ? totalDist / lapM : null)

      setStatus('Success.')
      setTrackStatus(`Telemetry points: ${points.length}`)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Request failed.'
      setStatus(msg)
      setTrackStatus(msg)
    }
  }

  const handleSegmentMove = (
    event: MouseEvent<SVGPathElement>,
    speed: number,
    accel: number,
    distance: number,
    vCap: number,
  ) => {
    setTooltip({
      visible: true,
      x: event.clientX + 12,
      y: event.clientY + 12,
      speed,
      accel,
      distance,
      vCap,
    })
  }

  const handleSegmentLeave = () => {
    setTooltip((prev) => ({ ...prev, visible: false }))
  }

  return (
    <div className="shell">
      <header>
        <h1>Flare Distance Calculator</h1>
        <p>Press Compute to optimize speed for full battery depletion by race end.</p>
      </header>

      <section className="panel">
        <form id="distance-form" onSubmit={handleSubmit}>
          <div className="grid">
            {fields.map((field) => (
              <label key={field.name}>
                {field.label}
                <input
                  type="number"
                  step={field.step}
                  min={field.min}
                  max={field.max}
                  name={field.name}
                  value={field.value}
                  onChange={(e) => handleInputChange(field.name, e.target.value)}
                />
              </label>
            ))}
          </div>

          <div className="toggles">
            <label>
              <input
                type="checkbox"
                checked={wraparoundLookahead}
                onChange={(e) => setWraparoundLookahead(e.target.checked)}
              />
              Wraparound lookahead (if off: start from 0 speed)
            </label>
          </div>

          <div className="actions">
            <button type="submit">Compute</button>
            <div className="result">
              Laps: <strong>{laps === null ? '--' : laps.toFixed(2)}</strong>
            </div>
            <div className="status">
              {status} {lapDistance !== null && <small>Lap {lapDistance.toFixed(1)} m</small>}{' '}
              {optimalV !== null && <small>Optimal v {optimalV.toFixed(2)} m/s</small>}{' '}
              {remainingWh !== null && <small>Remaining {remainingWh.toFixed(1)} Wh</small>}{' '}
              <small>Distance {result} m</small>
            </div>
          </div>
        </form>
      </section>

      <section className="panel">
        <h2>Track Preview</h2>
        <svg className="track-frame" viewBox={viewBox} role="img" aria-label="Track visualization">
          <g className="track-layer">
            {segments.map((seg) => (
              <path
                key={seg.key}
                d={seg.d}
                fill="none"
                stroke={seg.color}
                strokeWidth={6}
                strokeLinecap="round"
                onMouseMove={(e) =>
                  handleSegmentMove(e, seg.speed, seg.accel, seg.distance, seg.vCap)
                }
                onMouseLeave={handleSegmentLeave}
              />
            ))}
          </g>
        </svg>
        <div className="track-meta">
          {trackStatus} · Speed range {speedRange[0].toFixed(2)}–{speedRange[1].toFixed(2)} m/s
        </div>
      </section>

      <div
        className={`tooltip ${tooltip.visible ? 'visible' : ''}`}
        style={{ left: tooltip.x, top: tooltip.y }}
      >
        <div>
          Speed: <strong>{tooltip.speed.toFixed(2)}</strong> m/s
        </div>
        <div>Accel: {tooltip.accel.toFixed(3)} m/s²</div>
        <div>Dist: {tooltip.distance.toFixed(1)} m</div>
      </div>
    </div>
  )
}

export default App
