export type JobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'dead_letter'

export interface Job {
  id: string
  name: string
  payload: string
  priority: number
  status: JobStatus
  max_retries: number
  attempts: number
  worker_id?: string
  error?: string
  created_at: string
  updated_at: string
  started_at?: string
  completed_at?: string
  duration_ms?: number
}

export interface JobStats {
  total: number
  pending: number
  running: number
  completed: number
  failed: number
  dead_letter: number
}

export interface WSMessage {
  type: 'job_created' | 'job_updated'
  payload: Job
}

export interface CreateJobRequest {
  name: string
  payload: string
  priority: number
  max_retries: number
}
