export type Tier = {
  from: number
  rate_bp: number
}

export type SalesDashboard = {
  month: string
  customer_count: number
  month_consume_quota: number
  month_commission_quota: number
  commission_balance: number
  commission_history_total: number
  tiers: Tier[]
  aff_code?: string
  aff_count?: number
}

export type SalesCustomer = {
  id: number
  username: string
  display_name?: string
  email?: string
  status?: number
  created_at?: number
  used_quota?: number
  quota?: number
  request_count?: number
}

export type ConsumeRow = {
  customer_id: number
  customer_name: string
  consume_quota: number
  commission_quota: number
}

export type ConsumesPage = {
  month: string
  items: ConsumeRow[]
  total_consume: number
  total_commission: number
  tiers: Tier[]
}

export type CommissionBill = {
  id: number
  sales_user_id: number
  year_month: string
  total_consume_quota: number
  commission_quota: number
  status: number
  created_at: number
  confirmed_at: number
  confirmed_by: number
  detail?: string
}

export type WithdrawRequest = {
  id: number
  sales_user_id: number
  amount_quota: number
  status: number
  applied_at: number
  reviewed_at?: number
  reviewed_by?: number
  paid_at?: number
  paid_note?: string
  reject_reason?: string
  applicant_note?: string
}

export type SalesUserAdmin = {
  id: number
  username: string
  display_name?: string
  email?: string
  commission_rate: number
  commission_tier_config?: string
  commission_balance: number
  commission_history_total: number
}

export const BILL_STATUS = {
  DRAFT: 0,
  CONFIRMED: 1,
} as const

export const WITHDRAW_STATUS = {
  PENDING: 0,
  APPROVED: 1,
  REJECTED: 2,
  PAID: 3,
} as const

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export type Paged<T> = {
  items: T[]
  total: number
  page: number
  page_size: number
}

export const QUOTA_PER_USD = 500_000

export function formatUSD(quota: number | undefined | null): string {
  if (quota === undefined || quota === null || Number.isNaN(quota)) return '-'
  return `$${(quota / QUOTA_PER_USD).toFixed(2)}`
}

export function formatBP(bp: number | undefined | null): string {
  if (bp === undefined || bp === null) return '-'
  return `${(bp / 100).toFixed(2)}%`
}

export function formatTs(ts?: number | null) {
  if (!ts || ts <= 0) return '-'
  return new Date(ts * 1000).toLocaleString()
}
