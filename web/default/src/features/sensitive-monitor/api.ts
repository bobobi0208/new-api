import { api } from '@/lib/api'
import type {
  ApiResponse,
  PageData,
  SeedDefaultSensitiveRulesResult,
  SensitiveHit,
  SensitiveHitFilters,
  SensitiveRule,
  SensitiveRuleFormData,
} from './types'

const BASE_URL = '/api/sensitive_monitor'

export async function getSensitiveRules(params: {
  p?: number
  page_size?: number
  enabled?: boolean
}): Promise<ApiResponse<PageData<SensitiveRule>>> {
  const res = await api.get(`${BASE_URL}/rules`, { params })
  return res.data
}

export async function getSensitiveHits(params: {
  p?: number
  page_size?: number
} & SensitiveHitFilters): Promise<ApiResponse<PageData<SensitiveHit>>> {
  const res = await api.get(`${BASE_URL}/hits`, { params })
  return res.data
}

export async function exportSensitiveHits(
  params: SensitiveHitFilters
): Promise<Blob> {
  const res = await api.get(`${BASE_URL}/hits/export`, {
    params,
    responseType: 'blob',
    disableDuplicate: true,
  } as Record<string, unknown>)
  return res.data as Blob
}

export async function createSensitiveRule(
  data: SensitiveRuleFormData
): Promise<ApiResponse<SensitiveRule>> {
  const res = await api.post(`${BASE_URL}/rules`, data)
  return res.data
}

export async function updateSensitiveRule(
  id: number,
  data: SensitiveRuleFormData
): Promise<ApiResponse<null>> {
  const res = await api.put(`${BASE_URL}/rules/${id}`, data)
  return res.data
}

export async function reloadSensitiveRules(): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE_URL}/reload`)
  return res.data
}

export async function seedDefaultSensitiveRules(): Promise<
  ApiResponse<SeedDefaultSensitiveRulesResult>
> {
  const res = await api.post(`${BASE_URL}/seed_defaults`)
  return res.data
}

export async function markSensitiveHitFalsePositive(
  hitId: number,
  value: boolean
): Promise<ApiResponse<null>> {
  const res = await api.post(`${BASE_URL}/hits/${hitId}/false_positive`, {
    value,
  })
  return res.data
}
