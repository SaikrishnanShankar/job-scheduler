import { Job, JobStatus } from '../types'

interface Props {
  jobs: Job[]
}

const STATUS_COLORS: Record<JobStatus, { bg: string; fg: string }> = {
  pending:     { bg: '#f59e0b22', fg: '#f59e0b' },
  running:     { bg: '#3b82f622', fg: '#3b82f6' },
  completed:   { bg: '#10b98122', fg: '#10b981' },
  failed:      { bg: '#ef444422', fg: '#ef4444' },
  dead_letter: { bg: '#8b5cf622', fg: '#8b5cf6' },
}

function StatusBadge({ status }: { status: JobStatus }) {
  const { bg, fg } = STATUS_COLORS[status]
  return (
    <span style={{
      background: bg, color: fg,
      padding: '2px 10px', borderRadius: 99,
      fontSize: 12, fontWeight: 600,
      textTransform: 'uppercase', letterSpacing: '0.05em',
    }}>
      {status.replace('_', ' ')}
    </span>
  )
}

function PriorityDot({ priority }: { priority: number }) {
  const hue = Math.round(120 - (priority - 1) * 12) // green→red
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span style={{
        width: 10, height: 10, borderRadius: '50%',
        background: `hsl(${hue},80%,55%)`, flexShrink: 0,
      }} />
      {priority}
    </span>
  )
}

function formatDuration(ms?: number): string {
  if (ms == null) return '—'
  if (ms < 1000) return `${ms.toFixed(0)}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  return `${Math.floor(s / 3600)}h ago`
}

export function JobTable({ jobs }: Props) {
  const thStyle: React.CSSProperties = {
    padding: '10px 16px',
    textAlign: 'left',
    fontSize: 11,
    fontWeight: 600,
    color: '#64748b',
    textTransform: 'uppercase',
    letterSpacing: '0.08em',
    borderBottom: '1px solid #1e293b',
    whiteSpace: 'nowrap',
  }
  const tdStyle: React.CSSProperties = {
    padding: '12px 16px',
    fontSize: 13,
    color: '#cbd5e1',
    borderBottom: '1px solid #1e293b',
    verticalAlign: 'middle',
  }

  return (
    <div style={{ background: '#1e1e2e', border: '1px solid #334155', borderRadius: 12, overflow: 'hidden' }}>
      <div style={{ padding: '16px 20px', borderBottom: '1px solid #334155', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0, fontSize: 16, color: '#e2e8f0' }}>Jobs</h2>
        <span style={{ fontSize: 12, color: '#64748b' }}>{jobs.length} jobs</span>
      </div>

      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={thStyle}>Name</th>
              <th style={thStyle}>Status</th>
              <th style={thStyle}>Priority</th>
              <th style={thStyle}>Attempts</th>
              <th style={thStyle}>Duration</th>
              <th style={thStyle}>Worker</th>
              <th style={thStyle}>Created</th>
            </tr>
          </thead>
          <tbody>
            {jobs.length === 0 ? (
              <tr>
                <td colSpan={7} style={{ ...tdStyle, textAlign: 'center', color: '#475569', padding: '40px' }}>
                  No jobs yet — submit one above
                </td>
              </tr>
            ) : (
              jobs.map(job => (
                <tr key={job.id} style={{ transition: 'background 0.15s' }}
                  onMouseEnter={e => (e.currentTarget.style.background = '#0f0f1a')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <td style={tdStyle}>
                    <div style={{ fontWeight: 500, color: '#e2e8f0' }}>{job.name}</div>
                    <div style={{ fontSize: 11, color: '#475569', fontFamily: 'monospace' }}>
                      {job.id.slice(0, 8)}…
                    </div>
                  </td>
                  <td style={tdStyle}><StatusBadge status={job.status} /></td>
                  <td style={tdStyle}><PriorityDot priority={job.priority} /></td>
                  <td style={tdStyle}>{job.attempts}/{job.max_retries}</td>
                  <td style={tdStyle}>{formatDuration(job.duration_ms)}</td>
                  <td style={tdStyle}>
                    <span style={{ fontFamily: 'monospace', fontSize: 11 }}>
                      {job.worker_id?.slice(0, 16) ?? '—'}
                    </span>
                  </td>
                  <td style={tdStyle}>{timeAgo(job.created_at)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
