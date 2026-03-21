import { api } from "./client";
import type { AuthUser, Role } from "@/lib/stores/authStore";

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: string;
    email: string;
    workspace_id: string;
    role: Role;
    is_superadmin?: boolean;
  };
}

export interface RegisterResponse {
  user_id: string;
  workspace_id: string;
  message: string;
}

export interface RefreshResponse {
  access_token: string;
  expires_in: number;
}

export function toAuthUser(r: LoginResponse["user"]): AuthUser {
  return {
    id: r.id,
    email: r.email,
    workspaceId: r.workspace_id,
    role: r.role,
    isSuperAdmin: r.is_superadmin,
  };
}

export const authApi = {
  login: (email: string, password: string) =>
    api.post<LoginResponse>("/v1/auth/login", { email, password }),

  register: (email: string, password: string, workspace_name: string, cui?: string) =>
    api.post<RegisterResponse>("/v1/auth/register", { email, password, workspace_name, cui }),

  logout: (refresh_token: string) =>
    api.post<{ success: boolean }>("/v1/auth/logout", { refresh_token }),

  refresh: (refresh_token: string) =>
    api.post<RefreshResponse>("/v1/auth/refresh", { refresh_token }),

  oauthGoogle: (id_token: string) =>
    api.post<LoginResponse>("/v1/auth/oauth/google", { id_token }),
};

/** Store refresh token in httpOnly cookie via Next.js API route */
export async function storeRefreshCookie(refresh_token: string) {
  await fetch("/api/auth/set-refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token }),
  });
}

/** Clear the httpOnly refresh cookie via Next.js API route */
export async function clearRefreshCookie() {
  await fetch("/api/auth/clear-refresh", { method: "POST" });
}