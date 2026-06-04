import { useQuery } from '@tanstack/react-query'
import { fetchAvailabilityGroup, fetchAvailabilityOverview } from '../api'
import type { AvailabilityRange } from '../types'

const REFRESH_INTERVALS: Record<AvailabilityRange, number> = {
  '30m': 10_000,
  '1d': 60_000,
  '7d': 300_000,
}

export function useAvailabilityOverview(range: AvailabilityRange) {
  return useQuery({
    queryKey: ['availability', 'overview'] as const,
    queryFn: fetchAvailabilityOverview,
    refetchInterval: REFRESH_INTERVALS[range],
    refetchIntervalInBackground: false,
    staleTime: 5_000,
  })
}

export function useAvailabilityGroup(
  group: string | null,
  range: AvailabilityRange
) {
  return useQuery({
    queryKey: ['availability', 'group', group, range] as const,
    queryFn: () => fetchAvailabilityGroup(group as string, range),
    enabled: !!group,
    refetchInterval: REFRESH_INTERVALS[range],
    refetchIntervalInBackground: false,
    staleTime: 5_000,
  })
}
