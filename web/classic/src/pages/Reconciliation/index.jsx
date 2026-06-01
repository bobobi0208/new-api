import React, { useEffect, useState } from 'react';
import {
  Button,
  Card,
  Form,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconRefresh,
  IconSetting,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const STATUS_TAG = {
  match: { color: 'green', text: 'match' },
  mismatch: { color: 'red', text: 'mismatch' },
  inconclusive: { color: 'grey', text: 'inconclusive' },
  error: { color: 'red', text: 'error' },
};

const RUN_TYPE_LABEL = {
  balance_snapshot: 'balance_snapshot',
  daily_diff: 'daily_diff',
  manual: 'manual',
};

const UPSTREAM_OPTIONS = [
  { label: 'new-api', value: 'newapi' },
  { label: 'sub2api', value: 'sub2api' },
];

function formatTs(ts) {
  if (!ts || ts <= 0) return '-';
  return new Date(ts * 1000).toLocaleString();
}

function formatUSD(v) {
  if (v === undefined || v === null || Number.isNaN(v)) return '-';
  return '$' + Number(v).toFixed(6);
}

const DEFAULT_SETTING = {
  enabled: true,
  balance_poll_interval_sec: 600,
  daily_job_hour: 2,
  abs_threshold_usd: 0.01,
  rel_threshold: 0.05,
  http_timeout_sec: 15,
  retain_days: 90,
};

function statusTag(status) {
  const meta = STATUS_TAG[status] || { color: 'white', text: status };
  return <Tag color={meta.color}>{meta.text}</Tag>;
}

function ReconciliationPage() {
  const [tab, setTab] = useState('latest');
  const [latest, setLatest] = useState([]);
  const [records, setRecords] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(false);

  const [configs, setConfigs] = useState([]);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [configForm, setConfigForm] = useState({
    channel_id: 0,
    upstream_type: 'newapi',
    enabled: true,
    base_url: '',
    note: '',
  });

  const [settingsModalOpen, setSettingsModalOpen] = useState(false);
  const [settings, setSettings] = useState(DEFAULT_SETTING);

  const [detail, setDetail] = useState(null);

  const fetchLatest = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/reconciliation/latest');
      const { success, message, data } = res.data;
      if (success) setLatest(data || []);
      else showError(message);
    } finally {
      setLoading(false);
    }
  };

  const fetchAlerts = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/reconciliation/alerts');
      const { success, message, data } = res.data;
      if (success) setAlerts(data || []);
      else showError(message);
    } finally {
      setLoading(false);
    }
  };

  const fetchRecords = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/reconciliation/records', {
        params: { p: 1, page_size: 100 },
      });
      const { success, message, data } = res.data;
      if (success) setRecords(data?.items || []);
      else showError(message);
    } finally {
      setLoading(false);
    }
  };

  const fetchConfigs = async () => {
    const res = await API.get('/api/reconciliation/channels');
    const { success, message, data } = res.data;
    if (success) setConfigs(data || []);
    else showError(message);
  };

  const fetchSettings = async () => {
    const res = await API.get('/api/reconciliation/settings');
    const { success, message, data } = res.data;
    if (success) setSettings({ ...DEFAULT_SETTING, ...data });
    else showError(message);
  };

  const refresh = () => {
    if (tab === 'latest') fetchLatest();
    else if (tab === 'alerts') fetchAlerts();
    else if (tab === 'records') fetchRecords();
  };

  useEffect(() => {
    refresh();
  }, [tab]);

  const triggerOne = async (channel_id) => {
    try {
      const res = await API.post('/api/reconciliation/trigger', { channel_id });
      const { success, message } = res.data;
      if (success) {
        showSuccess('已触发');
        refresh();
      } else showError(message);
    } catch (e) {
      showError(e?.message || '触发失败');
    }
  };

  const saveConfig = async () => {
    if (!configForm.channel_id || configForm.channel_id <= 0) {
      showError('channel_id 必填');
      return;
    }
    const res = await API.post('/api/reconciliation/channels', configForm);
    const { success, message } = res.data;
    if (success) {
      showSuccess('已保存');
      setConfigForm({ channel_id: 0, upstream_type: 'newapi', enabled: true, base_url: '', note: '' });
      fetchConfigs();
      refresh();
    } else {
      showError(message);
    }
  };

  const deleteConfig = async (channel_id) => {
    const res = await API.delete('/api/reconciliation/channels/' + channel_id);
    const { success, message } = res.data;
    if (success) {
      showSuccess('已删除');
      fetchConfigs();
      refresh();
    } else {
      showError(message);
    }
  };

  const saveSettings = async () => {
    const res = await API.put('/api/reconciliation/settings', settings);
    const { success, message } = res.data;
    if (success) {
      showSuccess('已保存');
      setSettingsModalOpen(false);
    } else {
      showError(message);
    }
  };

  const recordColumns = [
    { title: '渠道', dataIndex: 'channel_id', key: 'channel_id' },
    { title: '渠道名', dataIndex: 'channel_name', key: 'channel_name', render: (v) => v || '-' },
    {
      title: '上游',
      dataIndex: 'upstream_type',
      key: 'upstream_type',
      render: (v) => <Tag>{v}</Tag>,
    },
    {
      title: '运行类型',
      dataIndex: 'run_type',
      key: 'run_type',
      render: (v) => RUN_TYPE_LABEL[v] || v,
    },
    { title: '状态', dataIndex: 'status', key: 'status', render: statusTag },
    { title: '上游已用 USD', dataIndex: 'upstream_used_usd', key: 'upstream_used_usd', render: formatUSD },
    { title: '本地已用 USD', dataIndex: 'local_used_usd', key: 'local_used_usd', render: formatUSD },
    {
      title: '差额',
      dataIndex: 'delta_usd',
      key: 'delta_usd',
      render: (v) => (
        <span style={Math.abs(v) > 0.0001 ? { color: '#e53935' } : { color: '#999' }}>
          {formatUSD(v)}
        </span>
      ),
    },
    { title: '运行时间', dataIndex: 'run_at', key: 'run_at', render: formatTs },
    {
      title: '操作',
      dataIndex: 'id',
      key: 'op',
      render: (_, row) => (
        <Space>
          <Button size='small' onClick={() => setDetail(row)}>详情</Button>
          <Button size='small' onClick={() => triggerOne(row.channel_id)}>触发</Button>
        </Space>
      ),
    },
  ];

  const dataset = tab === 'latest' ? latest : tab === 'alerts' ? alerts : records;

  return (
    <>
      <Card
        title='上游对账'
        headerExtraContent={
          <Space>
            <Button
              icon={<IconPlus />}
              onClick={() => {
                fetchConfigs();
                setConfigModalOpen(true);
              }}
            >
              渠道配置
            </Button>
            <Button
              icon={<IconSetting />}
              onClick={() => {
                fetchSettings();
                setSettingsModalOpen(true);
              }}
            >
              设置
            </Button>
            <Button icon={<IconRefresh />} onClick={refresh}>刷新</Button>
          </Space>
        }
      >
        <Tabs activeKey={tab} onChange={setTab}>
          <Tabs.TabPane tab='各渠道最新' itemKey='latest' />
          <Tabs.TabPane
            tab={
              <span>
                告警{alerts.length > 0 ? ` (${alerts.length})` : ''}
              </span>
            }
            itemKey='alerts'
          />
          <Tabs.TabPane tab='全部记录' itemKey='records' />
        </Tabs>

        <Table
          columns={recordColumns}
          dataSource={dataset}
          loading={loading}
          pagination={false}
          rowKey='id'
          empty={tab === 'alerts' ? '暂无对账告警' : '暂无记录'}
        />
      </Card>

      <Modal
        title='渠道对账配置'
        visible={configModalOpen}
        onCancel={() => setConfigModalOpen(false)}
        footer={null}
        width={780}
      >
        <Space style={{ marginBottom: 12 }} align='end'>
          <div>
            <Text>渠道 ID</Text>
            <InputNumber
              value={configForm.channel_id}
              onChange={(v) => setConfigForm({ ...configForm, channel_id: Number(v) || 0 })}
              style={{ width: 100, display: 'block' }}
            />
          </div>
          <div>
            <Text>上游类型</Text>
            <Select
              value={configForm.upstream_type}
              onChange={(v) => setConfigForm({ ...configForm, upstream_type: v })}
              optionList={UPSTREAM_OPTIONS}
              style={{ width: 130, display: 'block' }}
            />
          </div>
          <div>
            <Text>Base URL（可选）</Text>
            <Form.Input
              field='base_url'
              noLabel
              placeholder='留空则使用渠道 base_url'
              value={configForm.base_url}
              onChange={(v) => setConfigForm({ ...configForm, base_url: v })}
              style={{ width: 260 }}
            />
          </div>
          <div>
            <Text>启用</Text>
            <Switch
              checked={configForm.enabled}
              onChange={(v) => setConfigForm({ ...configForm, enabled: v })}
            />
          </div>
          <Button type='primary' onClick={saveConfig}>保存</Button>
        </Space>

        <Table
          columns={[
            { title: '渠道 ID', dataIndex: 'channel_id' },
            { title: '上游', dataIndex: 'upstream_type', render: (v) => <Tag>{v}</Tag> },
            { title: 'Base URL', dataIndex: 'base_url', render: (v) => v || <Text type='tertiary'>channel default</Text> },
            { title: '启用', dataIndex: 'enabled', render: (v) => <Tag color={v ? 'green' : 'grey'}>{v ? '启用' : '禁用'}</Tag> },
            {
              title: '操作',
              dataIndex: 'channel_id',
              key: 'op',
              render: (id) => <Button size='small' type='danger' onClick={() => deleteConfig(id)}>删除</Button>,
            },
          ]}
          dataSource={configs}
          rowKey='id'
          pagination={false}
        />
      </Modal>

      <Modal
        title='对账设置'
        visible={settingsModalOpen}
        onCancel={() => setSettingsModalOpen(false)}
        onOk={saveSettings}
        okText='保存'
        cancelText='取消'
        width={460}
      >
        <Space vertical style={{ width: '100%' }} align='start'>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Text>启用对账任务</Text>
            <Switch checked={settings.enabled} onChange={(v) => setSettings({ ...settings, enabled: v })} />
          </div>
          <div>
            <Text>余额轮询间隔（秒）</Text>
            <InputNumber
              min={60}
              value={settings.balance_poll_interval_sec}
              onChange={(v) => setSettings({ ...settings, balance_poll_interval_sec: Number(v) || 60 })}
              style={{ width: 180 }}
            />
          </div>
          <div>
            <Text>日终任务小时（0-23）</Text>
            <InputNumber
              min={0}
              max={23}
              value={settings.daily_job_hour}
              onChange={(v) => setSettings({ ...settings, daily_job_hour: Number(v) || 0 })}
              style={{ width: 180 }}
            />
          </div>
          <div>
            <Text>绝对偏差阈值（USD）</Text>
            <InputNumber
              min={0}
              step={0.01}
              value={settings.abs_threshold_usd}
              onChange={(v) => setSettings({ ...settings, abs_threshold_usd: Number(v) || 0 })}
              style={{ width: 180 }}
            />
          </div>
          <div>
            <Text>相对偏差阈值（0-1）</Text>
            <InputNumber
              min={0}
              max={1}
              step={0.01}
              value={settings.rel_threshold}
              onChange={(v) => setSettings({ ...settings, rel_threshold: Number(v) || 0 })}
              style={{ width: 180 }}
            />
          </div>
          <div>
            <Text>HTTP 超时（秒）</Text>
            <InputNumber
              min={3}
              max={120}
              value={settings.http_timeout_sec}
              onChange={(v) => setSettings({ ...settings, http_timeout_sec: Number(v) || 15 })}
              style={{ width: 180 }}
            />
          </div>
          <div>
            <Text>历史记录保留天数</Text>
            <InputNumber
              min={0}
              value={settings.retain_days}
              onChange={(v) => setSettings({ ...settings, retain_days: Number(v) || 0 })}
              style={{ width: 180 }}
            />
          </div>
        </Space>
      </Modal>

      <Modal
        title='对账记录详情'
        visible={!!detail}
        onCancel={() => setDetail(null)}
        footer={null}
        width={680}
      >
        {detail && (
          <Space vertical style={{ width: '100%' }} align='start'>
            <Text><strong>渠道 ID：</strong>{detail.channel_id}</Text>
            <Text><strong>渠道名：</strong>{detail.channel_name || '-'}</Text>
            <Text><strong>上游：</strong>{detail.upstream_type}</Text>
            <Text><strong>运行类型：</strong>{detail.run_type}</Text>
            <Text><strong>状态：</strong>{statusTag(detail.status)}</Text>
            <Text><strong>上游已用 USD：</strong>{formatUSD(detail.upstream_used_usd)}</Text>
            <Text><strong>本地已用 USD：</strong>{formatUSD(detail.local_used_usd)}</Text>
            <Text><strong>差额 USD：</strong>{formatUSD(detail.delta_usd)}</Text>
            <Text><strong>说明：</strong>{detail.message || '-'}</Text>
            <Text><strong>原始响应：</strong></Text>
            <pre style={{ width: '100%', maxHeight: 280, overflow: 'auto', background: '#f5f5f5', padding: 8, fontSize: 12 }}>
              {detail.upstream_raw_json || '-'}
            </pre>
          </Space>
        )}
      </Modal>
    </>
  );
}

export default ReconciliationPage;
