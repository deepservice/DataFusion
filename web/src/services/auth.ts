import { BaseAPI, TokenManager } from './api';
import { LoginRequest, LoginResponse, User } from '../types';

class AuthService extends BaseAPI {
  // 用户登录
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await this.post<LoginResponse>('/auth/login', credentials);
    
    // 保存 token 和用户信息
    TokenManager.setToken(response.token);
    TokenManager.setUser(response.user);
    
    return response;
  }

  // 用户登出
  async logout(): Promise<void> {
    try {
      await this.post('/auth/logout');
    } finally {
      // 无论请求是否成功，都清除本地存储
      TokenManager.removeToken();
    }
  }

  // 刷新 token
  async refreshToken(): Promise<LoginResponse> {
    const token = TokenManager.getToken();
    if (!token) {
      throw new Error('No token available');
    }

    const response = await this.post<LoginResponse>('/auth/refresh', { token });
    
    // 更新 token 和用户信息
    TokenManager.setToken(response.token);
    TokenManager.setUser(response.user);
    
    return response;
  }

  // 获取当前用户信息
  async getCurrentUser(): Promise<User> {
    return this.get<User>('/auth/profile');
  }

  // 更新用户信息
  async updateProfile(data: { email?: string }): Promise<void> {
    return this.put('/auth/profile', data);
  }

  // 修改密码
  async changePassword(data: { old_password: string; new_password: string }): Promise<void> {
    return this.post('/auth/change-password', data);
  }

  // 检查是否已登录
  isAuthenticated(): boolean {
    return TokenManager.isAuthenticated();
  }

  // 获取当前用户
  getCurrentUserFromStorage(): User | null {
    return TokenManager.getUser();
  }

  // 检查用户权限
  hasPermission(permission: string): boolean {
    const user = this.getCurrentUserFromStorage();
    if (!user) return false;

    // 管理员拥有所有权限
    if (user.role === 'admin') return true;

    // 这里可以根据实际的权限系统进行扩展
    // 目前简化处理
    switch (user.role) {
      case 'operator':
        return ['tasks', 'datasources', 'executions', 'cleaning-rules', 'stats'].some(
          resource => permission.startsWith(resource)
        );
      case 'viewer':
        return permission.includes(':read') || permission.includes('stats');
      case 'user':
        return permission.includes('tasks:read') || permission.includes('stats:read');
      default:
        return false;
    }
  }

  // 检查用户角色
  hasRole(role: string): boolean {
    const user = this.getCurrentUserFromStorage();
    return user?.role === role;
  }
}

export const authService = new AuthService();