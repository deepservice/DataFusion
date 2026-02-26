import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Row, Col, Card, Statistic, Table, Tag, Space, Button, Typography, Spin } from 'antd';
import {
  ScheduleOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { Task, TaskExecution } from '../../types';
import { taskService } from '../../services/task';

const { Title, Text } = Typography;

const DashboardPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [statsData, setStatsData] = useState({
    total_tasks: 0,
    enabled_tasks: 0,
    total_executions: 0,
    success_executions: 0,
    failed_executions: 0,
    running_executions: 0,
    total_records: 0,
  });
  const [recentTasks, setRecentTasks] = useState<Task[]>([]);
  const [recentExecutions, setRecentExecutions] = useState<TaskExecution[]>([]);

  useEffect(() => {
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    setLoading(true);
    try {
      const [statsResponse, tasksResponse, executionsResponse] = await Promise.all([
        taskService.getStats(),
        taskService.getTasks({ limit: 5 }),
        taskService.getExecutions({ limit: 10 }),
      ]);

      setStatsData(statsResponse);
      setRecentTasks(tasksResponse.items || []);
      setRecentExecutions(executionsResponse.items || []);
    } catch (error) {
      console.error('加载仪表板数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const successRate = statsData.total_executions > 0
    ? Math.round((statsData.success_executions / statsData.total_executions) * 100)
    : 0;

  // 任务状态标签
  const getTaskStatusTag = (status: string) => {
    const statusMap = {
      enabled: { color: 'green', text: '启用' },
      disabled: { color: 'red', text: '禁用' },
    };
    const config = statusMap[status as keyof typeof statusMap] || { color: 'default', text: status };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 执行状态标签
  const getExecutionStatusTag = (status: string) => {
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

  // 任务表格列配置
  const taskColumns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (text: string) => <Tag>{text}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: getTaskStatusTag,
    },
    {
      title: '下次运行',
      dataIndex: 'next_run_time',
      key: 'next_run_time',
      render: (text: string) => text ? new Date(text).toLocaleString() : '-',
    },
  ];

  // 执行历史表格列配置
  const executionColumns = [
    {
      title: '任务名称',
      dataIndex: 'task_name',
      key: 'task_name',
      render: (text: string) => text || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: getExecutionStatusTag,
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      render: (text: string) => text ? new Date(text).toLocaleString() : '-',
    },
    {
      title: '采集记录数',
      dataIndex: 'records_collected',
      key: 'records_collected',
      render: (num: number) => (num || 0).toLocaleString(),
    },
  ];

  if (loading) {
    return (
      <div className="loading-container">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="fade-in">
      {/* 页面标题 */}
      <div className="page-header">
        <Space size="middle" style={{ width: '100%', justifyContent: 'space-between' }}>
          <div>
            <Title level={3} className="page-title">仪表板</Title>
            <Text className="page-description">系统概览和快速操作</Text>
          </div>
          <Button icon={<ReloadOutlined />} onClick={loadDashboardData}>
            刷新
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总任务数"
              value={statsData.total_tasks}
              prefix={<ScheduleOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃任务"
              value={statsData.enabled_tasks}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总执行次数"
              value={statsData.total_executions}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="成功率"
              value={successRate}
              suffix="%"
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: successRate >= 90 ? '#52c41a' : successRate >= 70 ? '#faad14' : '#ff4d4f' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 内容区域 */}
      <Row gutter={[16, 16]}>
        {/* 最近任务 */}
        <Col xs={24} lg={12}>
          <Card title="最近任务" extra={<Button type="link" onClick={() => navigate('/tasks')}>查看全部</Button>}>
            <Table
              dataSource={recentTasks}
              columns={taskColumns}
              pagination={false}
              size="small"
              rowKey="id"
            />
          </Card>
        </Col>

        {/* 最近执行 */}
        <Col xs={24} lg={12}>
          <Card title="最近执行" extra={<Button type="link" onClick={() => navigate('/executions')}>查看全部</Button>}>
            <Table
              dataSource={recentExecutions}
              columns={executionColumns}
              pagination={false}
              size="small"
              rowKey="id"
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default DashboardPage;
