import { api } from '@/lib/api'
import type {
  ApiResponse,
  Paged,
  SalesDashboard,
  SalesCustomer,
  ConsumesPage,
  CommissionBill,
  WithdrawRequest,
} from './types'

const BASE = '/api/sales'

export async function getSalesDashboard(month?: string) {
  const res = await api.get<ApiResponse<SalesDashboard>>(`${BASE}/dashboard`, {
    params: month ? { month } : undefined,
  })
  return res.data
}

export async function listSalesCustomers(params: { page?: number; page_size?: number }) {
  const res = await api.get<ApiResponse<Paged<SalesCustomer>>>(`${BASE}/customers`, { params })
  return res.data
}

export async function getSalesConsumes(month?: string) {
  const res = await api.get<ApiResponse<ConsumesPage>>(`${BASE}/consumes`, {
    params: month ? { month } : undefined,
  })
  return res.data
}

export async function listSalesBills(params: { page?: number; page_size?: number }) {
  const res = await api.get<ApiResponse<Paged<CommissionBill>>>(`${BASE}/bills`, { params })
  return res.data
}

export async function applyWithdraw(payload: { amount_quota: number; applicant_note?: string }) {
  const res = await api.post<ApiResponse<WithdrawRequest>>(`${BASE}/withdraw`, payload)
  return res.data
}

export async function listSalesWithdraws(params: { page?: number; page_size?: number }) {
  const res = await api.get<ApiResponse<Paged<WithdrawRequest>>>(`${BASE}/withdraws`, { params })
  return res.data
}
