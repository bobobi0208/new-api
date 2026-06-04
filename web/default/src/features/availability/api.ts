import { api } from '@/lib/api'
import type {
  ApiEnvelope,
  AvailabilityRange,
  GroupAvailabilityOverview,
  GroupAvailabilityTimeseries,
} from './types'

export async function fetchAvailabilityOverview(): Promise<GroupAvailabilityOverview> {
  const res = await api.get<ApiEnvelope<GroupAvailabilityOverview>>(
    '/api/availability/overview'
  )
  if (!res.data.success || !res.data.data) {
    throw new Error(res.data.message || 'failed to fetch availability overview')
  }
  return res.data.data
}

export async function fetchAvailabilityGroup(
  group: string,
  range: AvailabilityRange
): Promise<GroupAvailabilityTimeseries> {
  const res = await api.get<ApiEnvelope<GroupAvailabilityTimeseries>>(
    '/api/availability/group',
    { params: { group, range } }
  )
  if (!res.data.success || !res.data.data) {
    throw new Error(res.data.message || 'failed to fetch group availability')
  }
  return res.data.data
}
