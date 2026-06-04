import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import type { ChannelAvailabilityRow } from '../types'
import { formatLatency } from '../lib/format'

interface ChannelTableProps {
  rows: ChannelAvailabilityRow[]
}

const STATUS_LABEL_KEYS: Record<number, string> = {
  0: 'Unknown',
  1: 'Enabled',
  2: 'Manually Disabled',
  3: 'Auto Disabled',
}

const STATUS_TONE: Record<number, string> = {
  0: 'bg-muted text-muted-foreground',
  1: 'bg-success/15 text-success',
  2: 'bg-warning/15 text-warning',
  3: 'bg-destructive/15 text-destructive',
}

function formatTimestamp(ts: number): string {
  if (!ts) return '—'
  const d = new Date(ts * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function ChannelTable(props: ChannelTableProps) {
  const { t } = useTranslation()
  if (props.rows.length === 0) {
    return (
      <div className='border-border bg-muted/20 text-muted-foreground flex w-full items-center justify-center rounded-lg border border-dashed py-10 text-sm'>
        {t('No channels in this group')}
      </div>
    )
  }
  return (
    <div className='border-border overflow-hidden rounded-lg border'>
      <table className='w-full text-sm'>
        <thead className='bg-muted/40 text-muted-foreground text-xs font-medium uppercase tracking-wide'>
          <tr>
            <th className='px-3 py-2.5 text-left'>{t('Channel')}</th>
            <th className='px-3 py-2.5 text-left'>{t('Status')}</th>
            <th className='px-3 py-2.5 text-right'>{t('Requests')}</th>
            <th className='px-3 py-2.5 text-right'>{t('Errors')}</th>
            <th className='px-3 py-2.5 text-right'>{t('Last Test')}</th>
            <th className='px-3 py-2.5 text-right'>{t('Test Latency')}</th>
          </tr>
        </thead>
        <tbody className='divide-border/60 divide-y'>
          {props.rows.map((r) => {
            const errRate =
              r.request_count > 0 ? r.error_count / r.request_count : 0
            return (
              <tr key={r.channel_id} className='hover:bg-muted/30'>
                <td className='px-3 py-2.5'>
                  <div className='flex flex-col'>
                    <span className='text-foreground font-medium'>
                      {r.name || `#${r.channel_id}`}
                    </span>
                    <span className='text-muted-foreground text-[11px]'>
                      ID {r.channel_id}
                    </span>
                  </div>
                </td>
                <td className='px-3 py-2.5'>
                  <span
                    className={cn(
                      'inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium',
                      STATUS_TONE[r.status] ?? STATUS_TONE[0]
                    )}
                  >
                    {t(STATUS_LABEL_KEYS[r.status] ?? 'Unknown')}
                  </span>
                </td>
                <td className='px-3 py-2.5 text-right font-mono tabular-nums'>
                  {r.request_count.toLocaleString()}
                </td>
                <td
                  className={cn(
                    'px-3 py-2.5 text-right font-mono tabular-nums',
                    r.error_count > 0
                      ? 'text-destructive'
                      : 'text-muted-foreground'
                  )}
                >
                  {r.error_count.toLocaleString()}
                  {errRate > 0.01 && r.request_count > 0 ? (
                    <span className='text-muted-foreground ml-1 text-[10px]'>
                      ({(errRate * 100).toFixed(1)}%)
                    </span>
                  ) : null}
                </td>
                <td className='text-muted-foreground px-3 py-2.5 text-right text-[11px]'>
                  {formatTimestamp(r.test_time)}
                </td>
                <td className='px-3 py-2.5 text-right font-mono tabular-nums'>
                  {formatLatency(r.response_time_ms)}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
