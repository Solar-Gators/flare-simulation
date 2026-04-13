import type { FormEvent } from 'react'
import type { FieldDef } from './types'

type Props = {
  fields: FieldDef[]
  result: string
  status: string
  onFieldChange: (name: string, value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

export default function DistanceForm({ fields, result, status, onFieldChange, onSubmit }: Props) {
  return (
    <section className="panel">
      <form id="distance-form" onSubmit={onSubmit}>
        <div className="grid">
          {fields.map((field) => (
            <label key={field.name}>
              {field.label}
              <input
                type="number"
                step={field.step}
                name={field.name}
                value={field.value}
                min={field.min}
                max={field.max}
                onChange={(e) => onFieldChange(field.name, e.target.value)}
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
  )
}
