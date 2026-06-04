/*
Copyright (C) 2025 QuantumNous

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

import React from 'react';
import {
  Avatar,
  Button,
  Empty,
  Pagination,
  Tag,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Copy } from 'lucide-react';
import { getLobeHubIcon } from '../../../../../helpers';
import PricingCardSkeleton from '../card/PricingCardSkeleton';
import { useMinimumLoadingTime } from '../../../../../hooks/common/useMinimumLoadingTime';
import { useIsMobile } from '../../../../../hooks/common/useIsMobile';

const EXCLUDED_GROUPS = new Set(['', 'auto']);

// Mirrors web/default/src/features/pricing/lib/price.ts so the two themes show
// identical numbers for the same model + group + token-unit + recharge config.
function calculateTokenPriceUSD(model, type, ratio) {
  const base = model.model_ratio * 2 * ratio;
  switch (type) {
    case 'input':
      return base;
    case 'output':
      return base * (model.completion_ratio ?? 1);
    case 'cache':
      return model.cache_ratio != null ? base * Number(model.cache_ratio) : NaN;
    case 'create_cache':
      return model.create_cache_ratio != null
        ? base * Number(model.create_cache_ratio)
        : NaN;
    default:
      return NaN;
  }
}

function applyRechargeRate(price, showRecharge, priceRate, usdExchangeRate) {
  if (!showRecharge) return price;
  return (price * priceRate) / usdExchangeRate;
}

function computeTokenPriceUSD(
  model,
  type,
  ratio,
  tokenUnit,
  showRecharge,
  priceRate,
  usdExchangeRate,
) {
  if (model.quota_type === 1) return NaN;
  let priceUSD = calculateTokenPriceUSD(model, type, ratio);
  priceUSD = applyRechargeRate(
    priceUSD,
    showRecharge,
    priceRate,
    usdExchangeRate,
  );
  const divisor = tokenUnit === 'K' ? 1000 : 1;
  return priceUSD / divisor;
}

function computeRequestPriceUSD(
  model,
  ratio,
  showRecharge,
  priceRate,
  usdExchangeRate,
) {
  if (model.quota_type !== 1) return NaN;
  let priceUSD = parseFloat(model.model_price || 0) * ratio;
  priceUSD = applyRechargeRate(
    priceUSD,
    showRecharge,
    priceRate,
    usdExchangeRate,
  );
  return priceUSD;
}

function formatUSDLabel(value) {
  if (!Number.isFinite(value)) return '-';
  if (value === 0) return '$0';
  const abs = Math.abs(value);
  if (abs < 0.0001) return '$' + value.toFixed(6);
  return '$' + value.toFixed(4);
}

function formatCNYLabel(value) {
  if (!Number.isFinite(value)) return '-';
  if (value === 0) return '¥0';
  const abs = Math.abs(value);
  if (abs < 0.01) return '¥' + value.toFixed(4);
  return '¥' + value.toFixed(2);
}

function formatDualCurrency(usdValue, usdExchangeRate) {
  if (!Number.isFinite(usdValue)) return { usd: '-', cny: '-' };
  return {
    usd: formatUSDLabel(usdValue),
    cny: formatCNYLabel(usdValue * usdExchangeRate),
  };
}

function buildPriceRows(model, t) {
  const rows = [
    { key: 'input', label: t('输入价格') },
    { key: 'output', label: t('补全价格') },
  ];
  if (model.cache_ratio != null) {
    rows.push({ key: 'cache', label: t('缓存读取') });
  }
  if (model.create_cache_ratio != null) {
    rows.push({ key: 'create_cache', label: t('缓存创建') });
  }
  return rows;
}

function formatRatioLabel(ratio) {
  const rounded = Number(ratio.toFixed(4));
  return `${rounded}x`;
}

function formatDiscountLabel(ratio) {
  const folded = ratio * 10;
  const text = folded.toFixed(2).replace(/\.?0+$/, '');
  return `${text}折`;
}

function getGroupTag(desc) {
  if (!desc || !desc.trim()) return { tag: null, subtitle: null };
  const trimmed = desc.trim();
  const match = trimmed.match(/^([A-Za-z0-9一-龥]{1,8})(?:\s+|（|\(|-)/);
  if (match) {
    const tag = match[1];
    const subtitle = trimmed.slice(tag.length).trim();
    return { tag, subtitle: subtitle || null };
  }
  if (trimmed.length <= 6) return { tag: trimmed, subtitle: null };
  return { tag: null, subtitle: trimmed };
}

function computeCellValues(
  model,
  rows,
  ratio,
  tokenUnit,
  showRecharge,
  priceRate,
  usdExchangeRate,
) {
  if (model.quota_type === 1) {
    const usd = computeRequestPriceUSD(
      model,
      ratio,
      showRecharge,
      priceRate,
      usdExchangeRate,
    );
    return {
      rows: [],
      perRequest: formatDualCurrency(usd, usdExchangeRate),
    };
  }
  return {
    rows: rows.map((row) => {
      const usd = computeTokenPriceUSD(
        model,
        row.key,
        ratio,
        tokenUnit,
        showRecharge,
        priceRate,
        usdExchangeRate,
      );
      const formatted = formatDualCurrency(usd, usdExchangeRate);
      return { row, usd: formatted.usd, cny: formatted.cny };
    }),
  };
}

const PriceCard = ({
  variant,
  title,
  subtitle,
  tag,
  ratioBadge,
  discountBadge,
  copyValue,
  values,
  tokenUnitLabel,
  perRequestLabel,
  copyText,
}) => {
  const isOriginal = variant === 'original';
  const containerCls = `flex w-[260px] shrink-0 flex-col rounded-xl border p-3 sm:w-[280px] ${
    isOriginal
      ? 'border-sky-200 bg-sky-50'
      : 'border-amber-200 bg-amber-50'
  }`;
  const valueColorCls = isOriginal ? 'text-sky-700' : 'text-amber-700';
  const tagCls = isOriginal
    ? 'bg-sky-100 text-sky-700'
    : 'bg-emerald-100 text-emerald-700';
  const ratioCls = isOriginal
    ? 'bg-sky-100 text-sky-700'
    : 'bg-amber-100 text-amber-700';
  const discountCls = isOriginal
    ? 'bg-sky-100 text-sky-700'
    : 'bg-amber-200 text-amber-800';

  return (
    <div className={containerCls}>
      <div className='mb-2 flex items-start justify-between gap-2'>
        <div className='flex min-w-0 flex-col gap-1'>
          {tag ? (
            <span
              className={`inline-flex h-5 w-fit items-center rounded-md px-1.5 text-[10px] font-medium tracking-wide ${tagCls}`}
            >
              {tag}
            </span>
          ) : null}
          <div className='flex items-center gap-1.5 min-w-0'>
            <span className='truncate font-semibold text-sm text-gray-900'>
              {title}
            </span>
            {copyValue ? (
              <Button
                size='small'
                theme='borderless'
                type='tertiary'
                icon={<Copy size={12} />}
                onClick={(e) => {
                  e.stopPropagation();
                  copyText(copyValue);
                }}
              />
            ) : null}
          </div>
        </div>
        <div className='flex shrink-0 flex-col items-end gap-1'>
          {ratioBadge ? (
            <span
              className={`inline-flex h-5 items-center rounded-full px-2 font-mono text-[11px] font-medium tabular-nums ${ratioCls}`}
            >
              {ratioBadge}
            </span>
          ) : null}
          {discountBadge ? (
            <span
              className={`inline-flex h-5 items-center rounded-full px-2 font-mono text-[11px] font-medium tabular-nums ${discountCls}`}
            >
              {discountBadge}
            </span>
          ) : null}
        </div>
      </div>

      {subtitle ? (
        <div className='-mt-1 mb-2 line-clamp-1 text-[11px] text-gray-500'>
          {subtitle}
        </div>
      ) : null}

      {values.perRequest ? (
        <div className='pt-1'>
          <div className='text-xs text-gray-500'>{perRequestLabel}</div>
          <div
            className={`mt-0.5 text-base font-semibold tabular-nums ${valueColorCls}`}
          >
            {values.perRequest.usd}
          </div>
          <div className='text-[11px] text-gray-500 tabular-nums'>
            {values.perRequest.cny}
          </div>
        </div>
      ) : (
        <div className='space-y-1.5'>
          {values.rows.map(({ row, usd, cny }) => (
            <div
              key={row.key}
              className='flex items-baseline justify-between gap-2'
            >
              <span className='shrink-0 truncate text-[11px] text-gray-500'>
                {row.label}
              </span>
              <div className='flex flex-col items-end'>
                <span
                  className={`font-mono text-sm font-semibold tabular-nums ${valueColorCls}`}
                >
                  {usd}{' '}
                  <span className='text-gray-500 font-normal'>
                    / {tokenUnitLabel}
                  </span>
                </span>
                <span className='font-mono text-[10px] text-gray-500 tabular-nums'>
                  {cny} <span className='font-normal'>/ {tokenUnitLabel}</span>
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const ComparisonRow = ({
  model,
  groupRatio,
  usableGroup,
  tokenUnit,
  showWithRecharge,
  priceRate,
  usdExchangeRate,
  copyText,
  openModelDetail,
  t,
}) => {
  const vendorIcon = model.vendor_icon
    ? getLobeHubIcon(model.vendor_icon, 24)
    : null;
  const initial = (model.model_name || '?').charAt(0).toUpperCase();
  const tokenUnitLabel = tokenUnit === 'K' ? '1K' : 'M';
  const perRequestLabel = t('每次调用价');
  const priceRows = buildPriceRows(model, t);

  const enable = Array.isArray(model.enable_groups) ? model.enable_groups : [];
  const groups = enable
    .filter((g) => g && !EXCLUDED_GROUPS.has(g))
    .map((g) => ({
      name: g,
      ratio: groupRatio[g] ?? 1,
      desc: usableGroup?.[g]?.desc,
    }))
    .sort((a, b) => a.ratio - b.ratio);

  const originalValues = computeCellValues(
    model,
    priceRows,
    1,
    tokenUnit,
    showWithRecharge,
    priceRate,
    usdExchangeRate,
  );

  return (
    <div className='rounded-xl border border-gray-200 bg-white'>
      <div className='flex flex-wrap items-center gap-3 border-b border-gray-100 px-4 py-3'>
        <div className='flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-gray-100'>
          {vendorIcon || (
            <Avatar size='small' style={{ borderRadius: 8 }}>
              {initial}
            </Avatar>
          )}
        </div>
        <button
          type='button'
          onClick={() => openModelDetail && openModelDetail(model)}
          className='text-left text-base font-semibold tracking-tight text-gray-900 transition-colors hover:text-blue-600'
        >
          {model.model_name}
        </button>
        <Button
          size='small'
          theme='borderless'
          type='tertiary'
          icon={<Copy size={14} />}
          onClick={(e) => {
            e.stopPropagation();
            copyText(model.model_name);
          }}
        />
        {model.vendor_name ? (
          <Tag size='small' color='white' shape='circle'>
            {model.vendor_name}
          </Tag>
        ) : null}
      </div>

      <div
        className='flex snap-x snap-mandatory gap-3 overflow-x-auto px-4 py-4'
        tabIndex={0}
      >
        <PriceCard
          variant='original'
          title={t('原价')}
          tag={t('官方公开价格')}
          values={originalValues}
          tokenUnitLabel={tokenUnitLabel}
          perRequestLabel={perRequestLabel}
          copyText={copyText}
        />
        {groups.length === 0 ? (
          <div className='flex items-center justify-center px-4 text-sm text-gray-500'>
            {t('该模型暂无分组定价')}
          </div>
        ) : (
          groups.map((g) => {
            const values = computeCellValues(
              model,
              priceRows,
              g.ratio,
              tokenUnit,
              showWithRecharge,
              priceRate,
              usdExchangeRate,
            );
            const tag = getGroupTag(g.desc);
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
                  g.ratio !== 1 ? formatDiscountLabel(g.ratio) : null
                }
                values={values}
                tokenUnitLabel={tokenUnitLabel}
                perRequestLabel={perRequestLabel}
                copyText={copyText}
              />
            );
          })
        )}
      </div>
    </div>
  );
};

const PricingComparisonView = ({
  filteredModels,
  loading,
  pageSize,
  setPageSize,
  currentPage,
  setCurrentPage,
  groupRatio,
  usableGroup,
  copyText,
  tokenUnit,
  showWithRecharge,
  priceRate,
  usdExchangeRate,
  openModelDetail,
  t,
}) => {
  const showSkeleton = useMinimumLoadingTime(loading);
  const isMobile = useIsMobile();
  const startIndex = (currentPage - 1) * pageSize;
  const paginatedModels = filteredModels.slice(
    startIndex,
    startIndex + pageSize,
  );

  if (showSkeleton) {
    return <PricingCardSkeleton showRatio={false} />;
  }

  if (!filteredModels || filteredModels.length === 0) {
    return (
      <div className='flex justify-center items-center py-20'>
        <Empty
          image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={t('搜索无结果')}
        />
      </div>
    );
  }

  return (
    <div className='px-2 pt-2'>
      <div className='flex flex-col gap-3'>
        {paginatedModels.map((model) => (
          <ComparisonRow
            key={model.key ?? model.model_name ?? model.id}
            model={model}
            groupRatio={groupRatio}
            usableGroup={usableGroup}
            tokenUnit={tokenUnit}
            showWithRecharge={showWithRecharge}
            priceRate={priceRate}
            usdExchangeRate={usdExchangeRate}
            copyText={copyText}
            openModelDetail={openModelDetail}
            t={t}
          />
        ))}
      </div>

      {filteredModels.length > 0 && (
        <div className='flex justify-center mt-6 py-4 border-t pricing-pagination-divider'>
          <Pagination
            currentPage={currentPage}
            pageSize={pageSize}
            total={filteredModels.length}
            showSizeChanger={true}
            pageSizeOptions={[10, 20, 50, 100]}
            size={isMobile ? 'small' : 'default'}
            showQuickJumper={isMobile}
            onPageChange={(page) => setCurrentPage(page)}
            onPageSizeChange={(size) => {
              setPageSize(size);
              setCurrentPage(1);
            }}
          />
        </div>
      )}
    </div>
  );
};

export default PricingComparisonView;
