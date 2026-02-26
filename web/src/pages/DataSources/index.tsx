import React, { useState, useEffect } from 'react';
import {
  Typography,
  Card,
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Tooltip,
  Drawer,
  Spin,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ApiOutlined,
  ApartmentOutlined,
  CopyOutlined,
  ExpandAltOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { dataSourceService, DataSource, DataSourceConfig, PageElement } from '../../services/datasource';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;

// Web 类型数据源的默认配置模板（不含 selectors，默认抓取全部内容）
const WEB_CONFIG_TEMPLATE = {
  url: '',
  method: 'GET',
  headers: {},
};

// 配置说明文本
const SELECTORS_HELP = `Web 数据源 config 配置说明：

【基础】不配置 selectors 或为空 {}：自动提取页面主要内容

【精确提取】配置 selectors：
{
  "url": "https://example.com",
  "selectors": {
    "title": "h1",
    "content": "#article-body"
  }
}
key = 存入数据库的字段名，value = CSS 选择器

【账号密码登录】在 rpa_config.login 中配置账号（自动模拟登录）：
{
  "url": "https://www.dxy.cn/board/articles",
  "rpa_config": {
    "login": {
      "url": "https://www.dxy.cn/login",
      "username_selector": "#username",
      "password_selector": "#password",
      "submit_selector": "button[type='submit']",
      "username": "your-username",
      "password": "your-password",
      "wait_after": ".nav-user-avatar",
      "check_selector": ".nav-user-avatar"
    }
  }
}
check_selector：会话有效时页面上存在的元素，不存在则自动重新登录

【Cookie 注入】适用于短信验证码/扫码等无法自动登录的场景：
1. 在浏览器手动登录目标网站
2. 打开 DevTools (F12) → Network → 选任意请求 → Headers → Cookie
3. 复制 Cookie 值，填入 cookie_string：
{
  "url": "https://www.dxy.cn/board/articles",
  "rpa_config": {
    "cookie_string": "session_id=xxx; token=yyy; user_id=123",
    "check_selector": ".nav-user-avatar"
  }
}
check_selector：Cookie 失效时页面不存在的元素，失效则报错提示重新复制 Cookie

【动态交互】在 rpa_config.actions 中配置页面动作（搜索/筛选/点击）：
{
  "url": "https://example.com/list",
  "rpa_config": {
    "actions": [
      {"type": "input",  "selector": "#search", "value": "关键词"},
      {"type": "click",  "selector": "#search-btn", "wait_for": ".result-list"},
      {"type": "select", "selector": "#sort-by", "value": "latest"},
      {"type": "wait",   "wait_ms": 1000}
    ]
  }
}
登录/Cookie 和动作可同时配置。账号密码登录的 Cookie 在内存中保存 24h 并自动复用。`;

const DataSourcesPage: React.FC = () => {
  const [dataSources, setDataSources] = useState<DataSource[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingDataSource, setEditingDataSource] = useState<DataSource | null>(null);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [form] = Form.useForm();

  // 页面结构预览 Drawer 状态
  const [previewDrawerVisible, setPreviewDrawerVisible] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewTitle, setPreviewTitle] = useState('');
  const [previewElements, setPreviewElements] = useState<PageElement[]>([]);
  const [previewError, setPreviewError] = useState('');
  const [previewResponseType, setPreviewResponseType] = useState<'html' | 'json'>('html');

  // 查看元素全文 Modal
  const [fullTextVisible, setFullTextVisible] = useState(false);
  const [fullTextContent, setFullTextContent] = useState('');
  const [fullTextSelector, setFullTextSelector] = useState('');

  const loadDataSources = async (page = 1, pageSize = 10) => {
    setLoading(true);
    try {
      const response = await dataSourceService.list({ page, page_size: pageSize });
      setDataSources(response.data || []);
      setPagination({ current: response.page, pageSize: response.page_size, total: response.total });
    } catch (error) {
      message.error('加载数据源列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadDataSources(); }, []);

  const handleOpenModal = (dataSource?: DataSource) => {
    setEditingDataSource(dataSource || null);
    if (dataSource) {
      form.setFieldsValue({
        ...dataSource,
        config: typeof dataSource.config === 'string'
          ? dataSource.config
          : JSON.stringify(dataSource.config, null, 2),
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        status: 'active',
        type: 'web',
        config: JSON.stringify(WEB_CONFIG_TEMPLATE, null, 2),
      });
    }
    setModalVisible(true);
  };

  const handleCloseModal = () => {
    setModalVisible(false);
    setEditingDataSource(null);
    form.resetFields();
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      try { JSON.parse(values.config); } catch {
        message.error('配置JSON格式错误');
        return;
      }
      const dataSourceData: DataSource = {
        name: values.name, type: values.type, config: values.config,
        description: values.description, status: values.status,
      };
      if (editingDataSource) {
        await dataSourceService.update(editingDataSource.id!, dataSourceData);
        message.success('数据源更新成功');
      } else {
        await dataSourceService.create(dataSourceData);
        message.success('数据源创建成功');
      }
      handleCloseModal();
      loadDataSources(pagination.current, pagination.pageSize);
    } catch (error: any) {
      if (error.errorFields) message.error('请填写所有必填字段');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await dataSourceService.deleteById(id);
      message.success('数据源删除成功');
      loadDataSources(pagination.current, pagination.pageSize);
    } catch { message.error('删除数据源失败'); }
  };

  const handleTestConnection = async (id: number) => {
    try {
      const result = await dataSourceService.testConnection(id);
      if (result.status === 'success') {
        message.success(result.message || '连接测试成功');
      } else {
        message.error(result.message || '连接测试失败');
      }
    } catch { message.error('连接测试失败'); }
  };

  const handlePreviewStructure = async (record: DataSource) => {
    setPreviewElements([]);
    setPreviewError('');
    setPreviewTitle('');
    setPreviewResponseType('html');
    setPreviewDrawerVisible(true);
    setPreviewLoading(true);
    try {
      const result = await dataSourceService.previewStructure(record.id!);
      setPreviewTitle(result.title || '（无标题）');
      setPreviewElements(result.elements || []);
      setPreviewResponseType(result.response_type || 'html');
      if (!result.elements?.length) {
        setPreviewError(
          result.response_type === 'json'
            ? 'API 返回了空响应或无法解析的 JSON'
            : '未找到有效元素，该页面可能需要登录或依赖 JavaScript 渲染'
        );
      }
    } catch (error: any) {
      setPreviewError(error?.response?.data?.error || '预览失败，请检查数据源 URL 是否正确');
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleCopySelector = (selector: string) => {
    navigator.clipboard.writeText(selector).then(() => {
      message.success(`已复制选择器: ${selector}`);
    });
  };

  const handleShowFullText = (selector: string, text: string) => {
    setFullTextSelector(selector);
    setFullTextContent(text);
    setFullTextVisible(true);
  };

  const handleTypeChange = (type: string) => {
    const templates: Record<string, DataSourceConfig> = {
      web: WEB_CONFIG_TEMPLATE,
      api: { url: '', method: 'GET', headers: {}, auth_type: 'none', timeout: 30 },
      database: { host: 'localhost', port: 5432, database: '', username: '', password: '', db_type: 'postgresql' },
    };
    form.setFieldsValue({ config: JSON.stringify(templates[type] || {}, null, 2) });
  };

  // 页面元素表格列（根据 response_type 动态调整）
  const isJSON = previewResponseType === 'json';
  const elementColumns: ColumnsType<PageElement> = [
    {
      title: isJSON ? '字段路径' : 'CSS 选择器',
      dataIndex: 'selector',
      key: 'selector',
      width: 200,
      render: (selector: string) => (
        <Space>
          <Text code style={{ fontSize: 12 }}>{selector}</Text>
          <Tooltip title={isJSON ? '复制字段路径' : '复制选择器'}>
            <Button type="text" size="small" icon={<CopyOutlined />}
              onClick={() => handleCopySelector(selector)} />
          </Tooltip>
        </Space>
      ),
    },
    {
      title: isJSON ? '类型' : '标签',
      dataIndex: 'tag',
      key: 'tag',
      width: 80,
      render: (tag: string) => {
        const colorMap: Record<string, string> = {
          string: 'blue', number: 'green', bool: 'orange',
          object: 'purple', array: 'cyan', null: 'default',
        };
        return <Tag color={colorMap[tag]}>{tag}</Tag>;
      },
    },
    {
      title: '值预览',
      dataIndex: 'text',
      key: 'text',
      render: (text: string, record: PageElement) => (
        <Space style={{ width: '100%' }}>
          <span style={{ color: '#555', flex: 1 }}>
            {text.length > 80 ? text.slice(0, 80) + '...' : text}
          </span>
          {text.length > 0 && (
            <Tooltip title="查看完整内容">
              <Button type="text" size="small" icon={<ExpandAltOutlined />}
                onClick={() => handleShowFullText(record.selector, text)} />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  const columns: ColumnsType<DataSource> = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
    { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
    {
      title: '类型', dataIndex: 'type', key: 'type', width: 120,
      render: (type: string) => {
        const colorMap: Record<string, string> = { web: 'blue', api: 'green', database: 'purple' };
        return <Tag color={colorMap[type] || 'default'}>{type.toUpperCase()}</Tag>;
      },
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (status: string) => (
        <Tag icon={status === 'active' ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
          color={status === 'active' ? 'success' : 'default'}>
          {status === 'active' ? '活动' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180,
      render: (text: string) => text ? new Date(text).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作', key: 'action', width: 160, fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="测试连接">
            <Button type="link" size="small" icon={<ApiOutlined />}
              onClick={() => handleTestConnection(record.id!)} />
          </Tooltip>
          <Tooltip title="预览结构（Web: CSS 选择器 / API: 响应字段）">
            <Button type="link" size="small" icon={<ApartmentOutlined />}
              style={{ color: '#52c41a' }}
              onClick={() => handlePreviewStructure(record)} />
          </Tooltip>
          <Tooltip title="编辑">
            <Button type="link" size="small" icon={<EditOutlined />}
              onClick={() => handleOpenModal(record)} />
          </Tooltip>
          <Popconfirm title="确定删除此数据源吗？" onConfirm={() => handleDelete(record.id!)}
            okText="确定" cancelText="取消">
            <Tooltip title="删除">
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="fade-in">
      <div className="page-header">
        <Title level={3} className="page-title">数据源管理</Title>
        <Text className="page-description">管理数据采集的数据源配置</Text>
      </div>

      <Card extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => handleOpenModal()}>添加数据源</Button>}>
        <Table
          columns={columns}
          dataSource={dataSources}
          loading={loading}
          rowKey="id"
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
          onChange={(p) => loadDataSources(p.current || 1, p.pageSize || 10)}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={editingDataSource ? '编辑数据源' : '添加数据源'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={handleCloseModal}
        width={700}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" initialValues={{ status: 'active', type: 'web' }}>
          <Form.Item name="name" label="数据源名称"
            rules={[{ required: true, message: '请输入数据源名称' }]}>
            <Input placeholder="请输入数据源名称" />
          </Form.Item>
          <Form.Item name="type" label="数据源类型"
            rules={[{ required: true, message: '请选择数据源类型' }]}>
            <Select onChange={handleTypeChange} disabled={!!editingDataSource}>
              <Option value="web">Web</Option>
              <Option value="api">API</Option>
              <Option value="database">Database</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="config"
            label={
              <Space>
                <span>配置 (JSON)</span>
                <Tooltip title={SELECTORS_HELP}>
                  <Text type="secondary" style={{ fontSize: 12, cursor: 'help' }}>
                    ❓ Web 类型通过 selectors 字段配置提取规则
                  </Text>
                </Tooltip>
              </Space>
            }
            rules={[
              { required: true, message: '请输入配置' },
              {
                validator: (_, value) => {
                  try { JSON.parse(value); return Promise.resolve(); }
                  catch { return Promise.reject(new Error('必须是有效的 JSON 格式')); }
                },
              },
            ]}>
            <TextArea rows={12} style={{ fontFamily: 'monospace', fontSize: 13 }} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={2} placeholder="请输入数据源描述" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select>
              <Option value="active">活动</Option>
              <Option value="inactive">禁用</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 页面/API 结构预览 Drawer */}
      <Drawer
        title={isJSON ? '预览 API 响应结构' : '预览页面结构'}
        open={previewDrawerVisible}
        onClose={() => setPreviewDrawerVisible(false)}
        width={720}
      >
        {previewLoading ? (
          <div style={{ textAlign: 'center', padding: 60 }}>
            <Spin size="large" />
            <div style={{ marginTop: 16, color: '#999' }}>
              {isJSON ? '正在请求 API，请稍候...' : '正在抓取页面结构，请稍候...'}
            </div>
          </div>
        ) : previewError ? (
          <div>
            <Alert type="warning" message={previewError} showIcon style={{ marginBottom: 16 }} />
            {(previewError.includes('登录') || previewError.includes('JavaScript') || previewError.includes('元素')) && (
              <Alert
                type="info"
                message="预览功能说明与解决方案"
                description={
                  <div>
                    <div style={{ marginBottom: 8 }}>预览功能通过直接 HTTP 请求获取内容，不支持 JavaScript 渲染和需要登录的页面。</div>
                    <div><strong>Web 类型数据源（需要登录或 JS 渲染）：</strong></div>
                    <ul style={{ marginTop: 4, paddingLeft: 20 }}>
                      <li>在数据源 config 中配置 <Text code>rpa_config.cookie_string</Text>，将浏览器已登录的 Cookie 粘贴进来</li>
                      <li>可点击上方 <strong>❓</strong> 查看 Cookie 注入配置示例</li>
                      <li>配置后直接运行关联任务即可采集，无需预览成功</li>
                    </ul>
                    <div style={{ marginTop: 8 }}><strong>API 类型数据源（需要认证）：</strong></div>
                    <ul style={{ marginTop: 4, paddingLeft: 20 }}>
                      <li>在 config 的 <Text code>headers</Text> 中加入 <Text code>Cookie</Text> 或 <Text code>Authorization</Text> 等认证头</li>
                      <li>使用"测试连接"功能验证配置是否正确</li>
                    </ul>
                  </div>
                }
                showIcon
              />
            )}
          </div>
        ) : (
          <>
            {previewTitle && (
              <div style={{ marginBottom: 12 }}>
                <Text strong>{isJSON ? 'API 地址：' : '页面标题：'}</Text>
                <Text>{previewTitle}</Text>
              </div>
            )}
            <Alert
              type="info"
              style={{ marginBottom: 16 }}
              message="如何使用"
              description={isJSON ? (
                <div>
                  <div>API 响应字段列表（嵌套字段用 <Text code>.</Text> 分隔，数组用 <Text code>[0]</Text> 表示）</div>
                  <div style={{ marginTop: 4 }}>点击 <CopyOutlined /> 复制字段路径，用于配置数据清洗规则或任务 config 中的字段映射</div>
                  <div style={{ marginTop: 4 }}>若 API 需要认证，请在数据源 config 的 <Text code>headers</Text> 中加入 Authorization 等头信息</div>
                </div>
              ) : (
                <div>
                  <div>1. 点击 <CopyOutlined /> 复制需要的 CSS 选择器</div>
                  <div>2. 编辑此数据源，在配置 JSON 的 <Text code>selectors</Text> 中填入：</div>
                  <pre style={{ margin: '6px 0 0', fontSize: 12, background: '#f0f5ff', padding: '8px 10px', borderRadius: 4 }}>
{`"selectors": {
  "title": "h1",
  "content": "#js_content"
}`}
                  </pre>
                  <div style={{ marginTop: 4 }}>3. 不配置 <Text code>selectors</Text> 或配置为空 <Text code>{'{}'}</Text>，均自动提取页面主要内容</div>
                </div>
              )}
            />
            <Table
              dataSource={previewElements}
              columns={elementColumns}
              rowKey={(r) => r.selector + r.tag}
              size="small"
              pagination={{ pageSize: 25, showTotal: (t) => `共 ${t} 个元素` }}
            />
          </>
        )}
      </Drawer>

      {/* 查看元素完整内容 Modal */}
      <Modal
        title={<Space><Text code>{fullTextSelector}</Text><Text type="secondary">完整内容</Text></Space>}
        open={fullTextVisible}
        onCancel={() => setFullTextVisible(false)}
        footer={
          <Button
            icon={<CopyOutlined />}
            onClick={() => {
              navigator.clipboard.writeText(fullTextContent);
              message.success('已复制内容');
            }}
          >
            复制内容
          </Button>
        }
        width={680}
      >
        <Paragraph
          style={{
            maxHeight: 400,
            overflowY: 'auto',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all',
            background: '#fafafa',
            padding: 12,
            borderRadius: 4,
            fontSize: 13,
          }}
        >
          {fullTextContent}
        </Paragraph>
      </Modal>
    </div>
  );
};

export default DataSourcesPage;
