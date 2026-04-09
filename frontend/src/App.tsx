import type { FormEvent, MouseEvent } from 'react'
import { useEffect, useMemo, useRef, useState } from 'react'
import './App.css'
import TelemetryGraph from './components/TelemetryGraph'
import {
  getDefaultPresets,
  type SimulationInputs,
  type SimulationPreset,
} from './services/getDefaultPresets'

type TelemetryPoint = {
  x: number
  y: number
  speed: number
  accel: number
  distance: number
}

type SimulateResponse = {
  distanceM?: number
  points?: TelemetryPoint[]
  ok?: boolean
  message?: string
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

type FieldTemplate = Omit<FieldDef, 'value'>

// UI field definitions live here, but the numeric defaults now live only in the backend.
const fieldTemplates: FieldTemplate[] = [
  { name: 'v', label: 'v (m/s)', step: '0.1' },
  { name: 'solarWhPerMin', label: 'solarWhPerMin', step: '0.1' },
  { name: 'etaDrive', label: 'etaDrive', step: '0.01' },
  { name: 'rWheel', label: 'rWheel (m)', step: '0.0001' },
  { name: 'tMax', label: 'tMax (N·m)', step: '0.1' },
  { name: 'pMax', label: 'pMax (W)', step: '1' },
  { name: 'm', label: 'm (kg)', step: '0.1' },
  { name: 'g', label: 'g (m/s^2)', step: '0.01' },
  { name: 'cRr', label: 'cRr', step: '0.0001' },
  { name: 'rho', label: 'rho', step: '0.001' },
  { name: 'cD', label: 'cD', step: '0.01' },
  { name: 'a', label: 'a (m^2)', step: '0.001' },
  { name: 'theta', label: 'theta (rad)', step: '0.001' },
  { name: 'gmax', label: 'gmax (lateral g limit)', step: '0.01', min: '0.1', max: '2.0' },
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
  'gmax',
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

const raceDayMinTemplate: FieldTemplate = { name: 'raceDayMin', label: 'raceDayMin', step: '1' }
const batteryWhTemplate: FieldTemplate = { name: 'batteryWh', label: 'batteryWh', step: '1' }

function createBlankField(field: FieldTemplate): FieldDef {
  return { ...field, value: '' }
}

function createBlankFields(): FieldDef[] {
  return fieldTemplates.map((field) => createBlankField(field))
}

function createFieldFromValue(field: FieldTemplate, value: number): FieldDef {
  return { ...field, value: String(value) }
}

function createFieldsFromInputs(inputs: SimulationInputs): FieldDef[] {
  return fieldTemplates.map((field) => {
    switch (field.name) {
      case 'v':
        return createFieldFromValue(field, inputs.v)
      case 'solarWhPerMin':
        return createFieldFromValue(field, inputs.solarWhPerMin)
      case 'etaDrive':
        return createFieldFromValue(field, inputs.etaDrive)
      case 'rWheel':
        return createFieldFromValue(field, inputs.rWheel)
      case 'tMax':
        return createFieldFromValue(field, inputs.tMax)
      case 'pMax':
        return createFieldFromValue(field, inputs.pMax)
      case 'm':
        return createFieldFromValue(field, inputs.m)
      case 'g':
        return createFieldFromValue(field, inputs.g)
      case 'cRr':
        return createFieldFromValue(field, inputs.cRr)
      case 'rho':
        return createFieldFromValue(field, inputs.rho)
      case 'cD':
        return createFieldFromValue(field, inputs.cD)
      case 'a':
        return createFieldFromValue(field, inputs.a)
      case 'theta':
        return createFieldFromValue(field, inputs.theta)
      case 'gmax':
        return createFieldFromValue(field, inputs.gmax)
      default:
        return createBlankField(field)
    }
  })
}

function createFormStateFromInputs(inputs: SimulationInputs): {
  fields: FieldDef[]
  batteryWh: FieldDef
  raceDayMin: FieldDef
} {
  return {
    fields: createFieldsFromInputs(inputs),
    batteryWh: createFieldFromValue(batteryWhTemplate, inputs.batteryWh),
    raceDayMin: createFieldFromValue(raceDayMinTemplate, inputs.raceDayMin),
  }
}

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

function formatTrackStatus(pointCount: number, wraparoundEnabled: boolean): string {
  return `Telemetry points: ${pointCount} · Wraparound ${wraparoundEnabled ? 'on' : 'off'}`
}

async function postSimulation(
  inputs: Record<string, number>,
  wraparoundEnabled: boolean,
): Promise<{ distanceM: number; points: TelemetryPoint[] }> {
  let response: Response

  try {
    response = await fetch('http://localhost:8080/simulate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ inputs, wraparound: wraparoundEnabled }),
    })
  } catch {
    throw new Error('Network error. Is the server running?')
  }

  let data: SimulateResponse
  try {
    data = (await response.json()) as SimulateResponse
  } catch {
    throw new Error('Request failed.')
  }

  if (
    !response.ok ||
    !data.ok ||
    typeof data.distanceM !== 'number' ||
    !Array.isArray(data.points)
  ) {
    throw new Error(data.message || 'Request failed.')
  }

  return { distanceM: data.distanceM, points: data.points }
}

function App() {
  const [inputsOpen, setInputsOpen] = useState<boolean>(false)

  const [fields, setFields] = useState<FieldDef[]>(() => createBlankFields())

  const [raceDayMin, setRaceDayMin] = useState<FieldDef>(() => createBlankField(raceDayMinTemplate))
  const [batteryWh, setBatteryWh] = useState<FieldDef>(() => createBlankField(batteryWhTemplate))
  const [presets, setPresets] = useState<SimulationPreset[]>([])
  const [selectedPresetId, setSelectedPresetId] = useState('')
  const [presetStatus, setPresetStatus] = useState('Loading presets...')

  const [result, setResult] = useState('--')
  const [status, setStatus] = useState('')
  const [trackStatus, setTrackStatus] = useState('Loading track...')
  const [telemetry, setTelemetry] = useState<TelemetryPoint[]>([])
  const trackWidth = 30
  const [wraparoundEnabled, setWraparoundEnabled] = useState(true)
  const [hoverPoint, setHoverPoint] = useState<HoverPoint | null>(null)
  const lastSimulationInputsRef = useRef<Record<string, number> | null>(null)

  const [tooltip, setTooltip] = useState<TooltipState>({
    visible: false,
    x: 0,
    y: 0,
    speed: 0,
    accel: 0,
    distance: 0,
  })

  const selectedPreset = presets.find((preset) => preset.id === selectedPresetId) ?? null

  useEffect(() => {
    let isMounted = true

    const loadPresets = async () => {
      try {
        const data = await getDefaultPresets()
        const preset =
          data.presets.find((item) => item.id === data.defaultPresetId) ?? data.presets[0]

        if (!isMounted) return

        setPresets(data.presets)

        if (!preset) {
          setPresetStatus('No backend presets available.')
          return
        }

        const nextFormState = createFormStateFromInputs(preset.inputs)

        setSelectedPresetId(preset.id)
        setFields(nextFormState.fields)
        setBatteryWh(nextFormState.batteryWh)
        setRaceDayMin(nextFormState.raceDayMin)
        setPresetStatus('')
      } catch (error) {
        console.error('Failed to load default presets', error)
        if (isMounted) setPresetStatus('Failed to load backend presets.')
      }
    }

    void loadPresets()

    return () => {
      isMounted = false
    }
  }, [])

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
        const lastSimulationInputs = lastSimulationInputsRef.current
        if (lastSimulationInputs) {
          const data = await postSimulation(lastSimulationInputs, wraparoundEnabled)

          if (isMounted) {
            setResult(Number(data.distanceM).toFixed(2))
            setTelemetry(data.points)
            setTrackStatus(formatTrackStatus(data.points.length, wraparoundEnabled))
          }
          return
        }

        //pause async func w/o freezing UI until backend response
        const response = await fetch(telemetryUrl(wraparoundEnabled))
        const data = await response.json()

        if (!response.ok || !Array.isArray(data.points)) {
          if (isMounted) setTrackStatus(data.message || 'Failed to load track telemetry.')
          return
        }

        if (isMounted) {
          setTelemetry(data.points)
          setTrackStatus(formatTrackStatus(data.points.length, wraparoundEnabled))
        }
      } catch (error) {
        if (isMounted) {
          setTrackStatus(
            error instanceof Error ? error.message : 'Unable to reach backend for track telemetry.',
          )
        }
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

  const handlePresetChange = (presetId: string) => {
    const preset = presets.find((item) => item.id === presetId)
    if (!preset) return

    const nextFormState = createFormStateFromInputs(preset.inputs)

    setSelectedPresetId(preset.id)
    setFields(nextFormState.fields)
    setBatteryWh(nextFormState.batteryWh)
    setRaceDayMin(nextFormState.raceDayMin)
  }

  const handlePresetReset = () => {
    if (!selectedPreset) return

    const nextFormState = createFormStateFromInputs(selectedPreset.inputs)

    setFields(nextFormState.fields)
    setBatteryWh(nextFormState.batteryWh)
    setRaceDayMin(nextFormState.raceDayMin)
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setStatus('Calculating...')

    const payload: Record<string, number> = {}

    for (const field of fields) {
      if (!DISTANCE_FIELD_NAMES.has(field.name)) continue
      if (field.value.trim() === '') continue
      const value = toNumber(field.value)
      if (value === null) {
        setStatus(`Invalid value for ${field.name}.`)
        return
      }
      payload[field.name] = value
    }

    if (raceDayMin.value.trim() !== '') {
      const raceMinValue = toNumber(raceDayMin.value)
      if (raceMinValue === null) {
        setStatus(`Invalid value for ${raceDayMin.name}.`)
        return
      }
      payload[raceDayMin.name] = raceMinValue
    }

    if (batteryWh.value.trim() !== '') {
      const batteryValue = toNumber(batteryWh.value)
      if (batteryValue === null) {
        setStatus(`Invalid value for ${batteryWh.name}.`)
        return
      }
      payload[batteryWh.name] = batteryValue
    }

    try {
      const data = await postSimulation(payload, wraparoundEnabled)

      lastSimulationInputsRef.current = payload
      setResult(data.distanceM.toFixed(2))
      setTelemetry(data.points)
      setTrackStatus(formatTrackStatus(data.points.length, wraparoundEnabled))
      setStatus('Success.')
    } catch (error) {
      setStatus(error instanceof Error ? error.message : 'Network error. Is the server running?')
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
              disabled={presets.length === 0}
              aria-label="Vehicle preset"
              onChange={(event) => handlePresetChange(event.target.value)}
            >
              {presets.length === 0 ? (
                <option value="">{presetStatus || 'Backend presets pending'}</option>
              ) : (
                presets.map((preset) => (
                  <option key={preset.id} value={preset.id}>
                    {preset.label}
                  </option>
                ))
              )}
            </select>
          </label>

          <button type="button" className="toggle" onClick={() => setInputsOpen((v) => !v)}>
            {inputsOpen ? 'Hide inputs' : 'Show inputs'}
          </button>

          <button type="button" className="reset" disabled={!selectedPreset} onClick={handlePresetReset}>
            Reset to preset
          </button>
        </div>
        {presetStatus ? <div className="status">{presetStatus}</div> : null}

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
