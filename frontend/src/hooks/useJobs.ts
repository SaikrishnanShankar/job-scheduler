import { useState, useEffect, useCallback } from 'react'
import { Job, JobStats, CreateJobRequest, WSMessage } from '../types'
import { useWebSocket } from './useWebSocket'

const API = '/api/v1'

export function useJobs() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [stats, setStats] = useState<JobStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchJobs = useCallback(async () => {
    try {
      const res = await fetch(`${API}/jobs?limit=100`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: Job[] = await res.json()
      setJobs(data)
    } catch (e) {
      setError(String(e))
    }
  }, [])

  const fetchStats = useCallback(async () => {
    try {
      const res = await fetch(`${API}/stats`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: JobStats = await res.json()
      setStats(data)
    } catch (e) {
      setError(String(e))
    }
  }, [])

  const loadAll = useCallback(async () => {
    setLoading(true)
    await Promise.all([fetchJobs(), fetchStats()])
    setLoading(false)
  }, [fetchJobs, fetchStats])

  useEffect(() => { loadAll() }, [loadAll])

  // Handle live WebSocket updates.
  const handleWS = useCallback((msg: WSMessage) => {
    const updated = msg.payload
    setJobs(prev => {
      const idx = prev.findIndex(j => j.id === updated.id)
      if (idx === -1) return [updated, ...prev]
      const next = [...prev]
      next[idx] = updated
      return next
    })
    // Refresh stats after any update.
    fetchStats()
  }, [fetchStats])

  useWebSocket(handleWS)

  const submitJob = useCallback(async (req: CreateJobRequest): Promise<Job> => {
    const res = await fetch(`${API}/jobs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error ?? `HTTP ${res.status}`)
    }
    const job: Job = await res.json()
    fetchStats()
    return job
  }, [fetchStats])

  return { jobs, stats, loading, error, submitJob, refresh: loadAll }
}
