/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { memo, useMemo } from 'react'
import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { EXCLUDED_GROUPS } from '../constants'
import {
  computeRequestPriceUSD,
  computeTokenPriceUSD,
  formatDualCurrency,
} from '../lib/price'
import type { PricingModel, PriceType, TokenUnit } from '../types'

export interface ModelComparisonRowProps {
  model: PricingModel
  groupRatio: Record<string, number>
  usableGroup: Record<string, { desc: string; ratio: number }>
  priceRate: number
  usdExchangeRate: number
  tokenUnit: TokenUnit
  showRechargePrice: boolean
  onModelClick?: (modelName: string) => void
}

type PriceRow = {
  key: PriceType
  label: string
}

const ORIGINAL_GROUP_KEY = '__original__'

function buildPriceRows(model: PricingModel, t: (k: string) => string): PriceRow[] {
  const rows: PriceRow[] = [
    { key: 'input', label: t('Input price') },
    { key: 'output', label: t('Completion price') },
  ]
  if (model.cache_ratio != null) {
    rows.push({ key: 'cache', label: t('Cache Read') })
  }
  if (model.create_cache_ratio != null) {
    rows.push({ key: 'create_cache', label: t('Cache Creation') })
  }
  return rows
}

function formatRatioLabel(ratio: number): string {
  // Avoid floating point cruft (1.0000000001x)
  const rounded = Number(ratio.toFixed(4))
  return `${rounded}x`
}

function formatDiscountLabel(ratio: number): string {
  // 折 = ratio × 10, two decimals, drop trailing zeros
  const folded = ratio * 10
  const text = folded.toFixed(2).replace(/\.?0+$/, '')
  return `${text}折`
}

