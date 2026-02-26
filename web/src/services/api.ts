import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { message } from 'antd';

// API 基础配置
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || '/api/v1';

// 创建 axios 实例
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Token 管理
class TokenManager {
  private static readonly TOKEN_KEY = 'datafusion_token';
  private static readonly USER_KEY = 'datafusion_user';

  static getToken(): string | null {
    return localStorage.getItem(this.TOKEN_KEY);
  }

  static setToken(token: string): void {
    localStorage.setItem(this.TOKEN_KEY, token);
  }

  static removeToken(): void {
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.USER_KEY);
  }

  static getUser(): any {
    const userStr = localStorage.getItem(this.USER_KEY);
    return userStr ? JSON.parse(userStr) : null;
  }

  static setUser(user: any): void {
    localStorage.setItem(this.USER_KEY, JSON.stringify(user));
  }

  static isAuthenticated(): boolean {
    return !!this.getToken();
  }
}

// 请求拦截器
apiClient.interceptors.request.use(
  (config: any) => {
    const token = TokenManager.getToken();
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    return response;
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response;
      
      switch (status) {
        case 401:
          // 未授权，清除token并跳转到登录页
          TokenManager.removeToken();
          if (window.location.pathname !== '/login') {
            window.location.href = '/login';
          }
          message.error('登录已过期，请重新登录');
          break;
        case 403:
          message.error('权限不足');
          break;
        case 404:
          message.error('请求的资源不存在');
          break;
        case 500:
          message.error('服务器内部错误');
          break;
        default:
          message.error(data?.error || data?.message || '请求失败');
      }
    } else if (error.request) {
      message.error('网络连接失败，请检查网络');
    } else {
      message.error('请求配置错误');
    }
    
    return Promise.reject(error);
  }
);

// API 响应类型
interface ApiResponse<T = any> {
  data?: T;
  message?: string;
  error?: string;
}

// 基础 API 类
class BaseAPI {
  protected client = apiClient;

  protected async request<T = any>(config: AxiosRequestConfig): Promise<T> {
    try {
      const response = await this.client.request<ApiResponse<T>>(config);
      return (response.data.data || response.data) as unknown as T;
    } catch (error) {
      throw error;
    }
  }

  protected get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: 'GET', url });
  }

  protected post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: 'POST', url, data });
  }

  protected put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: 'PUT', url, data });
  }

  protected delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: 'DELETE', url });
  }
}

export { apiClient, TokenManager, BaseAPI };
export type { ApiResponse };