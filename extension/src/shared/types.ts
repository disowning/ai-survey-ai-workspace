export type ExtractedPage = {
  url: string;
  domain: string;
  title: string;
  pageText: string;
  questionText?: string;
  optionsText?: string;
  language?: string;
  extractedAt: string;
};

export type SiteDetection = {
  site_key: string;
  site_name?: string;
  domain?: string;
};

export type LocalProfile = {
  profileId: number | null;
  profileKey: string;
  profileName: string;
};

export type ExtensionSettings = {
  apiBaseUrl: string;
  authToken?: string;
  authExpiresAt?: string;
  authUsername?: string;
};

export type SnapshotPayload = {
  profile_id: number;
  site_key: string;
  survey_id?: number;
  url: string;
  page_title: string;
  question_text?: string;
  options_text?: string;
  page_text: string;
  language?: string;
};

export type Note = {
  id: number;
  profile_id: number;
  site_key: string;
  survey_id?: number;
  page_snapshot_id?: number;
  note_text: string;
  created_at: string;
  updated_at: string;
};

export type Translation = {
  id: number;
  profile_id: number;
  site_key: string;
  source_text: string;
  translated_text: string;
  source_lang?: string;
  target_lang: string;
  created_at: string;
};

export type AIConversation = {
  id: number;
  profile_id: number;
  site_key: string;
  survey_id?: number;
  user_message: string;
  ai_message: string;
  created_at: string;
};

export type KnowledgeChunk = {
  id: number;
  profile_id: number;
  site_key: string;
  survey_id?: number;
  scope: string;
  source_type: string;
  source_id?: number;
  chunk_text: string;
  created_at: string;
};
