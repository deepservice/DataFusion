import { BaseAPI } from './api';
import { User, PaginatedResponse, Role } from '@/types';

interface CreateUserRequest {
  username: string;
  password: string;
  email?: string;
  role: string;
}

interface UpdateUserRequest {
  email?: string;
  role?: string;
  status?: string;
}

interface UserListParams {
  page?: number;
  limit?: number;
  search?: string;
}

class UserService extends BaseAPI {
  // 获取用户列表
  async getUsers(params?: UserListParams): Promise<PaginatedResponse<User>> {
    return this.get('/users', { params });
  }

  // 获取单个用户
  async getUser(id: number): Promise<User> {
    return this.get(`/users/${id}`);
  }

  // 创建用户
  async createUser(data: CreateUserRequest): Promise<User> {
    return this.post('/users', data);
  }

  // 更新用户
  async updateUser(id: number, data: UpdateUserRequest): Promise<void> {
    return this.put(`/users/${id}`, data);
  }

  // 删除用户
  async deleteUser(id: number): Promise<void> {
    return this.delete(`/users/${id}`);
  }

  // 重置用户密码
  async resetPassword(id: number, newPassword: string): Promise<void> {
    return this.post(`/users/${id}/reset-password`, { new_password: newPassword });
  }

  // 获取所有角色
  async getRoles(): Promise<{ roles: Role[] }> {
    return this.get('/roles');
  }
}

export const userService = new UserService();