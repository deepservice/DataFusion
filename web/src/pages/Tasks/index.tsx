import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Typography,
  Card,
  Input,
  Select,
  Modal,
  Form,
  message,
  Popconfirm,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { Task } from '@/types';
import { taskService } from '@/services/task';

const { Title, Text } = Typography;
const { Option } = Select;

const TasksPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    loadTasks();
  }, [currentPage, pageSize, searchText, statusFilter]);

  const loadTasks = async () => {
    setLoading(true);
    try {
      const response = await taskService.getTasks({
        page: currentPage,
        limit: pageSize,
        search: searchText || undefined,
        status: statusFilter || undefined,
      });
      
      setTasks(response.items || []);
      setTotal(response.pagination?.total || 0);
    } catch (error) {
      message.error('加载任务列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (value: string) => {
    setSearchText(value);
    setCurrentPage(1);
  };

  const handleStatusFilter = (value: string) => {
    setStatusFilter(value);
    setCurrentPage(1);
  };

  const handleCreateTask = () => {
    setEditingTask(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEditTask = (task: Task) => {
    setEditingTask(task);
    form.setFieldsValue(task);
    setModalVisible(true);
  };

  const handleDeleteTask = async (id: number) => {
    try {
      await taskService.deleteTask(id);
      message.success('任务删除成功');
      loadTasks();
    } catch (error) {
      message.error('任务删除失败');
    }
  };

  const handleRunTask = async (id: number) => {
    try {
      await taskService.runTask(id);
      message.success('任务启动成功');
      loadTasks();
    } catch (error) {
      message.error('任务启动失败');
    }
  };

  const handleStopTask = async (id: number) => {
    try {
      await taskService.stopTask(id);
      message.success('任务停止成功');
      loadTasks();
    } catch (error) {
      message.error('任务停止失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      
      if (editingTask) {
        await taskService.updateTask(editingTask.id, values);
        message.success('任务更新成功');
      } else {
        await taskService.createTask(values);
        message.success('任务创建成功');
      }
      
      setModalVisible(false);
      loadTasks();
    } catch (error) {
      message.error(editingTask ? '任务更新失败' : '任务创建失败');
    }
  };

  const getStatusTag = (status: string) => {
    const statusMap = {
      enabled: { color: 'green', text: '启用' },
      disabled: { color: 'red', text: '禁用' },
    };
    const config = statusMap[status as keyof typeof statusMap] || { color: 'default', text: status };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  const columns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
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
      render: getStatusTag,
    },
    {
      title: 'Cron表达式',
      dataIndex: 'cron',
      key: 'cron',
      render: (text: string) => <Text code>{text || '-'}</Text>,
    },
    {
      title: '下次运行',
      dataIndex: 'next_run_time',
      key: 'next_run_time',
      render: (text: string) => text ? new Date(text).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record: Task) => (
        <Space size="small">
          <Button
            type="text"
            icon={<PlayCircleOutlined />}
            onClick={() => handleRunTask(record.id)}
            title="运行任务"
          />
          <Button
            type="text"
            icon={<PauseCircleOutlined />}
            onClick={() => handleStopTask(record.id)}
            title="停止任务"
          />
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEditTask(record)}
            title="编辑任务"
          />
          <Popconfirm
            title="确定要删除这个任务吗？"
            onConfirm={() => handleDeleteTask(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              title="删除任务"
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="fade-in">
      {/* 页面标题 */}
      <div className="page-header">
        <Title level={3} className="page-title">任务管理</Title>
        <Text className="page-description">管理数据采集任务的创建、编辑和执行</Text>
      </div>

      {/* 操作栏 */}
      <Card style={{ marginBottom: 16 }}>
        <Space size="middle" style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space size="middle">
            <Input.Search
              placeholder="搜索任务名称或描述"
              allowClear
              style={{ width: 300 }}
              onSearch={handleSearch}
            />
            <Select
              placeholder="筛选状态"
              allowClear
              style={{ width: 120 }}
              onChange={handleStatusFilter}
            >
              <Option value="enabled">启用</Option>
              <Option value="disabled">禁用</Option>
            </Select>
          </Space>
          <Space>
            <Button icon={<ReloadOutlined />} onClick={loadTasks}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreateTask}>
              新建任务
            </Button>
          </Space>
        </Space>
      </Card>

      {/* 任务表格 */}
      <Card>
        <Table
          dataSource={tasks}
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
              setPageSize(size || 10);
            },
          }}
        />
      </Card>

      {/* 创建/编辑任务模态框 */}
      <Modal
        title={editingTask ? '编辑任务' : '新建任务'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="请输入任务名称" />
          </Form.Item>
          
          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="请输入任务描述" rows={3} />
          </Form.Item>
          
          <Form.Item
            name="type"
            label="任务类型"
            rules={[{ required: true, message: '请选择任务类型' }]}
          >
            <Select placeholder="请选择任务类型">
              <Option value="web-rpa">网页RPA</Option>
              <Option value="api">API采集</Option>
              <Option value="database">数据库同步</Option>
            </Select>
          </Form.Item>
          
          <Form.Item name="cron" label="Cron表达式">
            <Input placeholder="例如: 0 0 * * * (每小时执行一次)" />
          </Form.Item>
          
          <Form.Item name="status" label="状态" initialValue="enabled">
            <Select>
              <Option value="enabled">启用</Option>
              <Option value="disabled">禁用</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TasksPage;