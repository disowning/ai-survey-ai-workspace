export type ResourceKey =
  | "profiles"
  | "sites"
  | "surveys"
  | "snapshots"
  | "notes"
  | "translations"
  | "conversations"
  | "knowledge"
  | "personas";

export type ResourceConfig = {
  key: ResourceKey;
  label: string;
  endpoint: string;
  columns: string[];
  filters: Array<"profile_id" | "site_key" | "survey_id" | "scope" | "category">;
};

export type ApiListResponse<T> = {
  items: T[];
};

export type AnyRecord = Record<string, unknown>;
