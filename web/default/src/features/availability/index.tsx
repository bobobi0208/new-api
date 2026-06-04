import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { GroupCard } from './components/group-card'
import { GroupDetailSheet } from './components/group-detail-sheet'
import { RangeTabs } from './components/range-tabs'
import { useAvailabilityOverview } from './hooks/use-availability'
import type { AvailabilityRange } from './types'

export function AvailabilityPage() {
  const { t } = useTranslation()
  const [range, setRange] = useState<AvailabilityRange>('30m')
  const [activeGroup, setActiveGroup] = useState<string | null>(null)
  const [detailRange, setDetailRange] = useState<AvailabilityRange>('30m')

  const query = useAvailabilityOverview(range)

  const groups = useMemo(() => query.data?.groups ?? [], [query.data])
  const generatedAt = query.data?.generated_at

  const openGroup = (g: string) => {
    setActiveGroup(g)
    setDetailRange(range)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Availability Monitor')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex items-center gap-2'>
          <RangeTabs value={range} onChange={setRange} />
          <Button
            variant='outline'
            size='sm'
            onClick={() => query.refetch()}
            disabled={query.isFetching}
          >
            <RefreshCw
              className={cn(
                'size-3.5',
                query.isFetching && 'animate-spin'
              )}
            />
            <span className='hidden sm:inline'>{t('Refresh')}</span>
          </Button>
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-5'>
          <Summary
            total={groups.length}
            generatedAt={generatedAt}
            loading={query.isLoading}
          />

          {query.isError ? (
            <div className='border-destructive/40 bg-destructive/5 text-destructive rounded-lg border px-4 py-3 text-sm'>
              {(query.error as Error)?.message || t('Failed to load')}
            </div>
          ) : null}

          {query.isLoading ? (
            <div className='grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3'>
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className='h-44 rounded-2xl' />
              ))}
            </div>
          ) : groups.length === 0 ? (
            <div className='border-border bg-muted/20 text-muted-foreground flex h-40 w-full items-center justify-center rounded-lg border border-dashed text-sm'>
              {t('No groups available')}
            </div>
          ) : (
            <div className='grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3'>
              {groups.map((g) => (
                <GroupCard
                  key={g.group}
                  snapshot={g}
                  onClick={() => openGroup(g.group)}
                />
              ))}
            </div>
          )}
        </div>
      </SectionPageLayout.Content>

      <GroupDetailSheet
        group={activeGroup}
        open={activeGroup !== null}
        onOpenChange={(open) => {
          if (!open) setActiveGroup(null)
        }}
        range={detailRange}
        onRangeChange={setDetailRange}
      />
    </SectionPageLayout>
  )
}

interface SummaryProps {
  total: number
  generatedAt?: number
  loading?: boolean
}

function Summary(p: SummaryProps) {
  const { t } = useTranslation()
  const stamp = p.generatedAt
    ? new Date(p.generatedAt * 1000).toLocaleTimeString()
    : null
  return (
    <div className='border-border/60 bg-card flex flex-wrap items-center justify-between gap-3 rounded-xl border px-4 py-3'>
      <div className='flex flex-col gap-0.5'>
        <span className='text-foreground text-sm font-medium'>
          {p.loading ? t('Loading…') : t('{{n}} groups monitored', { n: p.total })}
        </span>
        <span className='text-muted-foreground text-xs'>
          {t('Snapshot uses the last 30 minutes; trend view uses the selected range.')}
        </span>
      </div>
      {stamp ? (
        <span className='text-muted-foreground font-mono text-xs'>
          {t('Updated at')} {stamp}
        </span>
      ) : null}
    </div>
  )
}
