import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { JobStats } from '../types'

interface Props {
  stats: JobStats | null
}

const SLICES = [
  { key: 'pending',     label: 'Pending',     color: '#f59e0b' },
  { key: 'running',     label: 'Running',     color: '#3b82f6' },
  { key: 'completed',   label: 'Completed',   color: '#10b981' },
  { key: 'failed',      label: 'Failed',      color: '#ef4444' },
  { key: 'dead_letter', label: 'Dead Letter', color: '#8b5cf6' },
] as const

export function StatusChart({ stats }: Props) {
  const data = SLICES
    .map(s => ({ name: s.label, value: stats ? stats[s.key] : 0, color: s.color }))
    .filter(d => d.value > 0)

  return (
    <div style={{ background: '#1e1e2e', border: '1px solid #334155', borderRadius: 12, padding: 24 }}>
      <h2 style={{ margin: '0 0 16px', fontSize: 16, color: '#e2e8f0' }}>Status Distribution</h2>
      {data.length === 0 ? (
        <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#475569' }}>
          No data yet
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={200}>
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              innerRadius={55}
              outerRadius={80}
              paddingAngle={3}
              dataKey="value"
            >
              {data.map((entry, i) => (
                <Cell key={i} fill={entry.color} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{ background: '#0f0f1a', border: '1px solid #334155', borderRadius: 8 }}
              labelStyle={{ color: '#e2e8f0' }}
              itemStyle={{ color: '#94a3b8' }}
            />
            <Legend
              iconType="circle"
              iconSize={8}
              wrapperStyle={{ fontSize: 12, color: '#94a3b8' }}
            />
          </PieChart>
        </ResponsiveContainer>
      )}
    </div>
  )
}
