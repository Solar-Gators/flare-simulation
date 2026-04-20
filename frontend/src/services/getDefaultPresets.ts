export type SimulationInputs = {
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

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080').replace(
  /\/+$/,
  '',
)

function apiUrl(path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${API_BASE_URL}${normalizedPath}`
}

export async function getDefaultPresets(): Promise<DefaultPresetsResponse> {
  try {
    const response = await fetch(apiUrl('/defaults'))

    if (!response.ok) {
      throw new Error(`Failed to fetch default presets: ${response.status}`)
    }

    return response.json()
  } catch (error) {
    console.error(error)
    throw new Error('Failed to fetch default presets')
  }
}