function getGroupTag(
  group: string,
  desc: string | undefined
): { tag: string | null; subtitle: string | null } {
  if (!desc || !desc.trim()) {
    return { tag: null, subtitle: null }
  }
  const trimmed = desc.trim()
  // Try to split "PREFIX rest..." or "PREFIX (subtitle)" style descriptions
  // into a chip prefix + sub-line. Otherwise use the whole desc as subtitle.
  const match = trimmed.match(/^([A-Za-z0-9一-龥]{1,8})(?:\s+|（|\(|-)/)
  if (match) {
    const tag = match[1]
    const subtitle = trimmed.slice(tag.length).trim()
    return { tag, subtitle: subtitle || null }
  }
  if (trimmed.length <= 6) {
    return { tag: trimmed, subtitle: null }
  }
  return { tag: null, subtitle: trimmed }
}

interface PriceCellValues {
  rows: { row: PriceRow; usd: string; cny: string }[]
  perRequest?: { usd: string; cny: string }
}

function computeCellValues(
  model: PricingModel,
  rows: PriceRow[],
  ratio: number,
  tokenUnit: TokenUnit,
  showRechargePrice: boolean,
  priceRate: number,
  usdExchangeRate: number
): PriceCellValues {
  if (model.quota_type === 1) {
    const usd = computeRequestPriceUSD(
      model,
      ratio,
      showRechargePrice,
      priceRate,
      usdExchangeRate
    )
    return {
      rows: [],
      perRequest: formatDualCurrency(usd, usdExchangeRate),
    }
  }
  return {
    rows: rows.map((row) => {
      const usd = computeTokenPriceUSD(
        model,
        row.key,
        ratio,
        tokenUnit,
        showRechargePrice,
        priceRate,
        usdExchangeRate
      )
      const formatted = formatDualCurrency(usd, usdExchangeRate)
      return { row, usd: formatted.usd, cny: formatted.cny }
    }),
  }
}

interface PriceCardProps {
  variant: 'original' | 'group'
  title: string
  subtitle?: string | null
  tag?: string | null
  ratioBadge?: string
  discountBadge?: string
  copyValue?: string
  values: PriceCellValues
  tokenUnitLabel: string
  perRequestLabel: string
}

function PriceCard(props: PriceCardProps) {
  const { copyToClipboard } = useCopyToClipboard()
  const isOriginal = props.variant === 'original'

  const containerCls = cn(
    'flex w-[260px] shrink-0 snap-start flex-col rounded-xl border p-3 sm:w-[280px]',
    isOriginal
      ? 'border-sky-200/80 bg-sky-50/60 dark:border-sky-500/30 dark:bg-sky-500/10'
      : 'border-amber-200/80 bg-amber-50/60 dark:border-amber-500/30 dark:bg-amber-500/10'
  )

  const valueColorCls = isOriginal
    ? 'text-sky-700 dark:text-sky-300'
    : 'text-amber-700 dark:text-amber-300'

  const tagCls = isOriginal
    ? 'bg-sky-100 text-sky-700 dark:bg-sky-500/20 dark:text-sky-200'
    : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-200'

  const ratioCls = isOriginal
    ? 'bg-sky-100 text-sky-700 dark:bg-sky-500/20 dark:text-sky-200'
    : 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-200'

  const discountCls = isOriginal
    ? ''
    : 'bg-amber-500/15 text-amber-700 dark:bg-amber-500/25 dark:text-amber-200'

  return (
    <div className={containerCls}>
      <div className='mb-2 flex items-start justify-between gap-2'>
        <div className='flex min-w-0 flex-col gap-1'>
          {props.tag ? (
            <span
              className={cn(
                'inline-flex h-5 w-fit items-center rounded-md px-1.5 text-[10px] font-medium tracking-wide',
                tagCls
              )}
            >
              {props.tag}
            </span>
          ) : null}
          <div className='flex items-center gap-1.5 min-w-0'>
            <span className='truncate font-semibold text-sm text-foreground'>
              {props.title}
            </span>
            {props.copyValue ? (
              <Button
                type='button'
                variant='ghost'
                size='icon'
                className='size-5 text-muted-foreground hover:text-foreground'
                onClick={(e) => {
                  e.stopPropagation()
                  copyToClipboard(props.copyValue || '')
                }}
                aria-label='copy'
              >
                <Copy className='size-3' />
              </Button>
            ) : null}
          </div>
        </div>
        <div className='flex shrink-0 flex-col items-end gap-1'>
          {props.ratioBadge ? (
            <span
              className={cn(
                'inline-flex h-5 items-center rounded-full px-2 font-mono text-[11px] font-medium tabular-nums',
                ratioCls
              )}
            >
              {props.ratioBadge}
            </span>
          ) : null}
          {props.discountBadge ? (
            <span
              className={cn(
                'inline-flex h-5 items-center rounded-full px-2 font-mono text-[11px] font-medium tabular-nums',
                discountCls
              )}
            >
              {props.discountBadge}
            </span>
          ) : null}
        </div>
      </div>

      {props.subtitle ? (
        <div className='text-muted-foreground -mt-1 mb-2 line-clamp-1 text-[11px]'>
          {props.subtitle}
        </div>
      ) : null}

      {props.values.perRequest ? (
        <div className='border-t border-current/10 pt-2 first:border-t-0 first:pt-0'>
          <div className='text-muted-foreground text-xs'>
            {props.perRequestLabel}
          </div>
          <div
            className={cn(
              'mt-0.5 text-base font-semibold tabular-nums',
              valueColorCls
            )}
          >
            {props.values.perRequest.usd}
          </div>
          <div className='text-muted-foreground text-[11px] tabular-nums'>
            {props.values.perRequest.cny}
          </div>
        </div>
      ) : (
        <div className='space-y-1.5'>
          {props.values.rows.map(({ row, usd, cny }) => (
            <div
              key={row.key}
              className='flex items-baseline justify-between gap-2'
            >
              <span className='text-muted-foreground shrink-0 truncate text-[11px]'>
                {row.label}
              </span>
              <div className='flex flex-col items-end'>
                <span
                  className={cn(
                    'font-mono text-sm font-semibold tabular-nums',
                    valueColorCls
                  )}
                >
                  {usd}{' '}
                  <span className='text-muted-foreground font-normal'>
                    / {props.tokenUnitLabel}
                  </span>
                </span>
                <span className='text-muted-foreground font-mono text-[10px] tabular-nums'>
                  {cny}{' '}
                  <span className='font-normal'>/ {props.tokenUnitLabel}</span>
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export const ModelComparisonRow = memo(function ModelComparisonRow(
  props: ModelComparisonRowProps
) {
  const { t } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const vendorIcon = props.model.vendor_icon
    ? getLobeIcon(props.model.vendor_icon, 24)
    : null
  const initial = props.model.model_name?.charAt(0).toUpperCase() || '?'
  const tokenUnitLabel = props.tokenUnit === 'K' ? '1K' : 'M'
  const perRequestLabel = t('Per request price')

  const priceRows = useMemo(
    () => buildPriceRows(props.model, t),
    [props.model, t]
  )

  const groups = useMemo(() => {
    const enable = Array.isArray(props.model.enable_groups)
      ? props.model.enable_groups
      : []
    return enable
      .filter((g) => g && !EXCLUDED_GROUPS.includes(g))
      .map((g) => ({
        name: g,
        ratio: props.groupRatio[g] ?? 1,
        desc: props.usableGroup?.[g]?.desc,
      }))
      .sort((a, b) => a.ratio - b.ratio)
  }, [props.model.enable_groups, props.groupRatio, props.usableGroup])

  const originalValues = useMemo(
    () =>
      computeCellValues(
        props.model,
        priceRows,
        1,
        props.tokenUnit,
        props.showRechargePrice,
        props.priceRate,
        props.usdExchangeRate
      ),
    [
      props.model,
      priceRows,
      props.tokenUnit,
      props.showRechargePrice,
      props.priceRate,
      props.usdExchangeRate,
    ]
  )

  return (
    <div className='rounded-xl border bg-card'>
      <div className='flex flex-wrap items-center gap-3 border-b px-4 py-3'>
        <div className='bg-muted/40 flex size-9 shrink-0 items-center justify-center rounded-lg'>
          {vendorIcon || (
            <span className='text-sm font-semibold'>{initial}</span>
          )}
        </div>
        <button
          type='button'
          onClick={() => props.onModelClick?.(props.model.model_name)}
          className='hover:text-primary text-foreground text-left text-base font-semibold tracking-tight transition-colors'
        >
          {props.model.model_name}
        </button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          className='size-6 text-muted-foreground hover:text-foreground'
          onClick={(e) => {
            e.stopPropagation()
            copyToClipboard(props.model.model_name)
          }}
          aria-label={t('Copy model name')}
        >
          <Copy className='size-3.5' />
        </Button>
        {props.model.vendor_name ? (
          <span className='inline-flex h-5 items-center rounded-md bg-muted px-1.5 text-[11px] text-muted-foreground'>
            {props.model.vendor_name}
          </span>
        ) : null}
        {props.model.vendor_description ? (
          <span className='text-muted-foreground/80 text-xs'>
            {props.model.vendor_description}
          </span>
        ) : null}
      </div>

      <div
        className='hover-scrollbar flex snap-x snap-mandatory gap-3 overflow-x-auto px-4 py-4'
        tabIndex={0}
        aria-label={`${props.model.model_name} ${t('Group price comparison')}`}
      >
        <PriceCard
          variant='original'
          title={t('Original Price')}
          tag={t('Public list price')}
          values={originalValues}
          tokenUnitLabel={tokenUnitLabel}
          perRequestLabel={perRequestLabel}
        />
        {groups.length === 0 ? (
          <div className='flex items-center justify-center px-4 text-sm text-muted-foreground'>
            {t('No group pricing available for this model')}
          </div>
        ) : (
          groups.map((g) => {
            const values = computeCellValues(
              props.model,
              priceRows,
              g.ratio,
              props.tokenUnit,
              props.showRechargePrice,
              props.priceRate,
              props.usdExchangeRate
            )
            const tag = getGroupTag(g.name, g.desc)
            return (
              <PriceCard
                key={g.name}
                variant='group'
                title={g.name}
                copyValue={g.name}
                subtitle={tag.subtitle}
                tag={tag.tag}
                ratioBadge={formatRatioLabel(g.ratio)}
                discountBadge={
                  g.ratio !== 1 ? formatDiscountLabel(g.ratio) : undefined
                }
                values={values}
                tokenUnitLabel={tokenUnitLabel}
                perRequestLabel={perRequestLabel}
              />
            )
          })
        )}
      </div>
    </div>
  )
})
