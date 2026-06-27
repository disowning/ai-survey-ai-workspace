import type { ResourceConfig } from "./types";

export const resources: ResourceConfig[] = [
  {
    key: "profiles",
    label: "Profile 管理",
    endpoint: "/api/profiles",
    filters: [],
    columns: ["id", "profile_key", "profile_name", "status", "remark", "created_at"]
  },
  {
    key: "sites",
    label: "问卷网站管理",
    endpoint: "/api/sites",
    filters: ["site_key"],
    columns: ["id", "site_key", "site_name", "domain", "created_at"]
  },
  {
    key: "surveys",
    label: "问卷项目记录",
    endpoint: "/api/surveys",
    filters: ["profile_id", "site_key"],
    columns: ["id", "profile_id", "site_key", "survey_title", "status", "created_at"]
  },
  {
    key: "snapshots",
    label: "页面快照记录",
    endpoint: "/api/page-snapshots",
    filters: ["profile_id", "site_key", "survey_id"],
    columns: ["id", "profile_id", "site_key", "survey_id", "page_title", "question_text", "created_at"]
  },
  {
    key: "notes",
    label: "笔记记录",
    endpoint: "/api/notes",
    filters: ["profile_id", "site_key", "survey_id"],
    columns: ["id", "profile_id", "site_key", "survey_id", "note_text", "created_at"]
  },
  {
    key: "translations",
    label: "翻译记录",
    endpoint: "/api/translations",
    filters: ["profile_id", "site_key"],
    columns: ["id", "profile_id", "site_key", "source_lang", "target_lang", "translated_text", "created_at"]
  },
  {
    key: "conversations",
    label: "AI 对话记录",
    endpoint: "/api/ai/conversations",
    filters: ["profile_id", "site_key", "survey_id"],
    columns: ["id", "profile_id", "site_key", "survey_id", "user_message", "ai_message", "created_at"]
  },
  {
    key: "answers",
    label: "答题库",
    endpoint: "/api/answers",
    filters: ["profile_id", "site_key", "survey_id"],
    columns: ["id", "profile_id", "site_key", "survey_id", "question_text", "final_answer", "reason", "created_at"]
  },
  {
    key: "knowledge",
    label: "知识库管理",
    endpoint: "/api/knowledge",
    filters: ["profile_id", "site_key", "survey_id", "scope"],
    columns: ["id", "profile_id", "site_key", "survey_id", "scope", "source_type", "chunk_text", "created_at"]
  },
  {
    key: "personas",
    label: "站点人设库",
    endpoint: "/api/personas",
    filters: ["profile_id", "site_key", "category"],
    columns: ["id", "profile_id", "site_key", "category", "persona_key", "persona_value", "confidence", "source_type", "updated_at"]
  }
];
