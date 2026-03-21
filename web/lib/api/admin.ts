import { api } from "./client";

export interface ProviderConfig {
  provider: string;
  model?: string;
  endpoint?: string;
  api_key_ref?: string;
}

export interface AgentConfig {
  workspace_id: string;
  vision: ProviderConfig;
  translation: ProviderConfig;
  validation: ProviderConfig;
  fallback?: ProviderConfig;
  rules?: string[];
  updated_at: string;
}

export interface CallLogEntry {
  id: string;
  workspace_id: string;
  agent_type: string;
  provider: string;
  model: string;
  success: boolean;
  latency_ms: number;
  tokens_used: number;
  error?: string;
  created_at: string;
}

export interface Metrics {
  total_labels: number;
  confirmed_labels: number;
  failed_labels: number;
  avg_confidence: number;
  labels_this_month: number;
}

export interface RateLimit {
  workspace_id: string;
  uploads_per_minute: number;
  prints_per_day: number;
}

export const adminApi = {
  getAgentConfig: (workspaceId: string) =>
    api.get<AgentConfig>(`/v1/admin/workspaces/${workspaceId}/agent-config`),

  updateAgentConfig: (workspaceId: string, config: Partial<AgentConfig>) =>
    api.put<AgentConfig>(`/v1/admin/workspaces/${workspaceId}/agent-config`, config),

  getAgentLogs: (workspaceId: string, params?: { page?: number; per_page?: number }) =>
    api.get<{ logs: CallLogEntry[]; total: number }>(
      `/v1/admin/workspaces/${workspaceId}/agent-logs?` +
      new URLSearchParams(Object.fromEntries(Object.entries(params ?? {}).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])))
    ),

  getMetrics: (workspaceId: string) =>
    api.get<Metrics>(`/v1/admin/workspaces/${workspaceId}/metrics`),

  getRateLimits: (workspaceId: string) =>
    api.get<RateLimit>(`/v1/admin/workspaces/${workspaceId}/rate-limits`),

  setRateLimits: (workspaceId: string, limits: Partial<RateLimit>) =>
    api.put<RateLimit>(`/v1/admin/workspaces/${workspaceId}/rate-limits`, limits),

  testAgentConfig: (workspaceId: string, type: "vision" | "translation" | "validation") =>
    api.post<{ success: boolean; latency_ms?: number; error?: string }>(
      `/v1/admin/workspaces/${workspaceId}/agent-config/test?type=${type}`
    ),
};

export interface SuperAdminWorkspace {
  id: string;
  name: string;
  cui?: string;
  plan: string;
  label_quota: number;
  labels_used: number;
  created_at: string;
}

export const superAdminApi = {
  listWorkspaces: (params?: { limit?: number; offset?: number }) =>
    api.get<{ workspaces: SuperAdminWorkspace[]; total: number }>(
      `/v1/superadmin/workspaces?` +
      new URLSearchParams(Object.fromEntries(Object.entries(params ?? {}).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])))
    ),

  getWorkspace: (id: string) =>
    api.get<SuperAdminWorkspace & { address?: string; phone?: string }>(`/v1/superadmin/workspaces/${id}`),
};
