import { useState, FormEvent } from 'react'
import { CreateJobRequest } from '../types'

interface Props {
  onSubmit: (req: CreateJobRequest) => Promise<void>
}

export function SubmitJobForm({ onSubmit }: Props) {
  const [name, setName] = useState('')
  const [payload, setPayload] = useState('{}')
  const [priority, setPriority] = useState(5)
  const [maxRetries, setMaxRetries] = useState(3)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    setSuccess(false)

    // Validate JSON payload.
    try {
      JSON.parse(payload)
    } catch {
      setError('Payload must be valid JSON')
      return
    }

    setSubmitting(true)
    try {
      await onSubmit({ name, payload, priority, max_retries: maxRetries })
      setSuccess(true)
      setName('')
      setPayload('{}')
      setTimeout(() => setSuccess(false), 3000)
    } catch (e) {
      setError(String(e))
    } finally {
      setSubmitting(false)
    }
  }

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 12px',
    background: '#0f0f1a',
    border: '1px solid #334155',
    borderRadius: 6,
    color: '#e2e8f0',
    fontSize: 14,
    boxSizing: 'border-box',
  }

  const labelStyle: React.CSSProperties = {
    display: 'block',
    fontSize: 12,
    color: '#94a3b8',
    marginBottom: 4,
  }

  return (
    <div style={{ background: '#1e1e2e', border: '1px solid #334155', borderRadius: 12, padding: 24 }}>
      <h2 style={{ margin: '0 0 20px', fontSize: 16, color: '#e2e8f0' }}>Submit Job</h2>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div>
          <label style={labelStyle}>Job Name *</label>
          <input
            style={inputStyle}
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="e.g. process-sensor-data"
            required
          />
        </div>

        <div>
          <label style={labelStyle}>Payload (JSON)</label>
          <textarea
            style={{ ...inputStyle, height: 80, resize: 'vertical', fontFamily: 'monospace' }}
            value={payload}
            onChange={e => setPayload(e.target.value)}
          />
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <div>
            <label style={labelStyle}>Priority (1–10)</label>
            <input
              style={inputStyle}
              type="number"
              min={1}
              max={10}
              value={priority}
              onChange={e => setPriority(Number(e.target.value))}
            />
          </div>
          <div>
            <label style={labelStyle}>Max Retries</label>
            <input
              style={inputStyle}
              type="number"
              min={0}
              max={10}
              value={maxRetries}
              onChange={e => setMaxRetries(Number(e.target.value))}
            />
          </div>
        </div>

        {error && (
          <div style={{ color: '#ef4444', fontSize: 13, background: '#ef444422', padding: '8px 12px', borderRadius: 6 }}>
            {error}
          </div>
        )}
        {success && (
          <div style={{ color: '#10b981', fontSize: 13, background: '#10b98122', padding: '8px 12px', borderRadius: 6 }}>
            Job submitted successfully!
          </div>
        )}

        <button
          type="submit"
          disabled={submitting}
          style={{
            padding: '10px 20px',
            background: submitting ? '#334155' : '#6366f1',
            color: '#fff',
            border: 'none',
            borderRadius: 8,
            cursor: submitting ? 'not-allowed' : 'pointer',
            fontSize: 14,
            fontWeight: 600,
            transition: 'background 0.2s',
          }}
        >
          {submitting ? 'Submitting…' : 'Submit Job'}
        </button>
      </form>
    </div>
  )
}
