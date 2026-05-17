export const SENSITIVE_RULE_ACTION = {
  BLOCK: 1,
  MONITOR: 2,
} as const

export type SensitiveRuleAction =
  (typeof SENSITIVE_RULE_ACTION)[keyof typeof SENSITIVE_RULE_ACTION]

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

export type SensitiveRule = {
  id: number
  pattern: string
  is_regex: boolean
  enabled: boolean
  action: SensitiveRuleAction
  category: string
  severity: number
  description: string
  hit_count: number
  last_hit_at?: string | null
  created_at: string
  updated_at: string
}

export type SensitiveHit = {
  id: number
  rule_id: number
  pattern: string
  is_regex: boolean
  action: SensitiveRuleAction
  user_id: number
  username: string
  token_id: number
  token_name: string
  model_name: string
  request_id: string
  ip: string
  channel_id: number
  group: string
  path: string
  prompt_snippet: string
  false_positive: boolean
  reviewed_by: number
  reviewed_at?: string | null
  created_at: string
}

export type SensitiveHitFilters = {
  action?: number
  keyword?: string
  username?: string
  token_name?: string
  model_name?: string
  request_id?: string
  path?: string
  false_positive?: 'true' | 'false' | ''
  start_time?: string
  end_time?: string
}

export type SensitiveRuleFormData = {
  pattern: string
  is_regex: boolean
  enabled: boolean
  action: SensitiveRuleAction
  category: string
  severity: number
  description: string
}

export type SeedDefaultSensitiveRulesResult = {
  created: number
  updated: number
  skipped: number
}
