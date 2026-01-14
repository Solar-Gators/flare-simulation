import type { FormEvent, MouseEvent } from 'react'
import { useEffect, useMemo, useState } from 'react'
import './App.css'

type TelemetryPoint = {
  x: number
  y: number
  speed: number
  accel: number
  distance: number
}

type TooltipState = {
  visible: boolean
  x: number
  y: number
  speed: number
  accel: number
  distance: number
}

type FieldDef = {
  name: string
  label: string
  step: string
  value: string
}

const initialFields: FieldDef[] = [
  { name: 'v', label: 'v (m/s)', step: '0.1', value: '20' },
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

function App() {
  const [fields, setFields] = useState(initialFields)
  const [result, setResult] = useState('--')
  const [status, setStatus] = useState('')
  const [trackStatus, setTrackStatus] = useState('Loading track...')
  const [telemetry, setTelemetry] = useState<TelemetryPoint[]>([])
  const [tooltip, setTooltip] = useState<TooltipState>({
    visible: false,
    x: 0,
    y: 0,
    speed: 0,
    accel: 0,
    distance: 0,
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

    const nextSegments = telemetry.slice(1).map((point, index) => {
      const prev = telemetry[index]
      return {
        d: `M ${prev.x} ${prev.y} L ${point.x} ${point.y}`,
        speed: point.speed,
        accel: point.accel,
        distance: point.distance,
        color: speedToColor(point.speed, minSpeed, maxSpeed),
      }
    })

    return { segments: nextSegments, viewBox: nextViewBox, speedRange: [minSpeed, maxSpeed] as const }
  }, [telemetry])

  useEffect(() => {
    let isMounted = true
    async function loadTelemetry() {
      try {
        const response = await fetch('http://localhost:8080/track/telemetry')
        const data = await response.json()
        if (!response.ok || !Array.isArray(data.points)) {
          if (isMounted) setTrackStatus('Failed to load track telemetry.')
          return
        }
        if (isMounted) {
          setTelemetry(data.points)
          setTrackStatus(`Telemetry points: ${data.points.length}`)
        }
      } catch {
        if (isMounted) setTrackStatus('Unable to reach backend for track telemetry.')
      }
    }

    loadTelemetry()
    return () => {
      isMounted = false
    }
  }, [])

  const handleInputChange = (name: string, value: string) => {
    setFields((prev) =>
      prev.map((field) => (field.name === name ? { ...field, value } : field)),
    )
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatus('Calculating...')

    const payload: Record<string, number> = {}
    for (const field of fields) {
      const value = toNumber(field.value)
      if (value === null) {
        setStatus(`Invalid value for ${field.name}.`)
        return
      }
      payload[field.name] = value
    }

    try {
      const response = await fetch('http://localhost:8080/distance', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      const data = await response.json()
      if (!response.ok || !data.ok) {
        setStatus(data.message || 'Request failed.')
        return
      }
      setResult(Number(data.distanceM).toFixed(2))
      setStatus('Success.')
    } catch {
      setStatus('Network error. Is the server running?')
    }
  }

  const handleSegmentMove = (
    event: MouseEvent<SVGPathElement>,
    speed: number,
    accel: number,
    distance: number,
  ) => {
    setTooltip({
      visible: true,
      x: event.clientX + 12,
      y: event.clientY + 12,
      speed,
      accel,
      distance,
    })
  }

  const handleSegmentLeave = () => {
    setTooltip((prev) => ({ ...prev, visible: false }))
  }

  return (
    <div className="shell">
      <header>
        <h1>Flare Distance Calculator</h1>
        <p>Enter your simulation values and fetch the predicted distance.</p>
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
                  name={field.name}
                  value={field.value}
                  onChange={(event) => handleInputChange(field.name, event.target.value)}
                />
              </label>
            ))}
          </div>
          <div className="actions">
            <button type="submit">Compute Distance</button>
            <div className="result">
              Distance: <strong>{result}</strong> m
            </div>
            <div className="status">{status}</div>
          </div>
        </form>
      </section>

      <section className="panel">
        <h2>Track Preview</h2>
        <svg className="track-frame" viewBox={viewBox} role="img" aria-label="Track visualization">
          <g className="track-layer">
            {segments.map((segment, index) => (
              <path
                key={`${segment.distance}-${index}`}
                d={segment.d}
                fill="none"
                stroke={segment.color}
                strokeWidth={6}
                strokeLinecap="round"
                onMouseMove={(event) =>
                  handleSegmentMove(event, segment.speed, segment.accel, segment.distance)
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