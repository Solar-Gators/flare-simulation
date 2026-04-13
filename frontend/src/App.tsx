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
  curveSpeedCap: number
}

type SimulateResponse = {
  distanceM?: number
  optimalV?: number
  remainingEnergyWh?: number
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
  curveSpeedCap: number
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
  curveSpeedCap: number
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
  { name: 'etaDrive', label: 'Drivetrain Efficiency (%)', step: '0.01', value: '' },
  { name: 'rWheel', label: 'Wheel Radius (m)', step: '0.0001', value: '' },
  { name: 'tMax', label: 'Max Motor Torque (N·m)', step: '0.1', value: '' },
  { name: 'pMax', label: 'Max Motor Power (W)', step: '1', value: '' },
  { name: 'm', label: 'Mass (kg)', step: '0.1', value: '' },
  { name: 'g', label: 'Gravity (m/s^2)', step: '0.01', value: '' },
  { name: 'cRr', label: 'Rolling Resistance Coefficient', step: '0.0001', value: '' },
  { name: 'rho', label: 'rho', step: '0.001', value: '' },
  { name: 'cD', label: 'Drag Coefficient', step: '0.01', value: '' },
  { name: 'a', label: 'Frontal Area (m^2)', step: '0.001', value: '' },
  { name: 'theta', label: 'Track Grade (rad)', step: '0.001', value: '' },
  { name: 'gmax', label: 'Lateral G-force Limit', step: '0.01', value: '' },
]

const ABS_COLOR_MIN_SPEED = 0.0
const ABS_COLOR_MAX_SPEED = 28.0
const ABS_COLOR_TICKS = [0, 7, 14, 21, 28] as const

