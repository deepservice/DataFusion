import React, { useState, useEffect } from 'react';
import { Card, Form, Input, Button, message, Typography, Space, Avatar } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { authService } from '../../services/auth';
import { User } from '../../types';

const { Title, Text } = Typography;

const ProfilePage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [profileForm] = Form.useForm();
  const [passwordForm] = Form.useForm();

  useEffect(() => {
    loadUserProfile();
  }, []);

  const loadUserProfile = async () => {
    try {
      const user = await authService.getCurrentUser();
      setCurrentUser(user);
      profileForm.setFieldsValue({
        username: user.username,
        email: user.email,
      });
    } catch (error) {
      message.error('加载用户信息失败');
    }
  };

  const handleUpdateProfile = async (values: { email: string }) => {
    setLoading(true);
    try {
      await authService.updateProfile(values);
      message.success('个人信息更新成功');
      loadUserProfile();
    } catch (error) {
      message.error('个人信息更新失败');
    } finally {
      setLoading(false);
    }
  };

  const handleChangePassword = async (values: { old_password: string; new_password: string }) => {
    setPasswordLoading(true);
    try {
      await authService.changePassword(values);
      message.success('密码修改成功');
      passwordForm.resetFields();
    } catch (error) {
      message.error('密码修改失败');
    } finally {
      setPasswordLoading(false);
    }
  };

  return (
    <div className="fade-in">
      <div className="page-header">
        <Title level={3} className="page-title">个人资料</Title>
        <Text className="page-description">管理您的个人信息和账户设置</Text>
      </div>

      <div style={{ maxWidth: 600 }}>
        {/* 用户信息卡片 */}
        <Card style={{ marginBottom: 24 }}>
          <Space size="large">
            <Avatar size={64} icon={<UserOutlined />} />
            <div>
              <Title level={4} style={{ margin: 0 }}>
                {currentUser?.username}
              </Title>
              <Text type="secondary">{currentUser?.role}</Text>
              <br />
              <Text type="secondary">
                注册时间: {currentUser?.created_at ? new Date(currentUser.created_at).toLocaleDateString() : '-'}
              </Text>
            </div>
          </Space>
        </Card>

        {/* 个人信息表单 */}
        <Card title="个人信息" style={{ marginBottom: 24 }}>
          <Form
            form={profileForm}
            layout="vertical"
            onFinish={handleUpdateProfile}
          >
            <Form.Item
              name="username"
              label="用户名"
            >
              <Input disabled />
            </Form.Item>

            <Form.Item
              name="email"
              label="邮箱"
              rules={[
                { type: 'email', message: '请输入有效的邮箱地址' },
              ]}
            >
              <Input placeholder="请输入邮箱地址" />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading}>
                更新信息
              </Button>
            </Form.Item>
          </Form>
        </Card>

        {/* 修改密码表单 */}
        <Card title="修改密码">
          <Form
            form={passwordForm}
            layout="vertical"
            onFinish={handleChangePassword}
          >
            <Form.Item
              name="old_password"
              label="当前密码"
              rules={[{ required: true, message: '请输入当前密码' }]}
            >
              <Input.Password prefix={<LockOutlined />} placeholder="请输入当前密码" />
            </Form.Item>

            <Form.Item
              name="new_password"
              label="新密码"
              rules={[
                { required: true, message: '请输入新密码' },
                { min: 8, message: '密码至少8个字符' },
              ]}
            >
              <Input.Password prefix={<LockOutlined />} placeholder="请输入新密码" />
            </Form.Item>

            <Form.Item
              name="confirm_password"
              label="确认新密码"
              dependencies={['new_password']}
              rules={[
                { required: true, message: '请确认新密码' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('new_password') === value) {
                      return Promise.resolve();
                    }
                    return Promise.reject(new Error('两次输入的密码不一致'));
                  },
                }),
              ]}
            >
              <Input.Password prefix={<LockOutlined />} placeholder="请确认新密码" />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={passwordLoading}>
                修改密码
              </Button>
            </Form.Item>
          </Form>
        </Card>
      </div>
    </div>
  );
};

export default ProfilePage;