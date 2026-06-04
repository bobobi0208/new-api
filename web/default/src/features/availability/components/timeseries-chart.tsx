import { useEffect, useMemo, useState } from 'react'
import { VChart } from '@visactor/react-vchart'
import { useTranslation } from 'react-i18next'
import { useTheme } from '@/context/theme-provider'
import { VCHART_OPTION } from '@/lib/vchart'
import type { AvailabilityBucket, AvailabilityRange } from '../types'

interface TimeseriesChartProps {
  points: AvailabilityBucket[]
  range: AvailabilityRange
  loading?: boolean
}

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

function formatBucketLabel(unixSec: number, range: AvailabilityRange): string {
  const d = new Date(unixSec * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  if (range === '7d') {
    return `${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:00`
  }
  if (range === '1d') {
    return `${pad(d.getHours())}:${pad(d.getMinutes())}`
  }
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function TimeseriesChart(props: TimeseriesChartProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const [themeReady, setThemeReady] = useState(false)

  useEffect(() => {
    const setup = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }
    setup()
  }, [resolvedTheme])

  const data = useMemo(() => {
    const rows = props.points.map((p) => {
      const total = p.success_count + p.error_count
      const rate = total > 0 ? (p.success_count / total) * 100 : null
      return {
        time: formatBucketLabel(p.bucket, props.range),
        success_rate: rate,
        request_count: total,
        success_count: p.success_count,
        error_count: p.error_count,
      }
    })
    return rows
  }, [props.points, props.range])

  const spec = useMemo(
    () => ({
      type: 'common',
      data: [{ id: 'metrics', values: data }],
      seriesField: 'kind',
      axes: [
        {
          orient: 'left',
          id: 'left',
          type: 'linear',
          min: 0,
          max: 100,
          label: {
            formatMethod: (v: number) => `${v}%`,
          },
          title: { visible: false },
        },
        {
          orient: 'right',
          id: 'right',
          type: 'linear',
          min: 0,
          label: { visible: true },
          grid: { visible: false },
          title: { visible: false },
        },
        {
          orient: 'bottom',
          type: 'band',
          label: {
            autoHide: true,
            autoLimit: true,
          },
        },
      ],
      series: [
        {
          type: 'bar',
          id: 'requests',
          dataIndex: 0,
          xField: 'time',
          yField: 'request_count',
          axisId: 'right',
          name: t('Request Count'),
          bar: {
            style: {
              fillOpacity: 0.45,
              cornerRadius: [3, 3, 0, 0],
            },
          },
        },
        {
          type: 'line',
          id: 'success_rate',
          dataIndex: 0,
          xField: 'time',
          yField: 'success_rate',
          axisId: 'left',
          name: t('Success Rate'),
          point: { visible: false },
          line: { style: { lineWidth: 2 } },
          invalidType: 'break',
        },
      ],
      legends: [
        {
          visible: true,
          position: 'start',
          orient: 'top',
        },
      ],
      tooltip: {
        mark: { visible: true },
        dimension: { visible: true },
      },
      animation: false,
    }),
    [data, t]
  )

  if (!themeReady) {
    return <div className='bg-muted/30 h-72 w-full animate-pulse rounded-lg' />
  }

  if (data.length === 0 || data.every((d) => d.request_count === 0)) {
    return (
      <div className='border-border bg-muted/20 text-muted-foreground flex h-72 w-full items-center justify-center rounded-lg border border-dashed text-sm'>
        {t('No data in this window')}
      </div>
    )
  }

  return (
    <div className='h-72 w-full'>
      <VChart
        spec={spec}
        options={VCHART_OPTION}
        key={`${props.range}-${data.length}`}
      />
    </div>
  )
}
