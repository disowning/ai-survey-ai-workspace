import type { AnyRecord, ApiListResponse, ResourceConfig } from "./types";

let authToken = "";

export function setAuthToken(token: string): void {
  authToken = token.trim();
}

export async function login(apiBaseUrl: string, username: string, password: string): Promise<{ token: string; expires_at: string }> {
  const response = await fetch(`${apiBaseUrl}/api/auth/login`, {
    method: "POST",
    headers: jsonHeaders(),
    body: JSON.stringify({ username, password })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "登录失败"));
  }

  const data = (await response.json()) as { token: string; expires_at: string };
  setAuthToken(data.token || "");
  return data;
}

export async function listResource(
  apiBaseUrl: string,
  config: ResourceConfig,
  filters: Record<string, string>,
  options: { limit?: number; offset?: number } = {}
): Promise<AnyRecord[]> {
  const params = new URLSearchParams();
  for (const key of config.filters) {
    const value = filters[key]?.trim();
    if (value) params.set(key, value);
  }
  params.set("limit", String(options.limit ?? 100));
  params.set("offset", String(options.offset ?? 0));

  const response = await fetch(`${apiBaseUrl}${config.endpoint}?${params.toString()}`, { headers: authHeaders() });
  if (!response.ok) {
    throw new Error(await readError(response, "读取数据失败"));
  }

  const data = (await response.json()) as ApiListResponse<AnyRecord>;
  return data.items ?? [];
}

export async function deleteKnowledge(apiBaseUrl: string, id: number, filters: Record<string, string>): Promise<void> {
  const params = new URLSearchParams();
  if (filters.profile_id) params.set("profile_id", filters.profile_id);
  if (filters.site_key) params.set("site_key", filters.site_key);

  const response = await fetch(`${apiBaseUrl}/api/knowledge/${id}?${params.toString()}`, {
    method: "DELETE",
    headers: authHeaders()
  });
  if (!response.ok) {
    throw new Error(await readError(response, "删除知识库内容失败"));
  }
}

export async function downloadExport(apiBaseUrl: string, path: string, filters: Record<string, string>): Promise<Blob> {
  const params = new URLSearchParams();
  if (filters.profile_id) params.set("profile_id", filters.profile_id);
  if (filters.site_key) params.set("site_key", filters.site_key);

  const response = await fetch(`${apiBaseUrl}${path}?${params.toString()}`, { headers: authHeaders() });
  if (!response.ok) {
    throw new Error(await readError(response, "导出失败"));
  }
  return response.blob();
}

function authHeaders(): HeadersInit {
  return authToken ? { Authorization: `Bearer ${authToken}` } : {};
}

function jsonHeaders(): HeadersInit {
  return {
    "Content-Type": "application/json",
    ...authHeaders()
  };
}

async function readError(response: Response, fallback: string): Promise<string> {
  try {
    const data = (await response.json()) as { error?: string };
    return data.error || fallback;
  } catch {
    return fallback;
  }
}
