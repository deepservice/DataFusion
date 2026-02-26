import { BaseAPI } from './api';
import { Task, TaskExecution, PaginatedResponse } from '../types';

interface CreateTaskRequest {
  name: string;
  description?: string;
  type: string;
  data_source_id: number;
  cron?: string;
  replicas?: number;
  execution_timeout?: number;
  max_retries?: number;
  config?: Record<string, any>;
}

interface UpdateTaskRequest extends Partial<CreateTaskRequest> {}

interface TaskListParams {
  page?: number;
  limit?: number;
  search?: string;
  status?: string;
  type?: string;
}

interface ExecutionListParams {
  page?: number;
  limit?: number;
  task_id?: number;
  status?: string;
}

class TaskService extends BaseAPI {
  // 获取任务列表
  async getTasks(params?: TaskListParams): Promise<PaginatedResponse<Task>> {
    return this.get('/tasks', { params });
  }

  // 获取单个任务
  async getTask(id: number): Promise<Task> {
    return this.get(`/tasks/${id}`);
  }

  // 创建任务
  async createTask(data: CreateTaskRequest): Promise<Task> {
    return this.post('/tasks', data);
  }

  // 更新任务
  async updateTask(id: number, data: UpdateTaskRequest): Promise<void> {
    return this.put(`/tasks/${id}`, data);
  }

  // 删除任务
  async deleteTask(id: number): Promise<void> {
    return this.delete(`/tasks/${id}`);
  }

  // 运行任务
  async runTask(id: number): Promise<void> {
    return this.post(`/tasks/${id}/run`);
  }

  // 停止任务
  async stopTask(id: number): Promise<void> {
    return this.post(`/tasks/${id}/stop`);
  }

  // 手动执行任务
  async executeTask(id: number): Promise<{ message: string }> {
    return this.post(`/tasks/${id}/execute`);
  }

  // 获取任务采集数据预览
  async getTaskData(id: number, params?: { page?: number; limit?: number }): Promise<{
    items: Record<string, any>[];
    columns: string[];
    pagination: { page: number; limit: number; total: number };
  }> {
    return this.get(`/tasks/${id}/data`, { params });
  }

  // 获取系统统计概览
  async getStats(): Promise<{
    total_tasks: number;
    enabled_tasks: number;
    total_executions: number;
    success_executions: number;
    failed_executions: number;
    running_executions: number;
    total_records: number;
  }> {
    return this.get('/stats/overview');
  }

  // 获取执行历史
  async getExecutions(params?: ExecutionListParams): Promise<PaginatedResponse<TaskExecution>> {
    return this.get('/executions', { params });
  }

  // 获取单个执行记录
  async getExecution(id: number): Promise<TaskExecution> {
    return this.get(`/executions/${id}`);
  }

  // 获取任务的执行历史
  async getTaskExecutions(taskId: number, params?: Omit<ExecutionListParams, 'task_id'>): Promise<PaginatedResponse<TaskExecution>> {
    return this.get(`/executions/task/${taskId}`, { params });
  }
}

export const taskService = new TaskService();