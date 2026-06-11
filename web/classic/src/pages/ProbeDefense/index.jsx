import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlus, IconRefresh, IconSearch } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;
const API_KEY_MASK = '__PROBE_DEFENSE_TARGET_API_KEY_CONFIGURED__';
const MATCH_TYPES = [
  { value: 'contains_all', label: '包含全部' },
  { value: 'contains_any', label: '包含任一' },
  { value: 'regex', label: '正则' },
];

const emptySource = { key: '', name: '', description: '', enabled: true };
const emptySignature = {
  name: '',
  source_key: '',
  protocol: 'claude_messages',
  topic: '',
  match_type: 'contains_all',
  patterns: '',
  enabled: true,
};

function parseArray(value) {
  if (Array.isArray(value)) return value;
  if (!value) return [];
  try {
    const parsed = JSON.parse(value);
    return Array.isArray(parsed) ? parsed : [];
  } catch (error) {
    return [];
  }
}

function normalizeLines(value) {
  return `${value || ''}`
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatTime(value) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

function formatPatterns(value) {
  const patterns = parseArray(value);
  if (patterns.length === 0) return '-';
  const preview = patterns.slice(0, 3).join(' / ');
  return patterns.length > 3 ? `${preview} ...` : preview;
}

function mapPoliciesByGroup(policies) {
  return (policies || []).reduce((acc, item) => {
    acc[item.group_name] = item;
    return acc;
  }, {});
}

export default function ProbeDefense() {
  const [groups, setGroups] = useState([]);
  const [sources, setSources] = useState([]);
  const [signatures, setSignatures] = useState([]);
  const [policies, setPolicies] = useState([]);
  const [events, setEvents] = useState([]);
  const [recentEvents, setRecentEvents] = useState([]);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventsPage, setEventsPage] = useState(1);
  const [selectedGroup, setSelectedGroup] = useState('');
  const [activeTab, setActiveTab] = useState('policies');
  const [policyForm, setPolicyForm] = useState({
    enabled: false,
    source_keys: [],
    target_url: '',
    target_api_key: '',
    log_only: false,
  });
  const [sourceModalVisible, setSourceModalVisible] = useState(false);
  const [signatureModalVisible, setSignatureModalVisible] = useState(false);
  const [editingSource, setEditingSource] = useState(null);
  const [editingSignature, setEditingSignature] = useState(null);
  const [sourceForm, setSourceForm] = useState(emptySource);
  const [signatureForm, setSignatureForm] = useState(emptySignature);
  const [eventFilters, setEventFilters] = useState({
    group: '',
    source_key: '',
    topic: '',
    action: '',
  });
  const [signatureSourceFilter, setSignatureSourceFilter] = useState([]);
  const [testText, setTestText] = useState('');
  const [testSources, setTestSources] = useState([]);
  const [testResult, setTestResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const pageSize = 20;

  const policyMap = useMemo(() => mapPoliciesByGroup(policies), [policies]);
  const sourceMap = useMemo(
    () =>
      sources.reduce((acc, item) => {
        acc[item.key] = item;
        return acc;
      }, {}),
    [sources],
  );
  const enabledSourceKeys = useMemo(
    () => sources.filter((item) => item.enabled).map((item) => item.key),
    [sources],
  );
  const enabledSignatureCountBySource = useMemo(
    () =>
      signatures.reduce((acc, item) => {
        const source = sourceMap[item.source_key];
        if (!item.enabled || !source?.enabled) return acc;
        acc[item.source_key] = (acc[item.source_key] || 0) + 1;
        return acc;
      }, {}),
    [signatures, sourceMap],
  );
  const latestEventByGroup = useMemo(
    () =>
      recentEvents.reduce((acc, item) => {
        if (item.group_name && !acc[item.group_name]) {
          acc[item.group_name] = item;
        }
        return acc;
      }, {}),
    [recentEvents],
  );
  const sourceOptions = useMemo(
    () =>
      sources.map((item) => ({
        value: item.key,
        label: item.name || item.key,
      })),
    [sources],
  );
  const groupOptions = useMemo(
    () => groups.map((item) => ({ value: item, label: item })),
    [groups],
  );
  const filteredSignatures = useMemo(() => {
    if (signatureSourceFilter.length === 0) return signatures;
    const sourceSet = new Set(signatureSourceFilter);
    return signatures.filter((item) => sourceSet.has(item.source_key));
  }, [signatures, signatureSourceFilter]);
  const policyOverviewRows = useMemo(
    () =>
      groups.map((group) => {
        const policy = policyMap[group];
        const sourceKeys = parseArray(policy?.source_keys);
        const effectiveSourceKeys =
          sourceKeys.length > 0 ? sourceKeys : enabledSourceKeys;
        const ruleCount = effectiveSourceKeys.reduce(
          (sum, key) => sum + (enabledSignatureCountBySource[key] || 0),
          0,
        );
        return {
          group_name: group,
          policy,
          enabled: !!policy?.enabled,
          source_keys: sourceKeys,
          effective_source_keys: effectiveSourceKeys,
          rule_count: ruleCount,
          target_url: policy?.target_url || '',
          has_api_key: policy?.target_api_key === API_KEY_MASK,
          log_only: !!policy?.log_only,
          latest_event: latestEventByGroup[group],
        };
      }),
    [
      groups,
      policyMap,
      enabledSourceKeys,
      enabledSignatureCountBySource,
      latestEventByGroup,
    ],
  );

  const syncPolicyForm = (group, list = policies) => {
    const policy = mapPoliciesByGroup(list)[group];
    setPolicyForm({
      enabled: !!policy?.enabled,
      source_keys: parseArray(policy?.source_keys),
      target_url: policy?.target_url || '',
      target_api_key: '',
      log_only: !!policy?.log_only,
    });
  };

  const fetchAll = async () => {
    setLoading(true);
    try {
      const [groupRes, sourceRes, signatureRes, policyRes, recentEventRes] =
        await Promise.all([
          API.get('/api/group/'),
          API.get('/api/probe_defense/sources'),
          API.get('/api/probe_defense/signatures'),
          API.get('/api/probe_defense/policies'),
          API.get('/api/probe_defense/events', {
            params: { p: 1, page_size: 200 },
          }),
        ]);
      const nextGroups = groupRes.data?.data || [];
      const nextSources = sourceRes.data?.data || [];
      const nextSignatures = signatureRes.data?.data || [];
      const nextPolicies = policyRes.data?.data || [];
      const nextRecentEvents = recentEventRes.data?.data?.items || [];
      setGroups(nextGroups);
      setSources(nextSources);
      setSignatures(nextSignatures);
      setPolicies(nextPolicies);
      setRecentEvents(nextRecentEvents);
      const group = selectedGroup || nextGroups[0] || '';
      setSelectedGroup(group);
      syncPolicyForm(group, nextPolicies);
    } catch (error) {
      showError(error);
    } finally {
      setLoading(false);
    }
  };

  const fetchEvents = async (page = eventsPage, filters = eventFilters) => {
    try {
      const res = await API.get('/api/probe_defense/events', {
        params: { p: page, page_size: pageSize, ...filters },
      });
      const payload = res.data?.data || {};
      setEvents(payload.items || []);
      setEventsTotal(payload.total || 0);
      setEventsPage(page);
    } catch (error) {
      showError(error);
    }
  };

  useEffect(() => {
    fetchAll();
    fetchEvents(1);
  }, []);

  const savePolicy = async () => {
    if (!selectedGroup) return;
    try {
      await API.put(
        `/api/probe_defense/policies/${encodeURIComponent(selectedGroup)}`,
        policyForm,
      );
      showSuccess('保存成功');
      await fetchAll();
    } catch (error) {
      showError(error);
    }
  };

  const openSourceModal = (source = null) => {
    setEditingSource(source);
    setSourceForm(source ? { ...source } : emptySource);
    setSourceModalVisible(true);
  };

  const saveSource = async () => {
    try {
      if (editingSource?.id) {
        await API.put(
          `/api/probe_defense/sources/${editingSource.id}`,
          sourceForm,
        );
      } else {
        await API.post('/api/probe_defense/sources', sourceForm);
      }
      showSuccess('保存成功');
      setSourceModalVisible(false);
      await fetchAll();
    } catch (error) {
      showError(error);
    }
  };

  const openSignatureModal = (signature = null) => {
    setEditingSignature(signature);
    setSignatureForm(
      signature
        ? { ...signature, patterns: parseArray(signature.patterns).join('\n') }
        : emptySignature,
    );
    setSignatureModalVisible(true);
  };

  const saveSignature = async () => {
    const payload = {
      ...signatureForm,
      patterns: normalizeLines(signatureForm.patterns),
    };
    try {
      if (editingSignature?.id) {
        await API.put(
          `/api/probe_defense/signatures/${editingSignature.id}`,
          payload,
        );
      } else {
        await API.post('/api/probe_defense/signatures', payload);
      }
      showSuccess('保存成功');
      setSignatureModalVisible(false);
      await fetchAll();
    } catch (error) {
      showError(error);
    }
  };

  const deleteSignature = (signature) => {
    Modal.confirm({
      title: '删除规则',
      content: `确认删除 ${signature.name}？`,
      onOk: async () => {
        try {
          await API.delete(`/api/probe_defense/signatures/${signature.id}`);
          showSuccess('删除成功');
          await fetchAll();
        } catch (error) {
          showError(error);
        }
      },
    });
  };

  const runTestMatch = async () => {
    try {
      const res = await API.post('/api/probe_defense/test_match', {
        protocol: 'claude_messages',
        text: testText,
        source_keys: testSources,
      });
      setTestResult(res.data?.data || null);
    } catch (error) {
      showError(error);
    }
  };

  const sourceColumns = [
    { title: '来源 Key', dataIndex: 'key' },
    { title: '名称', dataIndex: 'name' },
    { title: '说明', dataIndex: 'description' },
    {
      title: '状态',
      render: (_, record) => (
        <Tag color={record.enabled ? 'green' : 'grey'}>
          {record.enabled ? '启用' : '停用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      render: (_, record) => (
        <Button size='small' onClick={() => openSourceModal(record)}>
          编辑
        </Button>
      ),
    },
  ];

  const signatureColumns = [
    { title: '名称', dataIndex: 'name' },
    { title: '来源', dataIndex: 'source_key' },
    { title: '协议', dataIndex: 'protocol' },
    { title: '主题', dataIndex: 'topic' },
    { title: '匹配方式', dataIndex: 'match_type' },
    {
      title: '特征摘要',
      dataIndex: 'patterns',
      render: formatPatterns,
    },
    {
      title: '状态',
      render: (_, record) => (
        <Tag color={record.enabled ? 'green' : 'grey'}>
          {record.enabled ? '启用' : '停用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      render: (_, record) => (
        <Space>
          <Button size='small' onClick={() => openSignatureModal(record)}>
            编辑
          </Button>
          <Button
            size='small'
            type='danger'
            onClick={() => deleteSignature(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const policyOverviewColumns = [
    { title: '分组', dataIndex: 'group_name' },
    {
      title: '状态',
      render: (_, record) => (
        <Tag color={record.enabled ? 'green' : 'grey'}>
          {record.enabled ? '启用' : '未启用'}
        </Tag>
      ),
    },
    {
      title: '动作模式',
      render: (_, record) => {
        if (!record.enabled) return <Tag color='grey'>未启用</Tag>;
        return (
          <Tag color={record.log_only ? 'blue' : 'orange'}>
            {record.log_only ? '仅记录' : '转移'}
          </Tag>
        );
      },
    },
    {
      title: '防御来源',
      render: (_, record) => {
        if (record.source_keys.length === 0) {
          return <Tag color='cyan'>全部启用来源</Tag>;
        }
        const visibleKeys = record.source_keys.slice(0, 3);
        return (
          <Space wrap>
            {visibleKeys.map((key) => (
              <Tag key={key}>{sourceMap[key]?.name || key}</Tag>
            ))}
            {record.source_keys.length > visibleKeys.length && (
              <Tag>+{record.source_keys.length - visibleKeys.length}</Tag>
            )}
          </Space>
        );
      },
    },
    { title: '规则数量', dataIndex: 'rule_count' },
    {
      title: '目标 URL',
      render: (_, record) => {
        if (record.log_only) return '-';
        return record.target_url || <Text type='danger'>未配置</Text>;
      },
    },
    {
      title: 'API Key',
      render: (_, record) => {
        if (record.log_only) return '-';
        return (
          <Tag color={record.has_api_key ? 'green' : 'red'}>
            {record.has_api_key ? '已配置' : '未配置'}
          </Tag>
        );
      },
    },
    {
      title: '最近命中',
      render: (_, record) => {
        const event = record.latest_event;
        if (!event) return '-';
        return (
          <div>
            <div>{formatTime(event.created_at)}</div>
            <Text type='tertiary'>
              {event.source_key || '-'} / {event.topic || '-'}
            </Text>
          </div>
        );
      },
    },
    {
      title: '操作',
      render: (_, record) => (
        <Space>
          <Button
            size='small'
            onClick={() => {
              setSelectedGroup(record.group_name);
              syncPolicyForm(record.group_name);
            }}
          >
            编辑
          </Button>
          <Button
            size='small'
            onClick={() => {
              setSignatureSourceFilter(record.effective_source_keys);
              setActiveTab('signatures');
            }}
          >
            查看规则
          </Button>
        </Space>
      ),
    },
  ];

  const eventColumns = [
    { title: '时间', dataIndex: 'created_at', render: formatTime },
    { title: '分组', dataIndex: 'group_name' },
    { title: '渠道', dataIndex: 'channel_id' },
    { title: '来源', dataIndex: 'source_key' },
    { title: '主题', dataIndex: 'topic' },
    { title: '规则', dataIndex: 'signature_name' },
    { title: '动作', dataIndex: 'action' },
    { title: '目标', dataIndex: 'target_url' },
    { title: '预览', dataIndex: 'content_preview' },
  ];

  return (
    <div className='p-4'>
      <Card>
        <Space vertical align='start' style={{ width: '100%' }}>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Typography.Title heading={4} style={{ margin: 0 }}>
              探针防御
            </Typography.Title>
            <Button icon={<IconRefresh />} loading={loading} onClick={fetchAll}>
              刷新配置
            </Button>
          </Space>
          <Tabs type='line' activeKey={activeTab} onChange={setActiveTab}>
            <Tabs.TabPane tab='分组策略' itemKey='policies'>
              <Space vertical align='start' style={{ width: '100%' }}>
                <Table
                  rowKey='group_name'
                  columns={policyOverviewColumns}
                  dataSource={policyOverviewRows}
                  pagination={false}
                />
                <Select
                  value={selectedGroup}
                  optionList={groupOptions}
                  style={{ width: 260 }}
                  onChange={(value) => {
                    setSelectedGroup(value);
                    syncPolicyForm(value);
                  }}
                />
                <Space wrap>
                  <Text>启用</Text>
                  <Switch
                    checked={policyForm.enabled}
                    onChange={(value) =>
                      setPolicyForm({ ...policyForm, enabled: value })
                    }
                  />
                  <Text>仅记录</Text>
                  <Switch
                    checked={policyForm.log_only}
                    onChange={(value) =>
                      setPolicyForm({ ...policyForm, log_only: value })
                    }
                  />
                </Space>
                <Select
                  multiple
                  value={policyForm.source_keys}
                  optionList={sourceOptions}
                  style={{ width: '100%', maxWidth: 520 }}
                  placeholder='选择要防御的来源'
                  onChange={(value) =>
                    setPolicyForm({ ...policyForm, source_keys: value })
                  }
                />
                <Input
                  value={policyForm.target_url}
                  placeholder='转移目标 URL，例如 https://example.com/v1/messages'
                  style={{ maxWidth: 720 }}
                  onChange={(value) =>
                    setPolicyForm({ ...policyForm, target_url: value })
                  }
                />
                <Input
                  mode='password'
                  value={policyForm.target_api_key}
                  placeholder={
                    policyMap[selectedGroup]?.target_api_key === API_KEY_MASK
                      ? '已配置，留空保持不变'
                      : '目标 API Key'
                  }
                  style={{ maxWidth: 720 }}
                  onChange={(value) =>
                    setPolicyForm({ ...policyForm, target_api_key: value })
                  }
                />
                <Button type='primary' onClick={savePolicy}>
                  保存分组策略
                </Button>
              </Space>
            </Tabs.TabPane>
            <Tabs.TabPane tab='探针来源' itemKey='sources'>
              <Space vertical align='start' style={{ width: '100%' }}>
                <Button
                  icon={<IconPlus />}
                  type='primary'
                  onClick={() => openSourceModal()}
                >
                  新增来源
                </Button>
                <Table
                  rowKey='id'
                  columns={sourceColumns}
                  dataSource={sources}
                  pagination={false}
                />
              </Space>
            </Tabs.TabPane>
            <Tabs.TabPane tab='特征规则' itemKey='signatures'>
              <Space vertical align='start' style={{ width: '100%' }}>
                <Space wrap>
                  <Select
                    multiple
                    value={signatureSourceFilter}
                    optionList={sourceOptions}
                    style={{ minWidth: 260 }}
                    placeholder='按来源过滤规则'
                    onChange={setSignatureSourceFilter}
                  />
                  <Button onClick={() => setSignatureSourceFilter([])}>
                    清除过滤
                  </Button>
                  <Button
                    icon={<IconPlus />}
                    type='primary'
                    onClick={() => openSignatureModal()}
                  >
                    新增规则
                  </Button>
                </Space>
                <Table
                  rowKey='id'
                  columns={signatureColumns}
                  dataSource={filteredSignatures}
                  pagination={false}
                />
              </Space>
            </Tabs.TabPane>
            <Tabs.TabPane tab='命中日志' itemKey='events'>
              <Space vertical align='start' style={{ width: '100%' }}>
                <Space wrap>
                  <Select
                    value={eventFilters.group}
                    optionList={[
                      { value: '', label: '全部分组' },
                      ...groupOptions,
                    ]}
                    style={{ width: 180 }}
                    onChange={(value) =>
                      setEventFilters({ ...eventFilters, group: value })
                    }
                  />
                  <Select
                    value={eventFilters.source_key}
                    optionList={[
                      { value: '', label: '全部来源' },
                      ...sourceOptions,
                    ]}
                    style={{ width: 180 }}
                    onChange={(value) =>
                      setEventFilters({ ...eventFilters, source_key: value })
                    }
                  />
                  <Input
                    value={eventFilters.topic}
                    placeholder='主题'
                    style={{ width: 180 }}
                    onChange={(value) =>
                      setEventFilters({ ...eventFilters, topic: value })
                    }
                  />
                  <Input
                    value={eventFilters.action}
                    placeholder='动作'
                    style={{ width: 180 }}
                    onChange={(value) =>
                      setEventFilters({ ...eventFilters, action: value })
                    }
                  />
                  <Button
                    icon={<IconSearch />}
                    onClick={() => fetchEvents(1, eventFilters)}
                  >
                    筛选
                  </Button>
                </Space>
                <Table
                  rowKey='id'
                  columns={eventColumns}
                  dataSource={events}
                  pagination={{
                    currentPage: eventsPage,
                    pageSize,
                    total: eventsTotal,
                    onPageChange: (page) => fetchEvents(page),
                  }}
                />
              </Space>
            </Tabs.TabPane>
            <Tabs.TabPane tab='测试匹配' itemKey='test'>
              <Space vertical align='start' style={{ width: '100%' }}>
                <Select
                  multiple
                  value={testSources}
                  optionList={sourceOptions}
                  style={{ width: '100%', maxWidth: 520 }}
                  placeholder='留空则测试全部启用来源'
                  onChange={setTestSources}
                />
                <TextArea
                  value={testText}
                  autosize={{ minRows: 6, maxRows: 12 }}
                  placeholder='粘贴待检测内容'
                  onChange={setTestText}
                />
                <Button type='primary' onClick={runTestMatch}>
                  测试匹配
                </Button>
                {testResult && (
                  <Card style={{ width: '100%' }}>
                    <Space wrap>
                      <Tag color={testResult.matched ? 'red' : 'green'}>
                        {testResult.matched ? '命中' : '未命中'}
                      </Tag>
                      {testResult.source_key && (
                        <Text>来源：{testResult.source_key}</Text>
                      )}
                      {testResult.topic && (
                        <Text>主题：{testResult.topic}</Text>
                      )}
                      {testResult.signature_name && (
                        <Text>规则：{testResult.signature_name}</Text>
                      )}
                      {testResult.match_type && (
                        <Text>方式：{testResult.match_type}</Text>
                      )}
                    </Space>
                  </Card>
                )}
              </Space>
            </Tabs.TabPane>
          </Tabs>
        </Space>
      </Card>

      <Modal
        title={editingSource ? '编辑来源' : '新增来源'}
        visible={sourceModalVisible}
        onOk={saveSource}
        onCancel={() => setSourceModalVisible(false)}
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          <Input
            value={sourceForm.key}
            placeholder='来源 Key，例如 cctest'
            onChange={(value) => setSourceForm({ ...sourceForm, key: value })}
          />
          <Input
            value={sourceForm.name}
            placeholder='来源名称'
            onChange={(value) => setSourceForm({ ...sourceForm, name: value })}
          />
          <TextArea
            value={sourceForm.description}
            placeholder='说明'
            onChange={(value) =>
              setSourceForm({ ...sourceForm, description: value })
            }
          />
          <Space>
            <Text>启用</Text>
            <Switch
              checked={sourceForm.enabled}
              onChange={(value) =>
                setSourceForm({ ...sourceForm, enabled: value })
              }
            />
          </Space>
        </Space>
      </Modal>

      <Modal
        title={editingSignature ? '编辑规则' : '新增规则'}
        visible={signatureModalVisible}
        onOk={saveSignature}
        onCancel={() => setSignatureModalVisible(false)}
        style={{ width: 720 }}
      >
        <Space vertical align='start' style={{ width: '100%' }}>
          <Input
            value={signatureForm.name}
            placeholder='规则名称'
            onChange={(value) =>
              setSignatureForm({ ...signatureForm, name: value })
            }
          />
          <Select
            value={signatureForm.source_key}
            optionList={sourceOptions}
            style={{ width: '100%' }}
            placeholder='来源'
            onChange={(value) =>
              setSignatureForm({ ...signatureForm, source_key: value })
            }
          />
          <Input
            value={signatureForm.protocol}
            placeholder='协议'
            onChange={(value) =>
              setSignatureForm({ ...signatureForm, protocol: value })
            }
          />
          <Input
            value={signatureForm.topic}
            placeholder='主题'
            onChange={(value) =>
              setSignatureForm({ ...signatureForm, topic: value })
            }
          />
          <Select
            value={signatureForm.match_type}
            optionList={MATCH_TYPES}
            style={{ width: '100%' }}
            onChange={(value) =>
              setSignatureForm({ ...signatureForm, match_type: value })
            }
          />
          <TextArea
            value={signatureForm.patterns}
            autosize={{ minRows: 5, maxRows: 10 }}
            placeholder='每行一个特征'
            onChange={(value) =>
              setSignatureForm({ ...signatureForm, patterns: value })
            }
          />
          <Space>
            <Text>启用</Text>
            <Switch
              checked={signatureForm.enabled}
              onChange={(value) =>
                setSignatureForm({ ...signatureForm, enabled: value })
              }
            />
          </Space>
        </Space>
      </Modal>
    </div>
  );
}
