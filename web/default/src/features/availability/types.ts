export type AvailabilityRange = '30m' | '1d' | '7d'

export interface GroupAvailabilitySnapshot {
  group: string
  success_rate: number | null
  online_rate: number
  avg_latency_ms: number
  request_count: number
  error_count: number
  channel_total: number
  channel_online: number
}

export interface GroupAvailabilityOverview {
  groups: GroupAvailabilitySnapshot[]
  window_seconds: number
  generated_at: number
}

export interface AvailabilityBucket {
  bucket: number
  success_count: number
  error_count: number
  avg_latency_ms: number
}

export interface ChannelAvailabilityRow {
  channel_id: number
  name: string
  status: number
  response_time_ms: number
  test_time: number
  request_count: number
  error_count: number
}

export interface GroupAvailabilityTimeseries {
  group: string
  range: AvailabilityRange
  bucket_sec: number
  start_sec: number
  end_sec: number
  points: AvailabilityBucket[]
  p50_ms: number | null
  p95_ms: number | null
  success_rate: number | null
  request_count: number
  error_count: number
  avg_latency_ms: number
  channel_total: number
  channel_online: number
  channels: ChannelAvailabilityRow[]
}

export interface ApiEnvelope<T> {
  success: boolean
  message?: string
  data?: T
}
