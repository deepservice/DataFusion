import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { authService } from './services/auth';
import MainLayout from './components/Layout/MainLayout';
import LoginPage from './pages/Login';
import DashboardPage from './pages/Dashboard';
import TasksPage from './pages/Tasks';
import DataSourcesPage from './pages/DataSources';
import UsersPage from './pages/Users';
import ConfigPage from './pages/Config';
import BackupPage from './pages/Backup';
import ProfilePage from './pages/Profile';
import ExecutionsPage from './pages/Executions';

// 路由保护组件
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const isAuthenticated = authService.isAuthenticated();
  
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  
  return <>{children}</>;
};

// 管理员路由保护组件
const AdminRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const isAuthenticated = authService.isAuthenticated();
  const hasAdminRole = authService.hasRole('admin');
  
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  
  if (!hasAdminRole) {
    return <Navigate to="/dashboard" replace />;
  }
  
  return <>{children}</>;
};

const App: React.FC = () => {
  return (
    <Router>
      <Routes>
        {/* 登录页面 */}
        <Route path="/login" element={<LoginPage />} />
        
        {/* 主应用路由 */}
        <Route path="/" element={
          <ProtectedRoute>
            <MainLayout />
          </ProtectedRoute>
        }>
          {/* 默认重定向到仪表板 */}
          <Route index element={<Navigate to="/dashboard" replace />} />
          
          {/* 仪表板 */}
          <Route path="dashboard" element={<DashboardPage />} />
          
          {/* 任务管理 */}
          <Route path="tasks" element={<TasksPage />} />
          
          {/* 执行历史 */}
          <Route path="executions" element={<ExecutionsPage />} />

          {/* 数据源管理 */}
          <Route path="datasources" element={<DataSourcesPage />} />
          
          {/* 用户管理（仅管理员） */}
          <Route path="users" element={
            <AdminRoute>
              <UsersPage />
            </AdminRoute>
          } />
          
          {/* 系统配置（仅管理员） */}
          <Route path="config" element={
            <AdminRoute>
              <ConfigPage />
            </AdminRoute>
          } />
          
          {/* 备份管理（仅管理员） */}
          <Route path="backup" element={
            <AdminRoute>
              <BackupPage />
            </AdminRoute>
          } />
          
          {/* 个人资料 */}
          <Route path="profile" element={<ProfilePage />} />
        </Route>
        
        {/* 404 重定向 */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </Router>
  );
};

export default App;