const DISTANCE_FIELD_NAMES = new Set([
  'batteryWh',
  'additionalEfficiency',
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
  { speed: 7, color: [37, 99, 235] },
  { speed: 14, color: [6, 182, 212] },
  { speed: 21, color: [245, 158, 11] },
  { speed: 28, color: [220, 38, 38] },
] as const

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

// independent fields (outside the expandable inputs)
const initialRaceDayMin: FieldDef = {
  name: 'raceDayMin',
  label: 'Race Day Time (min)',
  step: '1',
  value: '',
}
const initialBatteryWh: FieldDef = {
  name: 'batteryWh',
  label: 'Battery Power (Wh)',
  step: '1',
  value: '',
}
const initialAdditionalEfficiency: FieldDef = {
  name: 'additionalEfficiency',
  label: 'Additional Efficiency (%)',
  step: '1',
  value: '',
  min: '-100.00',
  max: '100.00',
}

function createBlankField(field: FieldDef): FieldDef {
  return { ...field }
}

function createBlankFields(): FieldDef[] {
  return initialFields.map((field) => createBlankField(field))
}

function createFieldFromValue(field: FieldDef, value: number): FieldDef {
  return { ...field, value: String(value) }
}

function createFieldsFromInputs(inputs: SimulationInputs): FieldDef[] {
  return initialFields.map((field) => {
    switch (field.name) {
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
  additionalEfficiency: FieldDef
} {
  return {
    fields: createFieldsFromInputs(inputs),
    batteryWh: createFieldFromValue(initialBatteryWh, inputs.batteryWh),
    raceDayMin: createFieldFromValue(initialRaceDayMin, inputs.raceDayMin),
    additionalEfficiency: createFieldFromValue(
      initialAdditionalEfficiency,
      inputs.additionalEfficiency,
    ),
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
): Promise<{
  distanceM: number
  points: TelemetryPoint[]
  optimalV: number | null
  remainingEnergyWh: number | null
}> {
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

  return {
    distanceM: data.distanceM,
    points: data.points,
    optimalV: data.optimalV ?? null,
    remainingEnergyWh: data.remainingEnergyWh ?? null,
  }
}

function App() {
  const [inputsOpen, setInputsOpen] = useState<boolean>(false)

  const [fields, setFields] = useState<FieldDef[]>(() => createBlankFields())

  const [raceDayMin, setRaceDayMin] = useState<FieldDef>(() => createBlankField(initialRaceDayMin))
  const [batteryWh, setBatteryWh] = useState<FieldDef>(() => createBlankField(initialBatteryWh))
  const [presets, setPresets] = useState<SimulationPreset[]>([])
  const [selectedPresetId, setSelectedPresetId] = useState('')
  const [presetStatus, setPresetStatus] = useState('Loading presets...')
  const [additionalEfficiency, setAdditionalEfficiency] = useState<FieldDef>(() => ({
    ...initialAdditionalEfficiency,
  }))
  const [telemetryAdditionalEfficiency, setTelemetryAdditionalEfficiency] = useState(0)

  const [result, setResult] = useState('--')
  const [rawDistanceM, setRawDistanceM] = useState<number | null>(null)
  const [remainingEnergyWh, setRemainingEnergyWh] = useState<number | null>(null)
  const [optimalSpeedMps, setOptimalSpeedMps] = useState<number | null>(null)
  const [graphsOpen, setGraphsOpen] = useState(false)
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
    curveSpeedCap: 0,
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
        setAdditionalEfficiency(nextFormState.additionalEfficiency)
        setTelemetryAdditionalEfficiency(preset.inputs.additionalEfficiency)
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
        curveSpeedCap: point.curveSpeedCap,
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
            const lapLen = data.points.length > 0 ? data.points[data.points.length - 1].distance : 0
            setResult(lapLen > 0 ? (data.distanceM / lapLen).toFixed(2) : '--')
            setRawDistanceM(data.distanceM)
            setRemainingEnergyWh(data.remainingEnergyWh)
            setOptimalSpeedMps(data.optimalV)
            setTelemetry(data.points)
            setTelemetryAdditionalEfficiency(lastSimulationInputs.additionalEfficiency ?? 0)
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
    setAdditionalEfficiency(nextFormState.additionalEfficiency)
  }

  const handlePresetReset = () => {
    if (!selectedPreset) return

    const nextFormState = createFormStateFromInputs(selectedPreset.inputs)

    setFields(nextFormState.fields)
    setBatteryWh(nextFormState.batteryWh)
    setRaceDayMin(nextFormState.raceDayMin)
    setAdditionalEfficiency(nextFormState.additionalEfficiency)
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

    const effValue = toNumber(additionalEfficiency.value)
    if (effValue === null || effValue < -100 || effValue > 100) {
      setStatus(`Invalid value for ${additionalEfficiency.label}. Must be between -100 and 100.`)
      return
    }
    payload[additionalEfficiency.name] = effValue

    try {
      const data = await postSimulation(payload, wraparoundEnabled)

      lastSimulationInputsRef.current = payload
      const lapLen = data.points.length > 0 ? data.points[data.points.length - 1].distance : 0
      setResult(lapLen > 0 ? (data.distanceM / lapLen).toFixed(2) : '--')
      setRawDistanceM(data.distanceM)
      setRemainingEnergyWh(data.remainingEnergyWh)
      setOptimalSpeedMps(data.optimalV)
      setTelemetry(data.points)
      setTelemetryAdditionalEfficiency(effValue)
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
    curveSpeedCap: number,
  ) => {
    setTooltip({
      visible: true,
      x: event.clientX + 12,
      y: event.clientY + 12,
      speed,
      accel,
      distance,
      curveSpeedCap,
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

          <button
            type="button"
            className="reset"
            disabled={!selectedPreset}
            onClick={handlePresetReset}
          >
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

          <label>
            {additionalEfficiency.label}
            <input
              type="number"
              step={additionalEfficiency.step}
              min={additionalEfficiency.min}
              max={additionalEfficiency.max}
              name={additionalEfficiency.name}
              value={additionalEfficiency.value}
              onChange={(event) =>
                setAdditionalEfficiency((prev) => ({ ...prev, value: event.target.value }))
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
              Laps: <strong>{result}</strong>
            </div>
            {rawDistanceM !== null ? (
              <div className="result">
                Distance: <strong>{rawDistanceM.toFixed(2)}</strong> m
              </div>
            ) : null}
            {optimalSpeedMps !== null ? (
              <div className="result">
                Optimal speed: <strong>{optimalSpeedMps.toFixed(2)}</strong> m/s
              </div>
            ) : null}
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
                  handleSegmentMove(e, seg.speed, seg.accel, seg.distance, seg.x, seg.y, seg.color, seg.curveSpeedCap)
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
        {rawDistanceM !== null || remainingEnergyWh !== null ? (
          <div className="track-meta">
            {rawDistanceM !== null ? (
              <span>
                Distance: <strong>{rawDistanceM.toFixed(2)}</strong> m
              </span>
            ) : null}
            {rawDistanceM !== null && remainingEnergyWh !== null ? ' · ' : null}
            {remainingEnergyWh !== null ? (
              <span>
                Remaining energy: <strong>{remainingEnergyWh.toFixed(1)}</strong> Wh
              </span>
            ) : null}
          </div>
        ) : null}
        <div style={{ marginTop: 12 }}>
          <button type="button" className="toggle" onClick={() => setGraphsOpen((v) => !v)}>
            {graphsOpen ? 'Hide graphs' : 'Show graphs'}
          </button>
          {graphsOpen ? (
            <div style={{ marginTop: 8 }}>
              <TelemetryGraph
                telemetry={telemetry}
                additionalEfficiency={telemetryAdditionalEfficiency}
              />
            </div>
          ) : null}
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
        {tooltip.curveSpeedCap > 0 ? (
          <div>Curve max: {tooltip.curveSpeedCap.toFixed(2)} m/s</div>
        ) : null}
      </div>
    </div>
  )
}

export default App
