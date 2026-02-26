import React, { useState, useEffect } from 'react';
import {
  Table,
  Tag,
  Typography,
  Card,
  Select,
  Space,
  Button,
  message,
  Drawer,
  Tooltip,
} from 'antd';
import {
  ReloadOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { TaskExecution } from '../../types';
import { taskService } from '../../services/task';

const { Title, Text } = Typography;
const { Option } = Select;

const ExecutionsPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [executions, setExecutions] = useState<TaskExecution[]>([]);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState<string>('');
  // 数据预览
  const [dataDrawerVisible, setDataDrawerVisible] = useState(false);
  const [dataDrawerTitle, setDataDrawerTitle] = useState('');
  const [previewData, setPreviewData] = useState<Record<string, any>[]>([]);
  const [previewColumns, setPreviewColumns] = useState<string[]>([]);
  const [previewTotal, setPreviewTotal] = useState(0);
  const [previewPage, setPreviewPage] = useState(1);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewTaskId, setPreviewTaskId] = useState<number>(0);

  useEffect(() => {
    loadExecutions();
  }, [currentPage, pageSize, statusFilter]);

  const loadExecutions = async () => {
    setLoading(true);
    try {
      const response = await taskService.getExecutions({
        page: currentPage,
        limit: pageSize,
        status: statusFilter || undefined,
      });
      setExecutions(response.items || []);
      setTotal(response.pagination?.total || 0);
    } catch (error) {
      message.error('加载执行历史失败');
    } finally {
      setLoading(false);
    }
  };

  const handleStatusFilter = (value: string) => {
    setStatusFilter(value);
    setCurrentPage(1);
  };

  const handlePreviewData = async (taskId: number, taskName: string, page = 1) => {
    setPreviewTaskId(taskId);
    setDataDrawerTitle(taskName);
    setDataDrawerVisible(true);
    setPreviewPage(page);
    setPreviewLoading(true);
    try {
      const result = await taskService.getTaskData(taskId, { page, limit: 10 });
      setPreviewData(result.items || []);
      setPreviewColumns(result.columns || []);
      setPreviewTotal(result.pagination?.total || 0);
    } catch (error) {
      setPreviewData([]);
      setPreviewColumns([]);
      setPreviewTotal(0);
    } finally {
      setPreviewLoading(false);
    }
  };

  const getStatusTag = (status: string) => {
    const statusMap = {
      running: { color: 'blue', text: '运行中', icon: <ClockCircleOutlined /> },
      success: { color: 'green', text: '成功', icon: <CheckCircleOutlined /> },
      failed: { color: 'red', text: '失败', icon: <ExclamationCircleOutlined /> },
    };
    const config = statusMap[status as keyof typeof statusMap] || { color: 'default', text: status };
    return (
      <Tag color={config.color} icon={config.icon}>
        {config.text}
      </Tag>
    );
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '任务名称',
      dataIndex: 'task_name',
      key: 'task_name',
      render: (text: string) => text ? <Text strong>{text}</Text> : <Text type="secondary">-</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: getStatusTag,
    },
    {
      title: 'Worker Pod',
      dataIndex: 'worker_pod',
      key: 'worker_pod',
      render: (text: string) => text ? <Text code>{text}</Text> : '-',
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      render: (text: string) => text ? new Date(text).toLocaleString() : '-',
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      key: 'end_time',
      render: (text: string) => text ? new Date(text).toLocaleString() : '-',
    },
    {
      title: '采集记录数',
      dataIndex: 'records_collected',
      key: 'records_collected',
      width: 110,
      render: (num: number) => (num || 0).toLocaleString(),
    },
    {
      title: '重试次数',
      dataIndex: 'retry_count',
      key: 'retry_count',
      width: 90,
    },
    {
      title: '错误信息',
      dataIndex: 'error_message',
      key: 'error_message',
      ellipsis: true,
      render: (text: string) => text ? <Text type="danger">{text}</Text> : '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 80,
      render: (_: any, record: TaskExecution) => (
        <Tooltip title="查看数据">
          <Button
            type="text"
            icon={<EyeOutlined />}
            disabled={record.status !== 'success'}
            onClick={() => handlePreviewData(record.task_id, record.task_name || '', 1)}
            style={{ color: record.status === 'success' ? '#1890ff' : undefined }}
          />
        </Tooltip>
      ),
    },
  ];

  return (
    <div className="fade-in">
      {/* 页面标题 */}
      <div className="page-header">
        <Title level={3} className="page-title">执行历史</Title>
        <Text className="page-description">查看任务执行记录和结果</Text>
      </div>

      {/* 操作栏 */}
      <Card style={{ marginBottom: 16 }}>
        <Space size="middle" style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space size="middle">
            <Select
              defaultValue=""
              style={{ width: 120 }}
              onChange={handleStatusFilter}
            >
              <Option value="">全部</Option>
              <Option value="running">运行中</Option>
              <Option value="success">成功</Option>
              <Option value="failed">失败</Option>
            </Select>
          </Space>
          <Button icon={<ReloadOutlined />} onClick={loadExecutions}>
            刷新
          </Button>
        </Space>
      </Card>

      {/* 执行历史表格 */}
      <Card>
        <Table
          dataSource={executions}
          columns={columns}
          loading={loading}
          rowKey="id"
          pagination={{
            current: currentPage,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            onChange: (page, size) => {
              setCurrentPage(page);
              setPageSize(size || 20);
            },
          }}
        />
      </Card>

      {/* 数据预览抽屉 */}
      <Drawer
        title={`数据预览 - ${dataDrawerTitle}`}
        open={dataDrawerVisible}
        onClose={() => setDataDrawerVisible(false)}
        width={900}
      >
        {previewTotal === 0 && !previewLoading ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            暂无采集数据
          </div>
        ) : (
          <Table
            dataSource={previewData}
            loading={previewLoading}
            rowKey={(_, index) => String(index)}
            scroll={{ x: 'max-content' }}
            columns={previewColumns
              .filter(col => col !== 'id')
              .map(col => ({
                title: col,
                dataIndex: col,
                key: col,
                ellipsis: col === 'content' ? { showTitle: false } : true,
                width: col === 'content' ? 300 : undefined,
                render: (text: any) => {
                  const str = text != null ? String(text) : '-';
                  if (str.length > 100) {
                    return <Tooltip title={str.slice(0, 500)}><span>{str.slice(0, 100)}...</span></Tooltip>;
                  }
                  return str;
                },
              }))}
            pagination={{
              current: previewPage,
              pageSize: 10,
              total: previewTotal,
              showTotal: (t) => `共 ${t} 条`,
              onChange: (page) => handlePreviewData(previewTaskId, dataDrawerTitle, page),
            }}
          />
        )}
      </Drawer>
    </div>
  );
};

export default ExecutionsPage;
