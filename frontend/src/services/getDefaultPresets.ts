export type SimulationInputs = {
  v: number
  batteryWh: number
  solarWhPerMin: number
  etaDrive: number
  raceDayMin: number
  rWheel: number
  tMax: number
  pMax: number
  m: number
  g: number
  cRr: number
  rho: number
  cD: number
  a: number
  theta: number
  gmax: number
  additionalEfficiency: number
}

export type SimulationPreset = {
  id: string
  label: string
  inputs: SimulationInputs
}

export type DefaultPresetsResponse = {
  defaultPresetId: string
  presets: SimulationPreset[]
}

export async function getDefaultPresets(): Promise<DefaultPresetsResponse> {
  try {
    const response = await fetch('http://localhost:8080/defaults')

    if (!response.ok) {
      throw new Error(`Failed to fetch default presets: ${response.status}`)
    }

    return response.json()
  } catch (error) {
    console.error(error)
    throw new Error('Failed to fetch default presets')
  }
}
