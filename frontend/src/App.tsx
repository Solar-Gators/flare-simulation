import type { FormEvent, MouseEvent } from 'react'
import { useEffect, useMemo, useState } from 'react'
import './App.css'
import TelemetryGraph from './components/TelemetryGraph'

type TelemetryPoint = {
  x: number
  y: number
  speed: number
  accel: number
  distance: number
}

type TrackSegment = {
  key: string
  d: string
  speed: number
  accel: number
  distance: number
  color: string
  x: number
  y: number
}

type HoverPoint = {
  x: number
  y: number
  color: string
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
  min?: string
  max?: string
}

// input fields for distance calculator (no solarWhPerMin, no raceDayMin, no batteryWh here)
const initialFields: FieldDef[] = [
  { name: 'v', label: 'v (m/s)', step: '0.1', value: '20' },
  { name: 'etaDrive', label: 'etaDrive', step: '0.01', value: '0.9' },
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
  {
    name: 'gmax',
    label: 'gmax (lateral g limit)',
    step: '0.01',
    value: '1.00',
    min: '0.1',
    max: '2.0',
  },
]

const ABS_COLOR_MIN_SPEED = 0.0
const ABS_COLOR_MAX_SPEED = 24.0
const ABS_COLOR_TICKS = [0, 6, 12, 18, 24] as const

const DISTANCE_FIELD_NAMES = new Set([
  'v',
  'batteryWh',
  'solarWhPerMin',
  'etaDrive',
  'raceDayMin',
  'rWheel',
  'tMax',
  'pMax',
  'm',
  'g',
  'cRr',
  'rho',
  'cD',
  'a',
  'theta',
])

const ABS_SPEED_COLOR_STOPS = [
  { speed: 0, color: [25, 50, 95] },
  { speed: 6, color: [37, 99, 235] },
  { speed: 12, color: [6, 182, 212] },
  { speed: 18, color: [245, 158, 11] },
  { speed: 24, color: [220, 38, 38] },
] as const

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

// independent fields (outside the expandable inputs)
const initialRaceDayMin: FieldDef = {
  name: 'raceDayMin',
  label: 'raceDayMin',
  step: '1',
  value: '480',
}
const initialBatteryWh: FieldDef = {
  name: 'batteryWh',
  label: 'batteryWh',
  step: '1',
  value: '5000',
}

type VehiclePreset = {
  id: string
  label: string
  fields: FieldDef[]
  outside?: Partial<Record<'batteryWh' | 'raceDayMin', string>>
}

const VEHICLE_PRESETS: VehiclePreset[] = [
  {
    id: 'flare-default',
    label: 'Flare (default)',
    fields: initialFields,
    outside: { batteryWh: '5000', raceDayMin: '480' },
  },
  {
    id: 'lexus-gs350-awd',
    label: '2008 Lexus GS350 AWD (215/55R17 road tires)',
    outside: { batteryWh: '656000', raceDayMin: '480' },
    fields: initialFields.map((f) => {
      switch (f.name) {
        case 'v':
          return { ...f, value: '20' }
        case 'etaDrive':
          return { ...f, value: '0.22' }
        case 'rWheel':
          return { ...f, value: '0.334' }
        case 'tMax':
          return { ...f, value: '5725' }
        case 'pMax':
          return { ...f, value: '880000' }
        case 'm':
          return { ...f, value: '1840' }
        case 'g':
          return { ...f, value: '9.81' }
        case 'cRr':
          return { ...f, value: '0.010' }
        case 'rho':
          return { ...f, value: '1.225' }
        case 'cD':
          return { ...f, value: '0.27' }
        case 'a':
          return { ...f, value: '2.2' }
        case 'theta':
          return { ...f, value: '0' }
        case 'gmax':
          return { ...f, value: '0.88', min: f.min ?? '0.1', max: f.max ?? '2.0' }
        default:
          return f
      }
    }),
  },
]

