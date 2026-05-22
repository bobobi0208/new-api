import { api } from '@/lib/api'
import type {
  ApiResponse,
  CommissionBill,
  Paged,
  SalesUserAdmin,
  Tier,
  WithdrawRequest,
} from '../sales/types'

const BASE = '/api/admin/sales'

export async function listAllSales(params: { page?: number; page_size?: number }) {
  const res = await api.get<ApiResponse<Paged<SalesUserAdmin>>>(`${BASE}/users`, { params })
  return res.data
}

export async function promoteUserToSales(userId: number) {
  const res = await api.post<ApiResponse<unknown>>(`${BASE}/users/${userId}/promote`)
  return res.data
}

export async function demoteSalesToUser(userId: number) {
  const res = await api.post<ApiResponse<unknown>>(`${BASE}/users/${userId}/demote`)
  return res.data
}

export async function bindCustomerToSales(userId: number, inviterId: number) {
  const res = await api.patch<ApiResponse<unknown>>(`${BASE}/users/${userId}/bind`, {
    inviter_id: inviterId,
  })
  return res.data
}

export async function updateSalesTiers(
  userId: number,
  payload: { commission_rate: number; commission_tier_config: Tier[] },
) {
  const res = await api.patch<ApiResponse<unknown>>(`${BASE}/users/${userId}/tiers`, payload)
  return res.data
}

export async function getDefaultTiers() {
  const res = await api.get<ApiResponse<Tier[]>>(`${BASE}/commission/default-tiers`)
  return res.data
}

export async function setDefaultTiers(tiers: Tier[]) {
  const res = await api.put<ApiResponse<unknown>>(`${BASE}/commission/default-tiers`, { tiers })
  return res.data
}

export async function generateMonthlyBills(month: string) {
  const res = await api.post<ApiResponse<{ created: number; skipped: number }>>(
    `${BASE}/commission/bills/generate`,
    { month },
  )
  return res.data
}

export async function listAdminBills(params: {
  month?: string
  status?: number
  page?: number
  page_size?: number
}) {
  const res = await api.get<ApiResponse<Paged<CommissionBill>>>(`${BASE}/commission/bills`, {
    params,
  })
  return res.data
}

export async function confirmBill(id: number) {
  const res = await api.post<ApiResponse<unknown>>(`${BASE}/commission/bills/${id}/confirm`)
  return res.data
}

export async function listAdminWithdraws(params: { status?: number; page?: number; page_size?: number }) {
  const res = await api.get<ApiResponse<Paged<WithdrawRequest>>>(`${BASE}/withdraws`, { params })
  return res.data
}

export async function approveWithdraw(id: number) {
  const res = await api.post<ApiResponse<unknown>>(`${BASE}/withdraws/${id}/approve`)
  return res.data
}

export async function rejectWithdraw(id: number, rejectReason: string) {
  const res = await api.post<ApiResponse<unknown>>(`${BASE}/withdraws/${id}/reject`, {
    reject_reason: rejectReason,
  })
  return res.data
}

export async function markWithdrawPaid(id: number, note: string) {
  const res = await api.post<ApiResponse<unknown>>(`${BASE}/withdraws/${id}/paid`, { note })
  return res.data
}
