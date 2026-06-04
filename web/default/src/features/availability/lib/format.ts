import type { AvailabilityRange } from '../types'

export function formatPercent(v: number | null | undefined, digits = 1): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '—'
  return `${(v * 100).toFixed(digits)}%`
}

export function formatLatency(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v) || v <= 0) return '—'
  if (v < 1000) return `${Math.round(v)} ms`
  return `${(v / 1000).toFixed(2)} s`
}

export function formatCount(v: number | null | undefined): string {
  if (v === null || v === undefined || Number.isNaN(v)) return '0'
  if (v < 1000) return String(v)
  if (v < 1_000_000) return `${(v / 1000).toFixed(1)}k`
  return `${(v / 1_000_000).toFixed(2)}m`
}

export function rangeWindowSec(range: AvailabilityRange): number {
  switch (range) {
    case '1d':
      return 24 * 3600
    case '7d':
      return 7 * 24 * 3600
    default:
      return 30 * 60
  }
}

export function successRateTone(
  v: number | null | undefined
): 'success' | 'warning' | 'destructive' | 'muted' {
  if (v === null || v === undefined) return 'muted'
  if (v >= 0.99) return 'success'
  if (v >= 0.95) return 'warning'
  return 'destructive'
}

export function onlineRateTone(
  online: number,
  total: number
): 'success' | 'warning' | 'destructive' | 'muted' {
  if (total <= 0) return 'muted'
  if (online === total) return 'success'
  if (online * 2 >= total) return 'warning'
  return 'destructive'
}
