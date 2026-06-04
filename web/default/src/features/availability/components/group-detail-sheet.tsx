import { useTranslation } from 'react-i18next'
import { CircleDot, Gauge, Timer, Zap } from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { useAvailabilityGroup } from '../hooks/use-availability'
import type { AvailabilityRange } from '../types'
import {
  formatCount,
  formatLatency,
  formatPercent,
  successRateTone,
} from '../lib/format'
import { RangeTabs } from './range-tabs'
import { TimeseriesChart } from './timeseries-chart'
import { ChannelTable } from './channel-table'

interface GroupDetailSheetProps {
  group: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
  range: AvailabilityRange
  onRangeChange: (range: AvailabilityRange) => void
}

const TONE_TEXT: Record<string, string> = {
  success: 'text-success',
  warning: 'text-warning',
  destructive: 'text-destructive',
  muted: 'text-muted-foreground',
}

export function GroupDetailSheet(props: GroupDetailSheetProps) {
  const { t } = useTranslation()
  const query = useAvailabilityGroup(props.group, props.range)
  const data = query.data
  const tone = successRateTone(data?.success_rate)

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent
        side='right'
        className='flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl'
      >
        <SheetHeader className='border-border/60 border-b px-6 py-4'>
          <div className='flex items-center justify-between gap-3'>
            <div className='flex flex-col gap-1'>
              <SheetTitle className='text-xl'>
                {props.group || t('Group Availability')}
              </SheetTitle>
              <SheetDescription className='text-xs'>
                {t('Group Availability Description')}
              </SheetDescription>
            </div>
            <RangeTabs value={props.range} onChange={props.onRangeChange} />
          </div>
        </SheetHeader>

        <div className='flex-1 overflow-y-auto px-6 py-5'>
          <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
            <StatBlock
              icon={Gauge}
              label={t('Success Rate')}
              value={formatPercent(data?.success_rate)}
              tone={tone}
              loading={query.isLoading}
            />
            <StatBlock
              icon={CircleDot}
              label={t('Online Rate')}
              value={
                data && data.channel_total > 0
                  ? `${data.channel_online}/${data.channel_total}`
                  : '—'
              }
              loading={query.isLoading}
            />
            <StatBlock
              icon={Zap}
              label={t('Request Count')}
              value={formatCount(data?.request_count)}
              loading={query.isLoading}
            />
            <StatBlock
              icon={Timer}
              label={
                props.range === '30m' ? t('P95 Latency') : t('Avg Latency')
              }
              value={
                props.range === '30m'
                  ? formatLatency(data?.p95_ms ?? undefined)
                  : formatLatency(data?.avg_latency_ms)
              }
              loading={query.isLoading}
            />
          </div>

          <div className='mt-6'>
            <div className='text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide'>
              {t('Trend')}
            </div>
            {query.isLoading ? (
              <Skeleton className='h-72 w-full rounded-lg' />
            ) : (
              <TimeseriesChart
                points={data?.points ?? []}
                range={props.range}
              />
            )}
          </div>

          <div className='mt-6'>
            <div className='text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide'>
              {t('Channel Status')}
            </div>
            {query.isLoading ? (
              <Skeleton className='h-40 w-full rounded-lg' />
            ) : (
              <ChannelTable rows={data?.channels ?? []} />
            )}
          </div>

          {query.isError ? (
            <div className='border-destructive/40 bg-destructive/5 text-destructive mt-4 rounded-lg border px-3 py-2 text-sm'>
              {(query.error as Error)?.message || t('Failed to load')}
            </div>
          ) : null}
        </div>
      </SheetContent>
    </Sheet>
  )
}

interface StatBlockProps {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  tone?: 'success' | 'warning' | 'destructive' | 'muted'
  loading?: boolean
}

function StatBlock(p: StatBlockProps) {
  const Icon = p.icon
  const cls = p.tone ? TONE_TEXT[p.tone] : 'text-foreground'
  return (
    <div className='border-border/60 bg-background/60 flex flex-col gap-1 rounded-xl border px-4 py-3'>
      <div className='text-muted-foreground flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wide'>
        <Icon className='size-3.5' />
        {p.label}
      </div>
      {p.loading ? (
        <Skeleton className='h-7 w-20' />
      ) : (
        <div className={cn('font-mono text-2xl font-semibold tabular-nums', cls)}>
          {p.value}
        </div>
      )}
    </div>
  )
}
