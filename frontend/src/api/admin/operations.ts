import { apiClient } from '../client'

export type OperationPlatform = 'bluesky' | 'telegram' | 'discord'
export interface OperationConnection { id:number; platform:OperationPlatform; name:string; enabled:boolean; has_secrets:boolean; last_error?:string; last_success_at?:string }
export interface OperationPolicy { enabled:boolean; autonomous_enabled:boolean; auto_publish:boolean; require_approval:boolean; planning_interval_minutes:number; min_publish_interval_minutes:number; max_daily_actions:number; quiet_hours_start:number; quiet_hours_end:number; updated_at?:string }
export interface OperationTask { id:number; connection_id?:number; kind:string; platform:OperationPlatform; status:string; content:string; scheduled_at:string; attempts:number; max_attempts:number; last_error?:string; external_id?:string; created_at:string }
export interface OperationRun { id:number; task_id:number; status:string; output?:string; error?:string; started_at:string; finished_at?:string }
export interface OperationSummary { policy:OperationPolicy; connections:number; pending_tasks:number; succeeded_today:number }
export interface Page<T> { items:T[]; total:number; page:number; page_size:number }

const root = '/admin/assistant/operations'
export const operationsAPI = {
  summary: async () => (await apiClient.get<OperationSummary>(`${root}/summary`)).data,
  connections: async () => (await apiClient.get<OperationConnection[]>(`${root}/connections`)).data,
  saveConnection: async (payload:{platform:OperationPlatform;name:string;enabled:boolean;config:Record<string,string>}, id?:number) =>
    (await (id ? apiClient.put<OperationConnection>(`${root}/connections/${id}`, payload) : apiClient.post<OperationConnection>(`${root}/connections`, payload))).data,
  deleteConnection: (id:number) => apiClient.delete(`${root}/connections/${id}`),
  testConnection: (id:number) => apiClient.post(`${root}/connections/${id}/test`),
  policy: async () => (await apiClient.get<OperationPolicy>(`${root}/policy`)).data,
  updatePolicy: async (payload:OperationPolicy) => (await apiClient.put<OperationPolicy>(`${root}/policy`, payload)).data,
  tasks: async () => (await apiClient.get<Page<OperationTask>>(`${root}/tasks`)).data,
  runs: async () => (await apiClient.get<Page<OperationRun>>(`${root}/runs`)).data,
  createTask: (payload:{connection_id:number;kind:string;content:string}) => apiClient.post(`${root}/tasks`, payload),
  approveTask: (id:number) => apiClient.post(`${root}/tasks/${id}/approve`),
  runTask: (id:number) => apiClient.post(`${root}/tasks/${id}/run`),
  cancelTask: (id:number) => apiClient.post(`${root}/tasks/${id}/cancel`),
  planNow: () => apiClient.post(`${root}/plan-now`),
}
