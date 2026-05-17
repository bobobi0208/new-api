import { api } from '@/lib/api'
import type {
  ApiResponse,
  PageData,
  ReconciliationChannelConfig,
  ReconciliationChannelConfigFormData,
  ReconciliationRecord,
  ReconciliationRecordFilters,
  ReconciliationSetting,
} from './types'

const BASE_URL = '/api/reconciliation'

export async function listReconciliationRecords(
  params: { p?: number; page_size?: number } & ReconciliationRecordFilters
): Promise<ApiResponse<PageData<ReconciliationRecord>>> {
  const res = await api.get(`${BASE_URL}/records`, { params })
  return res.data
}

export async function getReconciliationRecord(
  id: number
): Promise<ApiResponse<ReconciliationRecord>> {
  const res = await api.get(`${BASE_URL}/records/${id}`)
  return res.data
}

export async function listReconciliationLatest(
  upstreamType?: string
): Promise<ApiResponse<ReconciliationRecord[]>> {
  const params = upstreamType ? { upstream_type: upstreamType } : undefined
  const res = await api.get(`${BASE_URL}/latest`, { params })
  return res.data
}

export async function listReconciliationAlerts(): Promise<
  ApiResponse<ReconciliationRecord[]>
> {
  const res = await api.get(`${BASE_URL}/alerts`)
  return res.data
}

export async function triggerReconciliation(payload: {
  channel_id: number
  run_type?: string
  window_start?: number
  window_end?: number
}): Promise<ApiResponse<ReconciliationRecord>> {
  const res = await api.post(`${BASE_URL}/trigger`, payload)
  return res.data
}

export async function getReconciliationSettings(): Promise<
  ApiResponse<ReconciliationSetting>
> {
  const res = await api.get(`${BASE_URL}/settings`)
  return res.data
}

export async function updateReconciliationSettings(
  data: ReconciliationSetting
): Promise<ApiResponse<ReconciliationSetting>> {
  const res = await api.put(`${BASE_URL}/settings`, data)
  return res.data
}

export async function listReconciliationChannelConfigs(): Promise<
  ApiResponse<ReconciliationChannelConfig[]>
> {
  const res = await api.get(`${BASE_URL}/channels`)
  return res.data
}

export async function upsertReconciliationChannelConfig(
  data: ReconciliationChannelConfigFormData
): Promise<ApiResponse<ReconciliationChannelConfig>> {
  const res = await api.post(`${BASE_URL}/channels`, data)
  return res.data
}

export async function deleteReconciliationChannelConfig(
  channelId: number
): Promise<ApiResponse<unknown>> {
  const res = await api.delete(`${BASE_URL}/channels/${channelId}`)
  return res.data
}
