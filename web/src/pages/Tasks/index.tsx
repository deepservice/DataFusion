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
  InputNumber,
  Drawer,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { Task } from '../../types';
import { taskService } from '../../services/task';
import { dataSourceService, DataSource } from '../../services/datasource';

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;

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
  const [dataSources, setDataSources] = useState<DataSource[]>([]);
  const [form] = Form.useForm();
  // 数据预览
  const [dataDrawerVisible, setDataDrawerVisible] = useState(false);
  const [dataDrawerTask, setDataDrawerTask] = useState<Task | null>(null);
  const [previewData, setPreviewData] = useState<Record<string, any>[]>([]);
  const [previewColumns, setPreviewColumns] = useState<string[]>([]);
  const [previewTotal, setPreviewTotal] = useState(0);
  const [previewPage, setPreviewPage] = useState(1);
  const [previewLoading, setPreviewLoading] = useState(false);

  useEffect(() => {
    loadTasks();
    loadDataSources();
  }, [currentPage, pageSize, searchText, statusFilter]);

  const loadDataSources = async () => {
    try {
      const response = await dataSourceService.list({ page: 1, page_size: 100 });
      setDataSources(response.data || []);
    } catch (error) {
      console.error('加载数据源列表失败', error);
    }
  };

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
    form.setFieldsValue({
      status: 'enabled',
      replicas: 1,
      execution_timeout: 3600,
      max_retries: 3,
    });
    setModalVisible(true);
  };

  const handleEditTask = (task: Task) => {
    setEditingTask(task);
    form.setFieldsValue({
      ...task,
      config: task.config ? JSON.stringify(task.config, null, 2) : undefined,
    });
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

  const handleExecuteTask = async (id: number) => {
    try {
      await taskService.executeTask(id);
      message.success('任务执行已触发');
      loadTasks();
    } catch (error) {
      message.error('任务执行失败');
    }
  };

  const handlePreviewData = async (task: Task, page = 1) => {
    setDataDrawerTask(task);
    setDataDrawerVisible(true);
    setPreviewPage(page);
    setPreviewLoading(true);
    try {
      const result = await taskService.getTaskData(task.id, { page, limit: 10 });
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

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();

      // 处理config字段，如果有值则验证JSON格式
      if (values.config) {
        try {
          JSON.parse(values.config);
        } catch (error) {
          message.error('任务配置必须是有效的JSON格式');
          return;
        }
      }

      // 将 config 字符串解析为 JSON 对象，空值则为 null
      let configValue = null;
      if (values.config && values.config.trim()) {
        configValue = JSON.parse(values.config);
      }

      const taskData = {
        ...values,
        config: configValue,
      };

      if (editingTask) {
        await taskService.updateTask(editingTask.id, taskData);
        message.success('任务更新成功');
      } else {
        await taskService.createTask(taskData);
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
      title: '数据源',
      dataIndex: 'data_source_id',
      key: 'data_source_id',
      render: (id: number) => {
        const ds = dataSources.find(d => d.id === id);
        return ds ? <Text>{ds.name}</Text> : <Text type="secondary">未知</Text>;
      },
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
      render: (_: any, record: Task) => (
        <Space size="small">
          <Button
            type="text"
            icon={<PlayCircleOutlined />}
            onClick={() => handleRunTask(record.id)}
            title="启用并调度"
          />
          <Button
            type="text"
            icon={<PauseCircleOutlined />}
            onClick={() => handleStopTask(record.id)}
            title="禁用任务"
          />
          <Tooltip title="手动执行">
            <Button
              type="text"
              icon={<ThunderboltOutlined />}
              onClick={() => handleExecuteTask(record.id)}
              style={{ color: '#722ed1' }}
            />
          </Tooltip>
          <Tooltip title="查看数据">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => handlePreviewData(record)}
              style={{ color: '#1890ff' }}
            />
          </Tooltip>
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
              defaultValue=""
              style={{ width: 120 }}
              onChange={handleStatusFilter}
            >
              <Option value="">全部</Option>
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

          <Form.Item
            name="data_source_id"
            label="数据源"
            rules={[{ required: true, message: '请选择数据源' }]}
          >
            <Select
              placeholder="请选择数据源"
              showSearch
              optionFilterProp="children"
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
              options={dataSources.map(ds => ({
                value: ds.id,
                label: `${ds.name} (${ds.type})`,
              }))}
            />
          </Form.Item>

          <Form.Item name="cron" label="Cron表达式">
            <Input placeholder="例如: 0 0 * * * (每小时执行一次)" />
          </Form.Item>

          <Form.Item name="replicas" label="并发数">
            <InputNumber min={1} max={10} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="execution_timeout" label="执行超时(秒)">
            <InputNumber min={60} max={86400} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="max_retries" label="最大重试次数">
            <InputNumber min={0} max={10} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="config"
            label={
              <Tooltip title={
                <div>
                  <div>通常无需填写。选择器配置在<b>数据源</b>的 selectors 字段中设置。</div>
                  <div style={{ marginTop: 4 }}>仅在需要覆盖数据源配置时填写，格式：</div>
                  <pre style={{ fontSize: 11, margin: '4px 0 0' }}>
{`{
  "data_source": {
    "url": "https://...",
    "selectors": {
      "title": "h1",
      "content": "#main"
    }
  }
}`}
                  </pre>
                </div>
              } overlayStyle={{ maxWidth: 360 }}>
                <span>任务配置(JSON) <Text type="secondary" style={{ fontSize: 12 }}>— 可选，选择器在数据源中配置</Text></span>
              </Tooltip>
            }
          >
            <TextArea
              rows={4}
              placeholder="通常留空。选择器在数据源的 selectors 字段中配置即可。"
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item name="status" label="状态" initialValue="enabled">
            <Select>
              <Option value="enabled">启用</Option>
              <Option value="disabled">禁用</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 数据预览抽屉 */}
      <Drawer
        title={`数据预览 - ${dataDrawerTask?.name || ''}`}
        open={dataDrawerVisible}
        onClose={() => setDataDrawerVisible(false)}
        width={900}
      >
        {previewTotal === 0 && !previewLoading ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            暂无采集数据，请先执行任务
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
              onChange: (page) => dataDrawerTask && handlePreviewData(dataDrawerTask, page),
            }}
          />
        )}
      </Drawer>
    </div>
  );
};

export default TasksPage;