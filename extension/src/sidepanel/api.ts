import type {
  AIConversation,
  KnowledgeChunk,
  LocalProfile,
  Note,
  SiteDetection,
  SnapshotPayload,
  Translation
} from "../shared/types";

type BindProfileResponse = {
  id: number;
  profile_key: string;
  profile_name?: string;
};

type DetectSiteResponse = {
  site: SiteDetection;
  detected: boolean;
  persisted: boolean;
};

export type SystemStatus = {
  status: string;
  database_ok: boolean;
  ai_configured: boolean;
  embedding_configured: boolean;
  ai_model?: string;
  ai_embedding_model?: string;
};

export async function checkHealth(apiBaseUrl: string): Promise<SystemStatus | null> {
  try {
    const response = await fetch(`${apiBaseUrl}/api/system/status`);
    if (!response.ok) return null;
    return (await response.json()) as SystemStatus;
  } catch {
    return null;
  }
}

export async function bindProfile(apiBaseUrl: string, profileKey: string, profileName: string): Promise<LocalProfile> {
  const response = await fetch(`${apiBaseUrl}/api/profiles/bind`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      profile_key: profileKey,
      profile_name: profileName || profileKey
    })
  });

  if (!response.ok) {
    throw new Error(await readError(response, "绑定 Profile 失败"));
  }

  const data = (await response.json()) as BindProfileResponse;
  return {
    profileId: data.id,
    profileKey: data.profile_key,
    profileName: data.profile_name || data.profile_key
  };
}

export async function detectSite(apiBaseUrl: string, pageUrl: string): Promise<DetectSiteResponse> {
  const response = await fetch(`${apiBaseUrl}/api/sites/detect`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url: pageUrl })
  });

  if (!response.ok) {
    throw new Error(await readError(response, "识别网站失败"));
  }

  return (await response.json()) as DetectSiteResponse;
}

export async function ensureSite(apiBaseUrl: string, site: SiteDetection): Promise<SiteDetection> {
  const response = await fetch(`${apiBaseUrl}/api/sites`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      site_key: site.site_key,
      site_name: site.site_name || site.site_key,
      domain: site.domain
    })
  });

  if (response.ok) {
    return (await response.json()) as SiteDetection;
  }

  throw new Error(await readError(response, "保存网站记录失败"));
}

export async function saveSnapshot(apiBaseUrl: string, payload: SnapshotPayload): Promise<void> {
  const response = await fetch(`${apiBaseUrl}/api/page-snapshots`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(await readError(response, "保存页面快照失败"));
  }
}

export async function createNote(
  apiBaseUrl: string,
  payload: { profile_id: number; site_key: string; survey_id?: number; note_text: string }
): Promise<Note> {
  const response = await fetch(`${apiBaseUrl}/api/notes`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(await readError(response, "保存笔记失败"));
  }

  return (await response.json()) as Note;
}

export async function listNotes(apiBaseUrl: string, profileId: number, siteKey: string): Promise<Note[]> {
  const params = new URLSearchParams({
    profile_id: String(profileId),
    site_key: siteKey,
    limit: "20"
  });
  const response = await fetch(`${apiBaseUrl}/api/notes?${params.toString()}`);

  if (!response.ok) {
    throw new Error(await readError(response, "读取历史笔记失败"));
  }

  const data = (await response.json()) as { items: Note[] };
  return data.items;
}

export async function translateText(
  apiBaseUrl: string,
  payload: {
    profile_id: number;
    site_key: string;
    source_text: string;
    source_lang?: string;
    target_lang: string;
  }
): Promise<Translation> {
  const response = await fetch(`${apiBaseUrl}/api/ai/translate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(await readError(response, "翻译失败"));
  }

  return (await response.json()) as Translation;
}

export async function chatWithAI(
  apiBaseUrl: string,
  payload: {
    profile_id: number;
    site_key: string;
    survey_id?: number;
    scope: string;
    page_text: string;
    question_text?: string;
    options_text?: string;
    user_message: string;
  }
): Promise<AIConversation> {
  const response = await fetch(`${apiBaseUrl}/api/ai/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(await readError(response, "AI 对话失败"));
  }

  return (await response.json()) as AIConversation;
}

export async function listAIConversations(apiBaseUrl: string, profileId: number, siteKey: string): Promise<AIConversation[]> {
  const params = new URLSearchParams({
    profile_id: String(profileId),
    site_key: siteKey,
    limit: "20"
  });
  const response = await fetch(`${apiBaseUrl}/api/ai/conversations?${params.toString()}`);

  if (!response.ok) {
    throw new Error(await readError(response, "读取 AI 对话失败"));
  }

  const data = (await response.json()) as { items: AIConversation[] };
  return data.items;
}

export async function searchKnowledge(
  apiBaseUrl: string,
  params: { profile_id: number; site_key: string; q: string; scope?: string; survey_id?: number }
): Promise<KnowledgeChunk[]> {
  const query = new URLSearchParams({
    profile_id: String(params.profile_id),
    site_key: params.site_key,
    q: params.q,
    limit: "20"
  });
  if (params.scope) query.set("scope", params.scope);
  if (params.survey_id) query.set("survey_id", String(params.survey_id));

  const response = await fetch(`${apiBaseUrl}/api/knowledge/search?${query.toString()}`);
  if (!response.ok) {
    throw new Error(await readError(response, "搜索知识库失败"));
  }

  const data = (await response.json()) as { items: KnowledgeChunk[] };
  return data.items;
}

async function readError(response: Response, fallback: string): Promise<string> {
  try {
    const data = (await response.json()) as { error?: string };
    return data.error || fallback;
  } catch {
    return fallback;
  }
}
