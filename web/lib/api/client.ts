const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// Token accessors — set by authStore after init to break the circular dependency
let _getToken: () => string | null = () => null;
let _setToken: (t: string) => void = () => {};

export function registerTokenAccessors(
  get: () => string | null,
  set: (t: string) => void
) {
  _getToken = get;
  _setToken = set;
}

// Single in-flight refresh promise to prevent parallel refresh storms
let refreshPromise: Promise<string | null> | null = null;

async function doRefresh(): Promise<string | null> {
  try {
    const res = await fetch("/api/auth/refresh", { method: "POST" });
    if (!res.ok) return null;
    const data = await res.json();
    return (data.access_token as string) ?? null;
  } catch {
    return null;
  }
}

export interface ApiError {
  error: string;
  code: string;
  status: number;
}

export class ApiException extends Error {
  constructor(public readonly info: ApiError) {
    super(info.error);
    this.name = "ApiException";
  }
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  retry = true
): Promise<T> {
  const token = _getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init.headers as Record<string, string>),
  };

  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${API_URL}${path}`, { ...init, headers });

  if (res.status === 401 && retry) {
    if (!refreshPromise) {
      refreshPromise = doRefresh().finally(() => {
        refreshPromise = null;
      });
    }
    const newToken = await refreshPromise;
    if (!newToken) {
      throw new ApiException({ error: "Unauthorized", code: "UNAUTHORIZED", status: 401 });
    }
    _setToken(newToken);
    return request<T>(path, init, false);
  }

  if (!res.ok) {
    let info: ApiError;
    try {
      const body = await res.json();
      info = { error: body.error ?? "Unknown error", code: body.code ?? "UNKNOWN", status: res.status };
    } catch {
      info = { error: res.statusText, code: "HTTP_ERROR", status: res.status };
    }
    throw new ApiException(info);
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const api = {
  get: <T>(path: string, init?: RequestInit) =>
    request<T>(path, { ...init, method: "GET" }),

  post: <T>(path: string, body?: unknown, init?: RequestInit) =>
    request<T>(path, {
      ...init,
      method: "POST",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  patch: <T>(path: string, body?: unknown, init?: RequestInit) =>
    request<T>(path, {
      ...init,
      method: "PATCH",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  put: <T>(path: string, body?: unknown, init?: RequestInit) =>
    request<T>(path, {
      ...init,
      method: "PUT",
      body: body !== undefined ? JSON.stringify(body) : undefined,
    }),

  delete: <T>(path: string, init?: RequestInit) =>
    request<T>(path, { ...init, method: "DELETE" }),

  upload: <T>(path: string, body: FormData) =>
    request<T>(path, {
      method: "POST",
      body,
      headers: {},
    }),
};

export { API_URL };
