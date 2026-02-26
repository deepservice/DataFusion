import { BaseAPI } from './api';

export interface DataSource {
  id?: number;
  name: string;
  type: 'web' | 'api' | 'database';
  config: string; // JSON字符串
  description?: string;
  status?: 'active' | 'inactive';
  created_at?: string;
  updated_at?: string;
}

export interface DataSourceListResponse {
  data: DataSource[];
  total: number;
  page: number;
  page_size: number;
}

export interface PageElement {
  selector: string;
  tag: string;
  id?: string;
  class?: string;
  text: string;
}

export interface PreviewStructureResponse {
  status: string;
  url: string;
  title: string;
  elements: PageElement[];
  response_type?: 'html' | 'json'; // html=CSS选择器，json=API字段结构
}

export interface DataSourceConfig {
  // Web类型配置
  url?: string;
  method?: string;
  headers?: Record<string, string>;

  // API类型配置
  endpoint?: string;
  auth_type?: string;
  api_key?: string;

  // Database类型配置
  host?: string;
  port?: number;
  database?: string;
  username?: string;
  password?: string;
  db_type?: 'mysql' | 'postgresql' | 'mongodb';

  // 通用配置
  timeout?: number;
  [key: string]: any;
}

class DataSourceService extends BaseAPI {
  /**
   * 获取数据源列表
   */
  async list(params?: {
    page?: number;
    page_size?: number;
    type?: string;
  }): Promise<DataSourceListResponse> {
    const response = await this.client.request({
      method: 'GET',
      url: '/datasources',
      params,
    });
    return response.data as DataSourceListResponse;
  }

  /**
   * 获取单个数据源
   */
  async getById(id: number): Promise<DataSource> {
    return super.get(`/datasources/${id}`);
  }

  /**
   * 创建数据源
   */
  async create(dataSource: DataSource): Promise<DataSource> {
    return this.post('/datasources', dataSource);
  }

  /**
   * 更新数据源
   */
  async update(id: number, dataSource: Partial<DataSource>): Promise<void> {
    return this.put(`/datasources/${id}`, dataSource);
  }

  /**
   * 删除数据源
   */
  async deleteById(id: number): Promise<void> {
    return super.delete(`/datasources/${id}`);
  }

  /**
   * 测试数据源连接
   */
  async testConnection(id: number): Promise<{ status: string; message: string }> {
    return this.post(`/datasources/${id}/test`);
  }

  /**
   * 预览页面结构，返回可用的 CSS 选择器列表
   */
  async previewStructure(id: number): Promise<PreviewStructureResponse> {
    return this.post(`/datasources/${id}/preview`);
  }
}

export const dataSourceService = new DataSourceService();
