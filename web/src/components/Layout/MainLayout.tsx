import React, { useState, useEffect } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Layout,
  Menu,
  Avatar,
  Dropdown,
  Space,
  Typography,
  Button,
  theme,
  MenuProps,
} from 'antd';
import {
  DashboardOutlined,
  ScheduleOutlined,
  DatabaseOutlined,
  UserOutlined,
  SettingOutlined,
  SaveOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
  HistoryOutlined,
} from '@ant-design/icons';
import { authService } from '../../services/auth';
import { User } from '../../types';

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

const MainLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const { token } = theme.useToken();

  useEffect(() => {
    // 获取当前用户信息
    const user = authService.getCurrentUserFromStorage();
    setCurrentUser(user);
  }, []);

  // 菜单项配置
  const menuItems: MenuProps['items'] = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: '仪表板',
    },
    {
      key: '/tasks',
      icon: <ScheduleOutlined />,
      label: '任务管理',
    },
    {
      key: '/executions',
      icon: <HistoryOutlined />,
      label: '执行历史',
    },
    {
      key: '/datasources',
      icon: <DatabaseOutlined />,
      label: '数据源',
    },
    // 管理员专用菜单
    ...(authService.hasRole('admin') ? [
      {
        key: '/users',
        icon: <UserOutlined />,
        label: '用户管理',
      },
      {
        key: 'admin',
        icon: <SettingOutlined />,
        label: '系统管理',
        children: [
          {
            key: '/config',
            label: '系统配置',
          },
          {
            key: '/backup',
            icon: <SaveOutlined />,
            label: '备份管理',
          },
        ],
      },
    ] : []),
  ];

  // 用户下拉菜单
  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人资料',
      onClick: () => navigate('/profile'),
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: handleLogout,
    },
  ];

  // 处理菜单点击
  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key);
  };

  // 处理登出
  async function handleLogout() {
    try {
      await authService.logout();
      navigate('/login');
    } catch (error) {
      // 即使登出请求失败，也要清除本地存储并跳转
      navigate('/login');
    }
  }

  // 获取当前选中的菜单项
  const getSelectedKeys = () => {
    const pathname = location.pathname;
    
    // 处理子菜单的情况
    if (pathname === '/config' || pathname === '/backup') {
      return [pathname];
    }
    
    return [pathname];
  };

  // 获取展开的菜单项
  const getOpenKeys = () => {
    const pathname = location.pathname;
    
    if (pathname === '/config' || pathname === '/backup') {
      return ['admin'];
    }
    
    return [];
  };

  return (
    <Layout className="main-layout">
      {/* 侧边栏 */}
      <Sider 
        trigger={null} 
        collapsible 
        collapsed={collapsed}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
        }}
      >
        {/* Logo */}
        <div style={{ 
          height: 64, 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          borderBottom: `1px solid ${token.colorBorder}`,
        }}>
          <Text 
            style={{ 
              color: 'white', 
              fontSize: collapsed ? 16 : 18, 
              fontWeight: 'bold' 
            }}
          >
            {collapsed ? 'DF' : 'DataFusion'}
          </Text>
        </div>

        {/* 菜单 */}
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={getSelectedKeys()}
          defaultOpenKeys={getOpenKeys()}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0 }}
        />
      </Sider>

      {/* 主内容区域 */}
      <Layout style={{ marginLeft: collapsed ? 80 : 200, transition: 'margin-left 0.2s' }}>
        {/* 顶部导航 */}
        <Header style={{ 
          padding: '0 24px', 
          background: token.colorBgContainer,
          borderBottom: `1px solid ${token.colorBorder}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}>
          {/* 左侧：折叠按钮 */}
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            style={{ fontSize: 16, width: 64, height: 64 }}
          />

          {/* 右侧：用户信息 */}
          <Space size="middle">
            {/* 通知按钮 */}
            <Button type="text" icon={<BellOutlined />} />

            {/* 用户下拉菜单 */}
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Space style={{ cursor: 'pointer' }}>
                <Avatar icon={<UserOutlined />} />
                <div style={{ display: collapsed ? 'none' : 'block' }}>
                  <div style={{ fontSize: 14, fontWeight: 500 }}>
                    {currentUser?.username || '用户'}
                  </div>
                  <div style={{ fontSize: 12, color: token.colorTextSecondary }}>
                    {currentUser?.role || '角色'}
                  </div>
                </div>
              </Space>
            </Dropdown>
          </Space>
        </Header>

        {/* 内容区域 */}
        <Content style={{
          margin: '24px',
          padding: '24px',
          background: token.colorBgContainer,
          borderRadius: token.borderRadius,
          minHeight: 'calc(100vh - 112px)',
          overflow: 'auto',
        }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
};

export default MainLayout;