function toNumber(value: string): number | null {
  const num = Number.parseFloat(value)
  return Number.isFinite(num) ? num : null
}

function mixChannel(a: number, b: number, t: number): number {
  return Math.round(a + (b - a) * t)
}

function rgbToCss(color: readonly number[]): string {
  return `rgb(${color[0]}, ${color[1]}, ${color[2]})`
}

function speedToColor(speed: number): string {
  const clampedSpeed = clamp(speed, ABS_COLOR_MIN_SPEED, ABS_COLOR_MAX_SPEED)

  for (let index = 1; index < ABS_SPEED_COLOR_STOPS.length; index += 1) {
    const prev = ABS_SPEED_COLOR_STOPS[index - 1]
    const next = ABS_SPEED_COLOR_STOPS[index]
    if (clampedSpeed <= next.speed) {
      const t = (clampedSpeed - prev.speed) / Math.max(1e-6, next.speed - prev.speed)
      const color = prev.color.map((channel, channelIndex) =>
        mixChannel(channel, next.color[channelIndex], t),
      )
      return rgbToCss(color)
    }
  }

  return rgbToCss(ABS_SPEED_COLOR_STOPS[ABS_SPEED_COLOR_STOPS.length - 1].color)
}

const ABS_SPEED_LEGEND_GRADIENT = `linear-gradient(90deg, ${ABS_SPEED_COLOR_STOPS.map((stop) => {
  const offset =
    ((stop.speed - ABS_COLOR_MIN_SPEED) /
      Math.max(1e-6, ABS_COLOR_MAX_SPEED - ABS_COLOR_MIN_SPEED)) *
    100
  return `${rgbToCss(stop.color)} ${offset.toFixed(1)}%`
}).join(', ')})`

function telemetryUrl(wraparoundEnabled: boolean): string {
  const url = new URL('http://localhost:8080/track/telemetry')
  url.searchParams.set('wraparound', String(wraparoundEnabled))
  return url.toString()
}

