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

import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Modal,
  Table,
  Button,
  Select,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

const WINDOW_OPTIONS = [5, 15, 60];

const ChannelLoadOverviewModal = ({ t, visible, onClose }) => {
  const [loading, setLoading] = useState(false);
  const [windowMinutes, setWindowMinutes] = useState(5);
  const [rows, setRows] = useState([]);
  const [errorLogEnabled, setErrorLogEnabled] = useState(true);
  const seqRef = useRef(0);

  const load = useCallback(() => {
    const seq = ++seqRef.current;
    setLoading(true);
    API.get(`/api/channel/load_overview?window_minutes=${windowMinutes}`)
      .then((res) => {
        if (seq !== seqRef.current) return;
        const { success, message, data } = res.data;
        if (success && data) {
          setRows(data.channels || []);
          setErrorLogEnabled(!!data.error_log_enabled);
        } else {
          showError(message);
        }
      })
      .catch((err) => {
        if (seq !== seqRef.current) return;
        showError(err.message);
      })
      .finally(() => {
        if (seq !== seqRef.current) return;
        setLoading(false);
      });
  }, [windowMinutes]);

  useEffect(() => {
    if (!visible) return;
    load();
  }, [visible, load]);

  const recover = async (channelId) => {
    try {
      const res = await API.post(`/api/channel/${channelId}/affinity/recover`);
      const { success, message, data } = res.data;
      if (success) {
        const deleted = data?.deleted ?? 0;
        showSuccess(
          t('已恢复渠道亲和（清除 ${count} 条计数）').replace(
            '${count}',
            deleted,
          ),
        );
        load();
      } else {
        showError(message);
      }
    } catch (err) {
      showError(err.message);
    }
  };

  const columns = [
    {
      title: t('渠道'),
      dataIndex: 'channel_name',
      render: (name, record) => (
        <span>
          <Text strong>{name}</Text>
          <Text type='tertiary' size='small' style={{ marginLeft: 4 }}>
            #{record.channel_id}
          </Text>
        </span>
      ),
    },
    {
      title: t('在途并发'),
      dataIndex: 'inflight',
      align: 'right',
      sorter: (a, b) => a.inflight - b.inflight,
      defaultSortOrder: 'descend',
    },
    {
      title: t('RPM'),
      dataIndex: 'rpm',
      align: 'right',
      sorter: (a, b) => a.rpm - b.rpm,
      render: (v) => Number(v || 0).toFixed(1),
    },
    {
      title: t('错误率'),
      dataIndex: 'error_rate',
      align: 'right',
      sorter: (a, b) => a.error_rate - b.error_rate,
      render: (v) =>
        errorLogEnabled ? `${(Number(v || 0) * 100).toFixed(1)}%` : '-',
    },
    {
      title: t('失败条目'),
      dataIndex: 'failover_entries',
      align: 'right',
      sorter: (a, b) => a.failover_entries - b.failover_entries,
    },
    {
      title: t('操作'),
      align: 'right',
      render: (_text, record) => (
        <Button
          size='small'
          type='tertiary'
          disabled={record.failover_entries === 0}
          onClick={() => recover(record.channel_id)}
        >
          {t('恢复')}
        </Button>
      ),
    },
  ];

  return (
    <Modal
      title={t('渠道负载总览')}
      visible={visible}
      onCancel={onClose}
      footer={null}
      width={760}
    >
      <Space style={{ marginBottom: 12 }}>
        <Text type='tertiary'>{t('时间窗口（分钟）')}</Text>
        <Select
          value={windowMinutes}
          onChange={(v) => setWindowMinutes(v)}
          style={{ width: 100 }}
        >
          {WINDOW_OPTIONS.map((w) => (
            <Select.Option key={w} value={w}>
              {w}
            </Select.Option>
          ))}
        </Select>
        <Button size='small' onClick={load} loading={loading}>
          {t('刷新')}
        </Button>
      </Space>
      {!errorLogEnabled && (
        <div style={{ marginBottom: 8 }}>
          <Text type='tertiary' size='small'>
            {t('错误日志未开启，错误率不可用（显示为 -）。')}
          </Text>
        </div>
      )}
      <Table
        columns={columns}
        dataSource={rows}
        loading={loading}
        rowKey='channel_id'
        pagination={false}
        size='small'
        scroll={{ y: 420 }}
      />
    </Modal>
  );
};

export default ChannelLoadOverviewModal;
