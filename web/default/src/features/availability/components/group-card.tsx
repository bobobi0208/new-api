import { useTranslation } from 'react-i18next'
import { ArrowRight, CircleDot, Gauge, Timer, Zap } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GroupAvailabilitySnapshot } from '../types'
import {
  formatCount,
  formatLatency,
  formatPercent,
  onlineRateTone,
  successRateTone,
} from '../lib/format'

interface GroupCardProps {
  snapshot: GroupAvailabilitySnapshot
  onClick: () => void
}

const TONE_RING: Record<string, string> = {
  success: 'ring-success/40 bg-success/5',
  warning: 'ring-warning/40 bg-warning/5',
  destructive: 'ring-destructive/40 bg-destructive/5',
  muted: 'ring-border bg-muted/10',
}

const TONE_DOT: Record<string, string> = {
  success: 'bg-success',
  warning: 'bg-warning',
  destructive: 'bg-destructive',
  muted: 'bg-muted-foreground/40',
}

const TONE_TEXT: Record<string, string> = {
  success: 'text-success',
  warning: 'text-warning',
  destructive: 'text-destructive',
  muted: 'text-muted-foreground',
}

export function GroupCard(props: GroupCardProps) {
  const { t } = useTranslation()
  const s = props.snapshot
  const sTone = successRateTone(s.success_rate)
  const oTone = onlineRateTone(s.channel_online, s.channel_total)
  const overallTone = sTone === 'muted' ? oTone : sTone

  return (
    <button
      type='button'
      onClick={props.onClick}
      className={cn(
        'group relative flex w-full flex-col gap-4 rounded-2xl border p-5 text-left transition-all ring-1',
        'hover:border-foreground/20 hover:shadow-md',
        TONE_RING[overallTone]
      )}
    >
      <div className='flex items-start justify-between gap-2'>
        <div className='flex items-center gap-2.5 min-w-0'>
          <span
            aria-hidden='true'
            className={cn(
              'inline-flex size-2.5 shrink-0 rounded-full',
              TONE_DOT[overallTone]
            )}
          />
          <h3 className='text-foreground truncate text-base font-semibold'>
            {s.group}
          </h3>
        </div>
        <ArrowRight className='text-muted-foreground/60 group-hover:text-foreground size-4 shrink-0 transition-colors' />
      </div>

      <div className='grid grid-cols-2 gap-3'>
        <Stat
          icon={Gauge}
          label={t('Success Rate')}
          value={formatPercent(s.success_rate)}
          tone={sTone}
        />
        <Stat
          icon={CircleDot}
          label={t('Online Rate')}
          value={
            s.channel_total > 0
              ? `${s.channel_online}/${s.channel_total}`
              : '—'
          }
          tone={oTone}
        />
        <Stat
          icon={Zap}
          label={t('Request Count')}
          value={formatCount(s.request_count)}
        />
        <Stat
          icon={Timer}
          label={t('Avg Latency')}
          value={formatLatency(s.avg_latency_ms)}
        />
      </div>
    </button>
  )
}

interface StatProps {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  tone?: 'success' | 'warning' | 'destructive' | 'muted'
}

function Stat(p: StatProps) {
  const Icon = p.icon
  const cls = p.tone ? TONE_TEXT[p.tone] : 'text-foreground'
  return (
    <div className='bg-background/60 border-border/60 flex flex-col gap-1 rounded-lg border px-3 py-2'>
      <div className='text-muted-foreground flex items-center gap-1.5 text-[11px] font-medium'>
        <Icon className='size-3.5' />
        {p.label}
      </div>
      <div className={cn('font-mono text-base font-semibold tabular-nums', cls)}>
        {p.value}
      </div>
    </div>
  )
}
