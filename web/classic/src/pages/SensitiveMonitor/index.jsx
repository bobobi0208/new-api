import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Modal,
  Space,
  Table,
  Tag,
  Tabs,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDownload,
  IconPlus,
  IconRefresh,
  IconSearch,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const ACTION_LABEL = {
  1: { text: '拦截并禁用 Token', color: 'red' },
  2: { text: '仅记录', color: 'blue' },
};

const EMPTY_RULE_FORM = {
  pattern: '',
  is_regex: false,
  enabled: true,
  action: 2,
  category: '',
  severity: 3,
  description: '',
};

function toRuleFormValues(rule) {
  if (!rule) {
    return EMPTY_RULE_FORM;
  }
  return {
    pattern: rule.pattern || '',
    is_regex: !!rule.is_regex,
    enabled: !!rule.enabled,
    action: rule.action || 2,
    category: rule.category || '',
    severity: rule.severity || 3,
    description: rule.description || '',
  };
}

function actionTag(action) {
  const item = ACTION_LABEL[action] || ACTION_LABEL[2];
  return <Tag color={item.color}>{item.text}</Tag>;
}

function formatTime(value) {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

function normalizeText(value) {
  const normalized = `${value || ''}`.trim();
  return normalized || undefined;
}

function normalizeTime(value) {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toISOString();
}

function buildHitFilterParams(values = {}) {
  const params = {
    action: values.action || undefined,
    keyword: normalizeText(values.keyword),
    username: normalizeText(values.username),
    token_name: normalizeText(values.token_name),
    model_name: normalizeText(values.model_name),
    request_id: normalizeText(values.request_id),
    path: normalizeText(values.path),
    false_positive:
      values.false_positive === 'true' || values.false_positive === 'false'
        ? values.false_positive
        : undefined,
  };
  if (Array.isArray(values.dateRange) && values.dateRange.length === 2) {
    params.start_time = normalizeTime(values.dateRange[0]);
    params.end_time = normalizeTime(values.dateRange[1]);
  }
  return params;
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export default function SensitiveMonitor() {
  const [rules, setRules] = useState([]);
  const [hits, setHits] = useState([]);
  const [ruleTotal, setRuleTotal] = useState(0);
  const [hitTotal, setHitTotal] = useState(0);
  const [rulePage, setRulePage] = useState(1);
  const [hitPage, setHitPage] = useState(1);
  const [loadingRules, setLoadingRules] = useState(false);
  const [loadingHits, setLoadingHits] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState(null);
  const [formApi, setFormApi] = useState(null);
  const [hitFilterFormApi, setHitFilterFormApi] = useState(null);
  const [hitFilters, setHitFilters] = useState({});
  const pageSize = 20;

  const fetchRules = async (page = rulePage) => {
    setLoadingRules(true);
    try {
      const res = await API.get('/api/sensitive_monitor/rules', {
        params: { p: page, page_size: pageSize },
      });
      const payload = res.data?.data || {};
      setRules(payload.items || []);
      setRuleTotal(payload.total || 0);
      setRulePage(page);
    } catch (error) {
      showError(error);
    } finally {
      setLoadingRules(false);
    }
  };

  const fetchHits = async (page = hitPage, filters = hitFilters) => {
    setLoadingHits(true);
    try {
      const res = await API.get('/api/sensitive_monitor/hits', {
        params: { p: page, page_size: pageSize, ...buildHitFilterParams(filters) },
      });
      const payload = res.data?.data || {};
      setHits(payload.items || []);
      setHitTotal(payload.total || 0);
      setHitPage(page);
    } catch (error) {
      showError(error);
    } finally {
      setLoadingHits(false);
    }
  };

  const searchHits = () => {
    const values = hitFilterFormApi?.getValues() || {};
    setHitFilters(values);
    fetchHits(1, values);
  };

  const clearHitFilters = () => {
    hitFilterFormApi?.setValues({});
    setHitFilters({});
    fetchHits(1, {});
  };

  const exportHits = async () => {
    const values = hitFilterFormApi?.getValues() || hitFilters;
    const params = buildHitFilterParams(values);
    setHitFilters(values);
    try {
      const res = await API.get('/api/sensitive_monitor/hits/export', {
        params,
        responseType: 'blob',
      });
      downloadBlob(
        res.data,
        `sensitive-monitor-hits-${new Date()
          .toISOString()
          .slice(0, 19)
          .replace(/[:T]/g, '-')}.csv`,
      );
      showSuccess('命中明细已导出');
    } catch (error) {
      showError(error);
    }
  };

  useEffect(() => {
    fetchRules(1);
    fetchHits(1);
  }, []);

  useEffect(() => {
    if (!modalVisible || !formApi) {
      return;
    }
    formApi.setValues(toRuleFormValues(editingRule));
  }, [modalVisible, formApi, editingRule]);

  const openCreateModal = () => {
    setEditingRule(null);
    setModalVisible(true);
  };

  const openEditModal = (rule) => {
    setEditingRule(rule);
    setModalVisible(true);
  };

  const saveRule = async () => {
    if (!formApi) {
      showError('表单未初始化');
      return;
    }
    const values = formApi.getValues();
    if (!values.pattern || !values.pattern.trim()) {
      showError('规则内容不能为空');
      return;
    }
    try {
      if (editingRule) {
        await API.put(`/api/sensitive_monitor/rules/${editingRule.id}`, values);
      } else {
        await API.post('/api/sensitive_monitor/rules', values);
      }
      showSuccess('保存成功');
      setModalVisible(false);
      fetchRules(rulePage);
    } catch (error) {
      showError(error);
    }
  };

  const seedDefaults = async () => {
    try {
      const res = await API.post('/api/sensitive_monitor/seed_defaults');
      const payload = res.data?.data || {};
      showSuccess(
        `已导入 ${payload.created || 0} 条默认规则，更新 ${
          payload.updated || 0
        } 条`,
      );
      fetchRules(1);
    } catch (error) {
      showError(error);
    }
  };

  const reloadRules = async () => {
    try {
      await API.post('/api/sensitive_monitor/reload');
      showSuccess('规则缓存已重载');
      fetchRules(rulePage);
    } catch (error) {
      showError(error);
    }
  };

  const markFalsePositive = async (hitId, value) => {
    try {
      await API.post(`/api/sensitive_monitor/hits/${hitId}/false_positive`, {
        value,
      });
      showSuccess(value ? '已标记为误报' : '已撤销误报标记');
      fetchHits(hitPage, hitFilters);
    } catch (error) {
      showError(error);
    }
  };

  const ruleColumns = useMemo(
    () => [
      { title: 'ID', dataIndex: 'id', width: 72 },
      {
        title: '状态',
        dataIndex: 'enabled',
        width: 96,
        render: (enabled) =>
          enabled ? <Tag color='green'>启用</Tag> : <Tag>停用</Tag>,
      },
      {
        title: '动作',
        dataIndex: 'action',
        width: 150,
        render: actionTag,
      },
      {
        title: '规则',
        dataIndex: 'pattern',
        render: (pattern, record) => (
          <Space vertical align='start' spacing={4}>
            <Text code>{pattern}</Text>
            {record.is_regex ? (
              <Tag color='purple'>正则</Tag>
            ) : (
              <Tag>普通词</Tag>
            )}
          </Space>
        ),
      },
      { title: '说明', dataIndex: 'description' },
      { title: '命中数', dataIndex: 'hit_count', width: 96 },
      {
        title: '最近命中',
        dataIndex: 'last_hit_at',
        width: 180,
        render: formatTime,
      },
      {
        title: '操作',
        width: 96,
        render: (_, record) => (
          <Button size='small' onClick={() => openEditModal(record)}>
            编辑
          </Button>
        ),
      },
    ],
    [],
  );

  const hitColumns = useMemo(
    () => [
      {
        title: '时间',
        dataIndex: 'created_at',
        width: 180,
        render: formatTime,
      },
      { title: '动作', dataIndex: 'action', width: 150, render: actionTag },
      { title: '用户', dataIndex: 'username', width: 120 },
      { title: 'Token', dataIndex: 'token_name', width: 140 },
      { title: '模型', dataIndex: 'model_name', width: 140 },
      { title: '渠道', dataIndex: 'channel_id', width: 80 },
      {
        title: '命中规则',
        dataIndex: 'pattern',
        render: (pattern) => <Text code>{pattern}</Text>,
      },
      { title: '请求片段', dataIndex: 'prompt_snippet' },
      { title: 'Request ID', dataIndex: 'request_id', width: 180 },
      {
        title: '误报',
        dataIndex: 'false_positive',
        width: 80,
        render: (fp) =>
          fp ? <Tag color='orange'>误报</Tag> : <Tag color='green'>有效</Tag>,
      },
      {
        title: '操作',
        width: 120,
        render: (_, record) => (
          <Button
            size='small'
            onClick={() => markFalsePositive(record.id, !record.false_positive)}
          >
            {record.false_positive ? '撤销误报' : '标记误报'}
          </Button>
        ),
      },
    ],
    [hitPage, hitFilters],
  );

  return (
    <div className='mt-[60px] px-2'>
      <Card>
        <div className='mb-4 flex items-center justify-between gap-3'>
          <div>
            <Typography.Title heading={4} style={{ margin: 0 }}>
              不良监控
            </Typography.Title>
            <Text type='secondary'>后置异步扫描请求文本，命中后记录明细。</Text>
          </div>
          <Space>
            <Button icon={<IconRefresh />} onClick={reloadRules}>
              重载缓存
            </Button>
            <Button onClick={seedDefaults}>导入默认规则</Button>
            <Button
              type='primary'
              icon={<IconPlus />}
              onClick={openCreateModal}
            >
              新增规则
            </Button>
          </Space>
        </div>

        <Tabs type='line'>
          <Tabs.TabPane tab='规则管理' itemKey='rules'>
            <Table
              rowKey='id'
              columns={ruleColumns}
              dataSource={rules}
              loading={loadingRules}
              pagination={{
                currentPage: rulePage,
                pageSize,
                total: ruleTotal,
                onPageChange: (page) => fetchRules(page),
              }}
            />
          </Tabs.TabPane>
          <Tabs.TabPane tab='命中明细' itemKey='hits'>
            <Form
              getFormApi={setHitFilterFormApi}
              onSubmit={searchHits}
              allowEmpty
              autoComplete='off'
              layout='vertical'
              className='mb-3'
            >
              <div className='grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-4'>
                <Form.Select
                  field='action'
                  placeholder='全部动作'
                  showClear
                  pure
                  size='small'
                >
                  <Form.Select.Option value={2}>仅记录</Form.Select.Option>
                  <Form.Select.Option value={1}>
                    拦截并禁用 Token
                  </Form.Select.Option>
                </Form.Select>
                <Form.Select
                  field='false_positive'
                  placeholder='误报状态'
                  showClear
                  pure
                  size='small'
                >
                  <Form.Select.Option value='false'>仅有效</Form.Select.Option>
                  <Form.Select.Option value='true'>仅误报</Form.Select.Option>
                </Form.Select>
                <Form.Input
                  field='keyword'
                  prefix={<IconSearch />}
                  placeholder='关键词'
                  showClear
                  pure
                  size='small'
                />
                <Form.Input
                  field='username'
                  prefix={<IconSearch />}
                  placeholder='用户'
                  showClear
                  pure
                  size='small'
                />
                <Form.Input
                  field='token_name'
                  prefix={<IconSearch />}
                  placeholder='Token'
                  showClear
                  pure
                  size='small'
                />
                <Form.Input
                  field='model_name'
                  prefix={<IconSearch />}
                  placeholder='模型'
                  showClear
                  pure
                  size='small'
                />
                <Form.Input
                  field='request_id'
                  prefix={<IconSearch />}
                  placeholder='Request ID'
                  showClear
                  pure
                  size='small'
                />
                <Form.Input
                  field='path'
                  prefix={<IconSearch />}
                  placeholder='路径'
                  showClear
                  pure
                  size='small'
                />
                <Form.DatePicker
                  field='dateRange'
                  className='w-full'
                  type='dateTimeRange'
                  placeholder={['开始时间', '结束时间']}
                  showClear
                  pure
                  size='small'
                />
              </div>
              <div className='mt-2 flex flex-wrap gap-2'>
                <Button
                  type='primary'
                  htmlType='submit'
                  icon={<IconSearch />}
                  size='small'
                >
                  筛选
                </Button>
                <Button onClick={clearHitFilters} size='small'>
                  清空
                </Button>
                <Button
                  icon={<IconDownload />}
                  onClick={exportHits}
                  size='small'
                >
                  导出 CSV
                </Button>
              </div>
            </Form>
            <Table
              rowKey='id'
              columns={hitColumns}
              dataSource={hits}
              loading={loadingHits}
              pagination={{
                currentPage: hitPage,
                pageSize,
                total: hitTotal,
                onPageChange: (page) => fetchHits(page, hitFilters),
              }}
            />
          </Tabs.TabPane>
        </Tabs>
      </Card>

      <Modal
        title={editingRule ? '编辑规则' : '新增规则'}
        visible={modalVisible}
        onOk={saveRule}
        onCancel={() => setModalVisible(false)}
        keepDOM
      >
        <Form getFormApi={setFormApi} labelPosition='top'>
          <Form.TextArea
            field='pattern'
            label='规则内容'
            autosize={{ minRows: 3 }}
          />
          <Form.Switch field='is_regex' label='正则规则' />
          <Form.Switch field='enabled' label='启用' initValue />
          <Form.Select field='action' label='命中动作' initValue={2}>
            <Form.Select.Option value={2}>仅记录</Form.Select.Option>
            <Form.Select.Option value={1}>拦截并禁用 Token</Form.Select.Option>
          </Form.Select>
          <Form.Input field='category' label='分类' placeholder='如 sexual_assault' />
          <Form.Select field='severity' label='严重程度' initValue={3}>
            <Form.Select.Option value={1}>低 (1)</Form.Select.Option>
            <Form.Select.Option value={2}>中低 (2)</Form.Select.Option>
            <Form.Select.Option value={3}>中 (3)</Form.Select.Option>
            <Form.Select.Option value={4}>高 (4)</Form.Select.Option>
            <Form.Select.Option value={5}>极高 (5)</Form.Select.Option>
          </Form.Select>
          <Form.TextArea
            field='description'
            label='说明'
            autosize={{ minRows: 2 }}
          />
        </Form>
      </Modal>
    </div>
  );
}
