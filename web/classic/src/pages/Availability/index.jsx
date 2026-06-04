/*
Copyright (C) 2026 QuantumNous

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useMemo, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Modal,
  Spin,
  Table,
  Tabs,
  TabPane,
  Tag,
  Typography,
  Empty,
} from '@douyinfe/semi-ui';
import {
  IconRefresh,
  IconActivity,
  IconUser,
  IconClock,
  IconTickCircle,
} from '@douyinfe/semi-icons';
import { VChart } from '@visactor/react-vchart';
import { API, showError } from '../../helpers';
import { CHART_CONFIG } from '../../constants/dashboard.constants';

const REFRESH_INTERVAL_MS = {
  '30m': 10_000,
  '1d': 60_000,
  '7d': 300_000,
};

const STATUS_LABELS = {
  0: 'Unknown',
  1: 'Enabled',
  2: 'Manually Disabled',
  3: 'Auto Disabled',
};

const STATUS_COLOR = {
  0: 'grey',
  1: 'green',
  2: 'orange',
  3: 'red',
};

function formatPercent(v) {
  if (v === null || v === undefined || Number.isNaN(v)) return '—';
  return `${(v * 100).toFixed(1)}%`;
}

function formatLatency(v) {
  if (!v || v <= 0) return '—';
  if (v < 1000) return `${Math.round(v)} ms`;
  return `${(v / 1000).toFixed(2)} s`;
}

function formatBucketLabel(unixSec, range) {
  const d = new Date(unixSec * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  if (range === '7d') {
    return `${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:00`;
  }
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function rateTone(value) {
  if (value === null || value === undefined) return 'grey';
  if (value >= 0.99) return 'green';
  if (value >= 0.95) return 'orange';
  return 'red';
}

function onlineTone(online, total) {
  if (!total) return 'grey';
  if (online === total) return 'green';
  if (online * 2 >= total) return 'orange';
  return 'red';
}

const Availability = () => {
  const { t } = useTranslation();
  const [range, setRange] = useState('30m');
  const [overview, setOverview] = useState(null);
  const [overviewLoading, setOverviewLoading] = useState(false);
  const [activeGroup, setActiveGroup] = useState(null);
  const [detail, setDetail] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const fetchOverview = useCallback(async () => {
    setOverviewLoading(true);
    try {
      const res = await API.get('/api/availability/overview');
      if (res.data?.success) setOverview(res.data.data);
      else showError(res.data?.message || 'failed');
    } catch (e) {
      showError(e?.message || 'failed');
    } finally {
      setOverviewLoading(false);
    }
  }, []);

  const fetchDetail = useCallback(async (group, rangeKey) => {
    if (!group) return;
    setDetailLoading(true);
    try {
      const res = await API.get('/api/availability/group', {
        params: { group, range: rangeKey },
      });
      if (res.data?.success) setDetail(res.data.data);
      else showError(res.data?.message || 'failed');
    } catch (e) {
      showError(e?.message || 'failed');
    } finally {
      setDetailLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOverview();
    const id = setInterval(fetchOverview, REFRESH_INTERVAL_MS[range]);
    return () => clearInterval(id);
  }, [range, fetchOverview]);

  useEffect(() => {
    if (!activeGroup) return;
    fetchDetail(activeGroup, range);
    const id = setInterval(
      () => fetchDetail(activeGroup, range),
      REFRESH_INTERVAL_MS[range],
    );
    return () => clearInterval(id);
  }, [activeGroup, range, fetchDetail]);

  const chartSpec = useMemo(() => {
    if (!detail) return null;
    const rows = (detail.points ?? []).map((p) => {
      const total = p.success_count + p.error_count;
      const rate = total > 0 ? (p.success_count / total) * 100 : null;
      return {
        time: formatBucketLabel(p.bucket, detail.range),
        success_rate: rate,
        request_count: total,
      };
    });
    return {
      type: 'common',
      data: [{ id: 'metrics', values: rows }],
      axes: [
        {
          orient: 'left',
          id: 'left',
          type: 'linear',
          min: 0,
          max: 100,
          label: { formatMethod: (v) => `${v}%` },
        },
        {
          orient: 'right',
          id: 'right',
          type: 'linear',
          min: 0,
          grid: { visible: false },
        },
        { orient: 'bottom', type: 'band', label: { autoHide: true } },
      ],
      series: [
        {
          type: 'bar',
          id: 'requests',
          dataIndex: 0,
          xField: 'time',
          yField: 'request_count',
          axisId: 'right',
          name: t('请求数'),
          bar: { style: { fillOpacity: 0.45, cornerRadius: [3, 3, 0, 0] } },
        },
        {
          type: 'line',
          id: 'success_rate',
          dataIndex: 0,
          xField: 'time',
          yField: 'success_rate',
          axisId: 'left',
          name: t('成功率'),
          point: { visible: false },
          line: { style: { lineWidth: 2 } },
          invalidType: 'break',
        },
      ],
      legends: [{ visible: true, position: 'start', orient: 'top' }],
      tooltip: { mark: { visible: true }, dimension: { visible: true } },
      animation: false,
    };
  }, [detail, t]);

  const groups = overview?.groups ?? [];

  return (
    <div className='mt-[60px] px-2'>
      <div className='mx-auto max-w-screen-2xl space-y-4'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex flex-col'>
            <Typography.Title heading={4} className='!mb-1'>
              {t('可用性监控')}
            </Typography.Title>
            <Typography.Text type='tertiary' size='small'>
              {t('按分组维度查看成功率、渠道在线率与延迟趋势')}
            </Typography.Text>
          </div>
          <div className='flex items-center gap-2'>
            <Tabs
              type='button'
              size='small'
              activeKey={range}
              onChange={setRange}
            >
              <TabPane tab={t('最近 30 分钟')} itemKey='30m' />
              <TabPane tab={t('最近 24 小时')} itemKey='1d' />
              <TabPane tab={t('最近 7 天')} itemKey='7d' />
            </Tabs>
            <Button
              icon={<IconRefresh />}
              size='small'
              onClick={fetchOverview}
              loading={overviewLoading}
            >
              {t('刷新')}
            </Button>
          </div>
        </div>

        {overviewLoading && !overview ? (
          <div className='flex justify-center py-12'>
            <Spin size='large' />
          </div>
        ) : groups.length === 0 ? (
          <Empty
            image={<IconActivity size='extra-large' />}
            title={t('暂无可监控的分组')}
          />
        ) : (
          <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3'>
            {groups.map((g) => (
              <GroupCard
                key={g.group}
                snapshot={g}
                onClick={() => setActiveGroup(g.group)}
                t={t}
              />
            ))}
          </div>
        )}
      </div>

      <Modal
        title={`${t('分组')}: ${activeGroup ?? ''}`}
        visible={!!activeGroup}
        onCancel={() => {
          setActiveGroup(null);
          setDetail(null);
        }}
        width={Math.min(960, typeof window !== 'undefined' ? window.innerWidth - 40 : 960)}
        footer={null}
      >
        {detailLoading && !detail ? (
          <div className='flex justify-center py-12'>
            <Spin size='large' />
          </div>
        ) : detail ? (
          <div className='space-y-4'>
            <div className='grid grid-cols-2 gap-2 md:grid-cols-4'>
              <Metric
                icon={<IconActivity />}
                label={t('成功率')}
                value={formatPercent(detail.success_rate)}
                tone={rateTone(detail.success_rate)}
              />
              <Metric
                icon={<IconTickCircle />}
                label={t('在线率')}
                value={
                  detail.channel_total > 0
                    ? `${detail.channel_online}/${detail.channel_total}`
                    : '—'
                }
                tone={onlineTone(detail.channel_online, detail.channel_total)}
              />
              <Metric
                icon={<IconUser />}
                label={t('请求数')}
                value={detail.request_count?.toLocaleString() ?? '0'}
              />
              <Metric
                icon={<IconClock />}
                label={
                  detail.range === '30m' ? t('P95 延迟') : t('平均延迟')
                }
                value={
                  detail.range === '30m'
                    ? formatLatency(detail.p95_ms)
                    : formatLatency(detail.avg_latency_ms)
                }
              />
            </div>

            <Card title={t('趋势')} bordered className='!shadow-none'>
              <div style={{ height: 280 }}>
                {chartSpec ? (
                  <VChart spec={chartSpec} option={CHART_CONFIG} />
                ) : (
                  <Empty title={t('该时间窗内暂无数据')} />
                )}
              </div>
            </Card>

            <Card title={t('渠道状态')} bordered className='!shadow-none'>
              <Table
                dataSource={detail.channels ?? []}
                rowKey='channel_id'
                pagination={{ pageSize: 10, showSizeChanger: false }}
                size='small'
                empty={<Empty title={t('该分组下暂无渠道')} />}
                columns={[
                  {
                    title: t('渠道'),
                    dataIndex: 'name',
                    render: (val, row) => (
                      <div className='flex flex-col'>
                        <span className='font-medium'>
                          {val || `#${row.channel_id}`}
                        </span>
                        <Typography.Text type='tertiary' size='small'>
                          ID {row.channel_id}
                        </Typography.Text>
                      </div>
                    ),
                  },
                  {
                    title: t('状态'),
                    dataIndex: 'status',
                    render: (s) => (
                      <Tag color={STATUS_COLOR[s] ?? 'grey'} type='light'>
                        {t(STATUS_LABELS[s] ?? 'Unknown')}
                      </Tag>
                    ),
                  },
                  {
                    title: t('请求数'),
                    dataIndex: 'request_count',
                    align: 'right',
                    render: (v) => (v ?? 0).toLocaleString(),
                  },
                  {
                    title: t('错误数'),
                    dataIndex: 'error_count',
                    align: 'right',
                    render: (v) =>
                      v > 0 ? (
                        <Typography.Text type='danger'>
                          {v.toLocaleString()}
                        </Typography.Text>
                      ) : (
                        (v ?? 0).toLocaleString()
                      ),
                  },
                  {
                    title: t('测试延迟'),
                    dataIndex: 'response_time_ms',
                    align: 'right',
                    render: (v) => formatLatency(v),
                  },
                ]}
              />
            </Card>
          </div>
        ) : null}
      </Modal>
    </div>
  );
};

const GroupCard = ({ snapshot, onClick, t }) => {
  const sTone = rateTone(snapshot.success_rate);
  const oTone = onlineTone(snapshot.channel_online, snapshot.channel_total);
  const overall = sTone === 'grey' ? oTone : sTone;
  return (
    <Card
      shadows='hover'
      style={{ cursor: 'pointer' }}
      onClick={onClick}
      bodyStyle={{ padding: 16 }}
    >
      <div className='mb-3 flex items-center gap-2'>
        <Tag color={overall} type='solid' size='small'>
          ●
        </Tag>
        <span className='truncate text-base font-semibold'>
          {snapshot.group}
        </span>
      </div>
      <div className='grid grid-cols-2 gap-2'>
        <MiniStat
          label={t('成功率')}
          value={formatPercent(snapshot.success_rate)}
          tone={sTone}
        />
        <MiniStat
          label={t('在线率')}
          value={
            snapshot.channel_total > 0
              ? `${snapshot.channel_online}/${snapshot.channel_total}`
              : '—'
          }
          tone={oTone}
        />
        <MiniStat
          label={t('请求数')}
          value={(snapshot.request_count ?? 0).toLocaleString()}
        />
        <MiniStat
          label={t('平均延迟')}
          value={formatLatency(snapshot.avg_latency_ms)}
        />
      </div>
    </Card>
  );
};

const TONE_TEXT_CLASS = {
  green: 'text-green-600',
  orange: 'text-orange-500',
  red: 'text-red-500',
  grey: 'text-gray-500',
};

const Metric = ({ icon, label, value, tone }) => (
  <Card bordered className='!shadow-none' bodyStyle={{ padding: 12 }}>
    <div className='flex items-center gap-1.5 text-xs text-gray-500'>
      {icon}
      {label}
    </div>
    <div
      className={`mt-1 font-mono text-xl font-semibold ${
        tone ? TONE_TEXT_CLASS[tone] : ''
      }`}
    >
      {value}
    </div>
  </Card>
);

const MiniStat = ({ label, value, tone }) => (
  <div className='rounded border border-gray-200 px-2.5 py-2'>
    <div className='text-[11px] text-gray-500'>{label}</div>
    <div
      className={`mt-1 font-mono text-sm font-semibold ${
        tone ? TONE_TEXT_CLASS[tone] : ''
      }`}
    >
      {value}
    </div>
  </div>
);

export default Availability;
