import { JobStats } from '../types'

interface Props {
  stats: JobStats | null
}

const cards = [
  { key: 'total',       label: 'Total',       color: '#6366f1' },
  { key: 'pending',     label: 'Pending',     color: '#f59e0b' },
  { key: 'running',     label: 'Running',     color: '#3b82f6' },
  { key: 'completed',   label: 'Completed',   color: '#10b981' },
  { key: 'failed',      label: 'Failed',      color: '#ef4444' },
  { key: 'dead_letter', label: 'Dead Letter', color: '#8b5cf6' },
] as const

export function StatsBar({ stats }: Props) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 12 }}>
      {cards.map(({ key, label, color }) => (
        <div
          key={key}
          style={{
            background: '#1e1e2e',
            border: `1px solid ${color}44`,
            borderRadius: 10,
            padding: '16px 12px',
            textAlign: 'center',
          }}
        >
          <div style={{ fontSize: 28, fontWeight: 700, color }}>{stats ? stats[key] : '—'}</div>
          <div style={{ fontSize: 12, color: '#94a3b8', marginTop: 4 }}>{label}</div>
        </div>
      ))}
    </div>
  )
}
