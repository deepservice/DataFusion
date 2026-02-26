import React, { useState } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { Form, Input, Button, Card, Typography, message, Space } from 'antd';
import { UserOutlined, LockOutlined, LoginOutlined } from '@ant-design/icons';
import { authService } from '../../services/auth';
import { LoginRequest } from '../../types';

const { Title, Text } = Typography;

const LoginPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  // 如果已经登录，重定向到仪表板
  if (authService.isAuthenticated()) {
    return <Navigate to="/dashboard" replace />;
  }

  const handleLogin = async (values: LoginRequest) => {
    setLoading(true);
    try {
      await authService.login(values);
      message.success('登录成功');
      navigate('/dashboard');
    } catch (error: any) {
      message.error(error.response?.data?.error || '登录失败，请检查用户名和密码');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <Card className="login-form" bordered={false}>
        {/* Logo 和标题 */}
        <div className="login-logo">
          <Space direction="vertical" size="small" style={{ width: '100%', textAlign: 'center' }}>
            <div style={{ fontSize: 48, color: '#1890ff' }}>
              📊
            </div>
            <Title level={2} style={{ margin: 0, color: '#262626' }}>
              DataFusion
            </Title>
            <Text type="secondary">数据采集和处理平台</Text>
          </Space>
        </div>

        {/* 登录表单 */}
        <Form
          name="login"
          size="large"
          onFinish={handleLogin}
          autoComplete="off"
          style={{ marginTop: 32 }}
        >
          <Form.Item
            name="username"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 2, message: '用户名至少2个字符' },
            ]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder="用户名"
              autoComplete="username"
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码至少6个字符' },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="密码"
              autoComplete="current-password"
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              icon={<LoginOutlined />}
            >
              登录
            </Button>
          </Form.Item>
        </Form>

        {/* 默认账户提示 */}
        <div style={{ marginTop: 24, padding: 16, background: '#f6f8fa', borderRadius: 6 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            <strong>默认管理员账户：</strong>
            <br />
            用户名：admin
            <br />
            密码：Admin@123
            <br />
            <span style={{ fontSize: 11, color: '#999' }}>
              (密码需要包含大小写字母、数字)
            </span>
          </Text>
        </div>

        {/* 版权信息 */}
        <div style={{ textAlign: 'center', marginTop: 24 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            © 2024 DataFusion. All rights reserved.
          </Text>
        </div>
      </Card>
    </div>
  );
};

export default LoginPage;