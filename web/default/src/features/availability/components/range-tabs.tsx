import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import type { AvailabilityRange } from '../types'

interface RangeTabsProps {
  value: AvailabilityRange
  onChange: (v: AvailabilityRange) => void
  className?: string
}

const ORDER: AvailabilityRange[] = ['30m', '1d', '7d']

export function RangeTabs(props: RangeTabsProps) {
  const { t } = useTranslation()
  const labels: Record<AvailabilityRange, string> = {
    '30m': t('Last 30 minutes'),
    '1d': t('Last 24 hours'),
    '7d': t('Last 7 days'),
  }
  return (
    <div
      role='tablist'
      className={cn(
        'bg-muted/60 inline-flex items-center gap-0.5 rounded-lg p-0.5',
        props.className
      )}
    >
      {ORDER.map((key) => {
        const active = props.value === key
        return (
          <button
            key={key}
            role='tab'
            type='button'
            aria-selected={active}
            onClick={() => props.onChange(key)}
            className={cn(
              'rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
              active
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {labels[key]}
          </button>
        )
      })}
    </div>
  )
}