function App() {
  const [selectedPresetId, setSelectedPresetId] = useState<string>(VEHICLE_PRESETS[0].id)
  const [inputsOpen, setInputsOpen] = useState<boolean>(false)

  const [fields, setFields] = useState<FieldDef[]>(() =>
    VEHICLE_PRESETS[0].fields.map((f) => ({ ...f })),
  )

  const [raceDayMin, setRaceDayMin] = useState<FieldDef>(() => ({ ...initialRaceDayMin }))
  const [batteryWh, setBatteryWh] = useState<FieldDef>(() => ({ ...initialBatteryWh }))

  const [result, setResult] = useState('--')
  const [status, setStatus] = useState('')
  const [trackStatus, setTrackStatus] = useState('Loading track...')
  const [telemetry, setTelemetry] = useState<TelemetryPoint[]>([])
  const trackWidth = 30
  const [wraparoundEnabled, setWraparoundEnabled] = useState(true)
  const [hoverPoint, setHoverPoint] = useState<HoverPoint | null>(null)

  const [tooltip, setTooltip] = useState<TooltipState>({
    visible: false,
    x: 0,
    y: 0,
    speed: 0,
    accel: 0,
    distance: 0,
  })

  const applyPreset = (presetId: string) => {
    const preset = VEHICLE_PRESETS.find((p) => p.id === presetId) ?? VEHICLE_PRESETS[0]

    setSelectedPresetId(preset.id)
    setFields(preset.fields.map((f) => ({ ...f })))

    // also apply "outside" fields if provided
    if (preset.outside?.raceDayMin != null) {
      setRaceDayMin((prev) => ({ ...prev, value: preset.outside!.raceDayMin! }))
    }
    if (preset.outside?.batteryWh != null) {
      setBatteryWh((prev) => ({ ...prev, value: preset.outside!.batteryWh! }))
    }
  }

  const { segments, viewBox, speedRange } = useMemo(() => {
    if (telemetry.length < 2) {
      return { segments: [] as TrackSegment[], viewBox: '0 0 600 360', speedRange: [0, 1] as const }
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

    const nextSegments: TrackSegment[] = telemetry.slice(1).map((point, index) => {
      const prev = telemetry[index]
      return {
        key: `${prev.distance}-${point.distance}-${index}`,
        d: `M ${prev.x} ${prev.y} L ${point.x} ${point.y}`,
        speed: point.speed,
        accel: point.accel,
        distance: point.distance,
        color: speedToColor(point.speed),
        x: point.x,
        y: point.y,
      }
    })

    return {
      segments: nextSegments,
      viewBox: nextViewBox,
      speedRange: [minSpeed, maxSpeed] as const,
    }
  }, [telemetry])

  const startPoint = telemetry.length > 0 ? telemetry[0] : null
  const endPoint = telemetry.length > 0 ? telemetry[telemetry.length - 1] : null
  const startMarkerColor = startPoint
    ? speedToColor(startPoint.speed)
    : rgbToCss(ABS_SPEED_COLOR_STOPS[0].color)

  useEffect(() => {
    let isMounted = true

    async function loadTelemetry() {
      try {
        //pause async func w/o freezing UI until backend response
        const response = await fetch(telemetryUrl(wraparoundEnabled))
        const data = await response.json()

        if (!response.ok || !Array.isArray(data.points)) {
          if (isMounted) setTrackStatus(data.message || 'Failed to load track telemetry.')
          return
        }

        if (isMounted) {
          setTelemetry(data.points)
          setTrackStatus(
            `Telemetry points: ${data.points.length} · Wraparound ${wraparoundEnabled ? 'on' : 'off'}`,
          )
        }
      } catch {
        if (isMounted) setTrackStatus('Unable to reach backend for track telemetry.')
      }
    }

    loadTelemetry()
    return () => {
      isMounted = false
    }
  }, [wraparoundEnabled])

  const handleInputChange = (name: string, value: string) => {
    setFields((prev) => prev.map((field) => (field.name === name ? { ...field, value } : field)))
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatus('Calculating...')

    const payload: Record<string, number> = {}

    for (const field of fields) {
      if (!DISTANCE_FIELD_NAMES.has(field.name)) continue
      const value = toNumber(field.value)
      if (value === null) {
        setStatus(`Invalid value for ${field.name}.`)
        return
      }
      payload[field.name] = value
    }

    const raceMinValue = toNumber(raceDayMin.value)
    if (raceMinValue === null) {
      setStatus(`Invalid value for ${raceDayMin.name}.`)
      return
    }
    payload[raceDayMin.name] = raceMinValue

    const batteryValue = toNumber(batteryWh.value)
    if (batteryValue === null) {
      setStatus(`Invalid value for ${batteryWh.name}.`)
      return
    }
    payload[batteryWh.name] = batteryValue

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
    x: number,
    y: number,
    color: string,
  ) => {
    setTooltip({
      visible: true,
      x: event.clientX + 12,
      y: event.clientY + 12,
      speed,
      accel,
      distance,
    })
    setHoverPoint({ x, y, color })
  }

  const handleSegmentLeave = () => {
    setTooltip((prev) => ({ ...prev, visible: false }))
    setHoverPoint(null)
  }

  return (
    <div className="shell">
      <header>
        <h1>Flare Distance Calculator</h1>
        <p>Enter your simulation values and fetch the predicted distance.</p>
      </header>

      <section className="panel">
        <div className="preset-row">
          <label>
            Vehicle preset
            <select
              value={selectedPresetId}
              onChange={(e) => applyPreset(e.target.value)}
              aria-label="Select vehicle preset"
            >
              {VEHICLE_PRESETS.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.label}
                </option>
              ))}
            </select>
          </label>

          <button type="button" className="toggle" onClick={() => setInputsOpen((v) => !v)}>
            {inputsOpen ? 'Hide inputs' : 'Show inputs'}
          </button>

          <button type="button" className="reset" onClick={() => applyPreset(selectedPresetId)}>
            Reset to preset
          </button>
        </div>

        {/* batteryWh + raceDayMin outside the expandable inputs */}
        <div className="preset-row" style={{ marginTop: 12 }}>
          <label>
            {batteryWh.label}
            <input
              type="number"
              step={batteryWh.step}
              name={batteryWh.name}
              value={batteryWh.value}
              onChange={(event) => setBatteryWh((prev) => ({ ...prev, value: event.target.value }))}
            />
          </label>

          <label>
            {raceDayMin.label}
            <input
              type="number"
              step={raceDayMin.step}
              name={raceDayMin.name}
              value={raceDayMin.value}
              onChange={(event) =>
                setRaceDayMin((prev) => ({ ...prev, value: event.target.value }))
              }
            />
          </label>
        </div>

        <form id="distance-form" onSubmit={handleSubmit}>
          {inputsOpen ? (
            <div className="inputs-block">
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
                      onChange={(event) => handleInputChange(field.name, event.target.value)}
                    />
                  </label>
                ))}
              </div>
            </div>
          ) : null}

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
        <div className="track-controls">
          <label className="toggle-label">
            <input
              type="checkbox"
              checked={wraparoundEnabled}
              onChange={(event) => setWraparoundEnabled(event.target.checked)}
            />
            <span>Wraparound continuation</span>
          </label>
        </div>
        <svg className="track-frame" viewBox={viewBox} role="img" aria-label="Track visualization">
          <g className="track-layer">
            {segments.map((seg) => (
              <path
                key={seg.key}
                d={seg.d}
                fill="none"
                stroke={seg.color}
                strokeWidth={trackWidth}
                strokeLinecap="round"
                pointerEvents="stroke"
                onMouseMove={(e) =>
                  handleSegmentMove(e, seg.speed, seg.accel, seg.distance, seg.x, seg.y, seg.color)
                }
                onMouseLeave={handleSegmentLeave}
              />
            ))}
            {hoverPoint ? (
              <circle
                className="track-hover"
                cx={hoverPoint.x}
                cy={hoverPoint.y}
                r={6}
                fill="#ffffff"
                stroke={hoverPoint.color}
                strokeWidth={3}
                pointerEvents="none"
              />
            ) : null}
            {startPoint ? (
              <>
                <circle
                  className="track-start-marker"
                  cx={startPoint.x}
                  cy={startPoint.y}
                  r={10}
                  fill="#fffdf8"
                  stroke={startMarkerColor}
                  strokeWidth={4}
                  pointerEvents="none"
                />
                <circle
                  className="track-start-marker"
                  cx={startPoint.x}
                  cy={startPoint.y}
                  r={3.5}
                  fill={startMarkerColor}
                  pointerEvents="none"
                />
                <text
                  className="track-start-label"
                  x={startPoint.x}
                  y={startPoint.y - 16}
                  textAnchor="middle"
                  pointerEvents="none"
                >
                  S/F
                </text>
              </>
            ) : null}
          </g>
        </svg>

        <div className="speed-legend" aria-label="Absolute speed legend">
          <div className="speed-legend-title">Absolute speed color scale</div>
          <div
            className="speed-legend-bar"
            style={{ backgroundImage: ABS_SPEED_LEGEND_GRADIENT }}
          />
          <div className="speed-legend-scale">
            {ABS_COLOR_TICKS.map((tick) => (
              <span key={tick}>{tick.toFixed(0)} m/s</span>
            ))}
          </div>
        </div>

        <div className="track-meta">
          {trackStatus} · Speed range {speedRange[0].toFixed(2)}–{speedRange[1].toFixed(2)} m/s
        </div>
        {startPoint && endPoint ? (
          <div className="track-meta">
            Start speed {startPoint.speed.toFixed(2)} m/s · End speed {endPoint.speed.toFixed(2)}{' '}
            m/s
          </div>
        ) : null}
        <div style={{ marginTop: 12 }}>
          <TelemetryGraph telemetry={telemetry} />
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
