import { useJobs } from './hooks/useJobs'
import { StatsBar } from './components/StatsBar'
import { SubmitJobForm } from './components/SubmitJobForm'
import { JobTable } from './components/JobTable'
import { StatusChart } from './components/StatusChart'
import { CreateJobRequest } from './types'

export default function App() {
  const { jobs, stats, loading, error, submitJob, refresh } = useJobs()

  const handleSubmit = async (req: CreateJobRequest) => {
    await submitJob(req)
  }

  return (
    <div style={{
      minHeight: '100vh',
      background: '#0f0f1a',
      color: '#e2e8f0',
      fontFamily: "'Inter', 'Segoe UI', system-ui, sans-serif",
      padding: '24px 32px',
    }}>
      {/* Header */}
      <header style={{ marginBottom: 28, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700, color: '#f1f5f9' }}>
            <span style={{ color: '#6366f1' }}>⬡</span> Job Scheduler
          </h1>
          <p style={{ margin: '4px 0 0', fontSize: 13, color: '#64748b' }}>
            Distributed real-time job processing dashboard
          </p>
        </div>
        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#10b981', boxShadow: '0 0 6px #10b981' }} />
          <span style={{ fontSize: 12, color: '#64748b' }}>Live</span>
          <button
            onClick={refresh}
            style={{
              padding: '6px 14px', background: '#1e293b', border: '1px solid #334155',
              borderRadius: 8, color: '#94a3b8', cursor: 'pointer', fontSize: 13,
            }}
          >
            Refresh
          </button>
        </div>
      </header>

      {error && (
        <div style={{ marginBottom: 16, padding: '12px 16px', background: '#ef444422', border: '1px solid #ef444466', borderRadius: 8, color: '#ef4444', fontSize: 13 }}>
          {error}
        </div>
      )}

      {/* Stats */}
      <section style={{ marginBottom: 24 }}>
        <StatsBar stats={stats} />
      </section>

      {/* Submit + Chart row */}
      <section style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, marginBottom: 24 }}>
        <SubmitJobForm onSubmit={handleSubmit} />
        <StatusChart stats={stats} />
      </section>

      {/* Job table */}
      <section>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 60, color: '#475569' }}>Loading jobs…</div>
        ) : (
          <JobTable jobs={jobs} />
        )}
      </section>
    </div>
  )
}
