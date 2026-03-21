import { api } from "./client";

export interface WorkspaceProfile {
  id: string;
  name: string;
  cui?: string;
  plan: string;
  label_quota: number;
  labels_used: number;
  created_at: string;
}

export interface Member {
  id: string;
  user_id: string;
  email: string;
  role: string;
  joined_at: string;
}

export interface Subscription {
  workspace_id: string;
  plan: string;
  status: string;
  current_period_end?: string;
  cancel_at_period_end: boolean;
}

export const workspaceApi = {
  getProfile: () =>
    api.get<WorkspaceProfile>("/v1/workspace"),

  updateProfile: (data: { name?: string; cui?: string }) =>
    api.put<WorkspaceProfile>("/v1/workspace/profile", data),

  getSubscription: () =>
    api.get<Subscription>("/v1/workspace/subscription"),

  listMembers: () =>
    api.get<{ members: Member[] }>("/v1/workspace/members"),

  inviteMember: (email: string, role: string) =>
    api.post<{ message: string }>("/v1/workspace/invite", { email, role }),

  revokeMember: (memberId: string) =>
    api.delete<{ success: boolean }>(`/v1/workspace/members/${memberId}`),

  acceptInvitation: (token: string) =>
    api.get<{ workspace_id: string; role: string }>(`/v1/workspace/invite/${token}`),
};
