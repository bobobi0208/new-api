export type ApiResponse<T = unknown> = {
  success: boolean
  message?: string
  data?: T
}

export type PageData<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export const RECONCILIATION_STATUS = {
  MATCH: 'match',
  MISMATCH: 'mismatch',
  INCONCLUSIVE: 'inconclusive',
  ERROR: 'error',
} as const

export const RECONCILIATION_RUN_TYPE = {
  BALANCE: 'balance_snapshot',
  DAILY: 'daily_diff',
  MANUAL: 'manual',
} as const

export const RECONCILIATION_UPSTREAM_TYPE = {
  NEWAPI: 'newapi',
  SUB2API: 'sub2api',
} as const

export type ReconciliationStatus =
  (typeof RECONCILIATION_STATUS)[keyof typeof RECONCILIATION_STATUS]

export type ReconciliationRunType =
  (typeof RECONCILIATION_RUN_TYPE)[keyof typeof RECONCILIATION_RUN_TYPE]

export type ReconciliationUpstreamType =
  (typeof RECONCILIATION_UPSTREAM_TYPE)[keyof typeof RECONCILIATION_UPSTREAM_TYPE]

export type ReconciliationRecord = {
  id: number
  channel_id: number
  channel_name?: string
  upstream_type: ReconciliationUpstreamType | string
  run_type: ReconciliationRunType | string
  run_at: number
  window_start: number
  window_end: number
  upstream_used_usd: number
  upstream_remain_usd: number
  upstream_raw_json: string
  local_used_usd: number
  delta_usd: number
  delta_rel: number
  status: ReconciliationStatus | string
  message: string
  created_at: string
  updated_at: string
}

export type ReconciliationChannelConfig = {
  id: number
  channel_id: number
  upstream_type: string
  enabled: boolean
  base_url: string
  note: string
  created_at: string
  updated_at: string
}

export type ReconciliationSetting = {
  enabled: boolean
  balance_poll_interval_sec: number
  daily_job_hour: number
  abs_threshold_usd: number
  rel_threshold: number
  http_timeout_sec: number
  retain_days: number
}

export type ReconciliationRecordFilters = {
  channel_id?: number
  upstream_type?: string
  run_type?: string
  status?: string
  start_at?: number
  end_at?: number
}

export type ReconciliationChannelConfigFormData = {
  channel_id: number
  upstream_type: string
  enabled: boolean
  base_url: string
  note: string
}
