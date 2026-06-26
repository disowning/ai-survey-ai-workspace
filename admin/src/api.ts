import type { AnyRecord, ApiListResponse, ResourceConfig } from "./types";

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

  const response = await fetch(`${apiBaseUrl}${config.endpoint}?${params.toString()}`);
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
    method: "DELETE"
  });
  if (!response.ok) {
    throw new Error(await readError(response, "删除知识库内容失败"));
  }
}

async function readError(response: Response, fallback: string): Promise<string> {
  try {
    const data = (await response.json()) as { error?: string };
    return data.error || fallback;
  } catch {
    return fallback;
  }
}
