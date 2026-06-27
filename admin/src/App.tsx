import { useEffect, useMemo, useState } from "react";
import {
  Bot,
  ChevronLeft,
  ChevronRight,
  ClipboardList,
  Copy,
  Database,
  Download,
  FileStack,
  FileText,
  Languages,
  MessageSquareText,
  Network,
  NotebookText,
  RefreshCw,
  RotateCcw,
  Search,
  ShieldCheck,
  Trash2,
  UserRound,
  XCircle
} from "lucide-react";
import { deleteKnowledge, downloadExport, listResource, login, setAuthToken } from "./api";
import { resources } from "./resources";
import type { AnyRecord, ResourceConfig, ResourceKey } from "./types";

const defaultApiBaseUrl = inferDefaultApiBaseUrl();
const storageKeys = {
  apiBaseUrl: "surveyAiAdmin.apiBaseUrl",
  authToken: "surveyAiAdmin.authToken",
  authUsername: "surveyAiAdmin.authUsername"
} as const;

export function App() {
  const [apiBaseUrl, setApiBaseUrl] = useState(defaultApiBaseUrl);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loggedIn, setLoggedIn] = useState(false);
  const [activeKey, setActiveKey] = useState<ResourceKey>("profiles");
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [rows, setRows] = useState<AnyRecord[]>([]);
  const [selected, setSelected] = useState<AnyRecord | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(100);
  const [offset, setOffset] = useState(0);
  const [copied, setCopied] = useState(false);

  const active = useMemo(() => resources.find((item) => item.key === activeKey) ?? resources[0], [activeKey]);
  const trimmedApiBaseUrl = apiBaseUrl.replace(/\/+$/, "");
  const visibleRows = useMemo(() => rows.filter((row) => rowMatchesQuery(row, query)), [rows, query]);
  const tableColumns = useMemo(() => columnsForResource(active), [active]);

  useEffect(() => {
    const savedApiBaseUrl = localStorage.getItem(storageKeys.apiBaseUrl);
    const savedToken = localStorage.getItem(storageKeys.authToken);
    const savedUsername = localStorage.getItem(storageKeys.authUsername);
    if (savedApiBaseUrl) setApiBaseUrl(savedApiBaseUrl);
    if (savedUsername) setUsername(savedUsername);
    if (savedToken) {
      setAuthToken(savedToken);
      setLoggedIn(true);
    }
  }, []);

  useEffect(() => {
    if (!loggedIn) return;
    setOffset(0);
    setQuery("");
    void load(active, 0);
  }, [activeKey, loggedIn]);

  async function run(task: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await task();
    } catch (err) {
      setError(err instanceof Error ? err.message : "操作失败");
    } finally {
      setBusy(false);
    }
  }

  async function load(config = active, nextOffset = offset) {
    if (!loggedIn) return;
    await run(async () => {
      const data = await listResource(trimmedApiBaseUrl, config, filters, { limit, offset: nextOffset });
      setRows(data);
      setSelected(data[0] ?? null);
      setOffset(nextOffset);
    });
  }

  async function handleLogin() {
    await run(async () => {
      const data = await login(trimmedApiBaseUrl, username.trim(), password);
      localStorage.setItem(storageKeys.apiBaseUrl, trimmedApiBaseUrl);
      localStorage.setItem(storageKeys.authUsername, username.trim());
      localStorage.setItem(storageKeys.authToken, data.token);
      setPassword("");
      setLoggedIn(true);
      const items = await listResource(trimmedApiBaseUrl, active, filters, { limit, offset: 0 });
      setRows(items);
      setSelected(items[0] ?? null);
      setOffset(0);
    });
  }

  function handleLogout() {
    localStorage.removeItem(storageKeys.authToken);
    setAuthToken("");
    setLoggedIn(false);
    setRows([]);
    setSelected(null);
  }

  async function handleDeleteKnowledge(row: AnyRecord) {
    const id = Number(row.id);
    if (!Number.isInteger(id) || id <= 0) return;
    await run(async () => {
      await deleteKnowledge(trimmedApiBaseUrl, id, filters);
      const data = await listResource(trimmedApiBaseUrl, active, filters, { limit, offset });
      setRows(data);
      setSelected(data[0] ?? null);
    });
  }

  function resetFilters() {
    setFilters({});
    setQuery("");
    setOffset(0);
    void load(active, 0);
  }

  function openExport(path: string, needsProfile: boolean) {
    if (needsProfile && !filters.profile_id?.trim()) {
      setError("请先输入 profile_id 再导出");
      return;
    }
    void run(async () => {
      const blob = await downloadExport(trimmedApiBaseUrl, path, filters);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = exportFilename(path);
      anchor.click();
      URL.revokeObjectURL(url);
    });
  }

  async function copySelected() {
    if (!selected) return;
    await navigator.clipboard.writeText(JSON.stringify(selected, null, 2));
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  }

  if (!loggedIn) {
    return (
      <main className="login-shell">
        <section className="login-card">
          <div className="login-brand">
            <span className="brand-mark">
              <ShieldCheck size={18} aria-hidden="true" />
            </span>
            <div>
              <h1>Survey AI Admin</h1>
              <p>管理员登录</p>
            </div>
          </div>

          <label>
            <span>API 地址</span>
            <input value={apiBaseUrl} onChange={(event) => setApiBaseUrl(event.target.value)} />
          </label>
          <label>
            <span>用户名</span>
            <input value={username} autoComplete="username" onChange={(event) => setUsername(event.target.value)} />
          </label>
          <label>
            <span>密码</span>
            <input
              value={password}
              type="password"
              autoComplete="current-password"
              onChange={(event) => setPassword(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && username.trim() && password) void handleLogin();
              }}
            />
          </label>

          {error ? <div className="error-banner">{error}</div> : null}

          <button className="login-submit" type="button" onClick={() => void handleLogin()} disabled={busy || !username.trim() || !password}>
            {busy ? "登录中" : "登录后台"}
          </button>
        </section>
      </main>
    );
  }

  return (
    <main className="admin-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">
            <ShieldCheck size={17} aria-hidden="true" />
          </span>
          <div>
            <h1>Survey AI Admin</h1>
            <p>问卷工作台后台</p>
          </div>
        </div>

        <nav>
          {resources.map((resource) => (
            <button
              key={resource.key}
              type="button"
              className={resource.key === activeKey ? "active" : ""}
              onClick={() => {
                setActiveKey(resource.key);
                setSelected(null);
              }}
            >
              {iconForResource(resource.key)}
              {resource.label}
            </button>
          ))}
        </nav>
      </aside>

      <section className="content">
        <header className="topbar">
          <div>
            <span className="eyebrow">Data Console</span>
            <h2>{active.label}</h2>
            <p>按 Profile、网站、问卷隔离查看数据</p>
          </div>

          <div className="topbar-tools compact-tools">
            <div className="status-row">
              <span>{rows.length} 条已加载</span>
              <span>{visibleRows.length} 条可见</span>
              <span>{busy ? "同步中" : "已就绪"}</span>
              <span>{username}</span>
            </div>

            <div className="export-buttons">
              <button type="button" onClick={() => openExport("/api/export/markdown", true)}>
                <FileText size={15} aria-hidden="true" />
                Markdown
              </button>
              <button type="button" onClick={() => openExport("/api/export/notes.csv", true)}>
                <Download size={15} aria-hidden="true" />
                CSV
              </button>
              <button type="button" onClick={() => openExport("/api/export/daily-report", false)}>
                <FileText size={15} aria-hidden="true" />
                日报
              </button>
              <button type="button" onClick={handleLogout}>
                退出
              </button>
            </div>
          </div>
        </header>

        <FilterBar
          active={active}
          filters={filters}
          query={query}
          limit={limit}
          offset={offset}
          busy={busy}
          onChange={setFilters}
          onQueryChange={setQuery}
          onLimitChange={setLimit}
          onSearch={() => load(active, 0)}
          onReset={resetFilters}
          onPrev={() => load(active, Math.max(0, offset - limit))}
          onNext={() => load(active, offset + limit)}
        />

        {error ? <div className="error-banner">{error}</div> : null}

        <div className="workspace">
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  {tableColumns.map((column) => (
                    <th key={column}>{columnLabel(column)}</th>
                  ))}
                  {active.key === "knowledge" ? <th>操作</th> : null}
                </tr>
              </thead>
              <tbody>
                {visibleRows.map((row) => (
                  <tr key={String(row.id ?? JSON.stringify(row))} className={selected === row ? "selected" : ""} onClick={() => setSelected(row)}>
                    {tableColumns.map((column) => (
                      <td key={column} className={`column-${column.replace(/_/g, "-")}`}>
                        {formatResourceCell(active.key, column, row[column], row)}
                      </td>
                    ))}
                    {active.key === "knowledge" ? (
                      <td>
                        <button className="icon-button danger" type="button" onClick={() => void handleDeleteKnowledge(row)}>
                          <Trash2 size={15} aria-hidden="true" />
                        </button>
                      </td>
                    ) : null}
                  </tr>
                ))}
                {!visibleRows.length ? (
                  <tr>
                    <td colSpan={tableColumns.length + (active.key === "knowledge" ? 1 : 0)} className="empty">
                      暂无数据
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>

          <aside className="detail">
            <div className="detail-heading">
              <h3>详情</h3>
              <span>{selected?.id ? `ID ${selected.id}` : "未选择"}</span>
            </div>

            {selected ? (
              <>
                <div className="detail-actions">
                  <button type="button" onClick={() => void copySelected()}>
                    <Copy size={15} aria-hidden="true" />
                    {copied ? "已复制" : "复制 JSON"}
                  </button>
                </div>
                <RecordDetail resource={active.key} row={selected} />
              </>
            ) : (
              <pre>未选择记录</pre>
            )}
          </aside>
        </div>
      </section>
    </main>
  );
}

function iconForResource(key: ResourceKey) {
  const props = { size: 16, "aria-hidden": true as const };
  switch (key) {
    case "profiles":
      return <UserRound {...props} />;
    case "sites":
      return <Network {...props} />;
    case "surveys":
      return <ClipboardList {...props} />;
    case "snapshots":
      return <FileStack {...props} />;
    case "notes":
      return <NotebookText {...props} />;
    case "translations":
      return <Languages {...props} />;
    case "conversations":
      return <MessageSquareText {...props} />;
    case "knowledge":
      return <Bot {...props} />;
    default:
      return <Database {...props} />;
  }
}

function RecordDetail({ resource, row }: { resource: ResourceKey; row: AnyRecord }) {
  if (resource === "knowledge") return <KnowledgeDetail row={row} />;

  const sections = detailSections(resource, row);
  return (
    <div className="record-detail">
      {sections.length ? (
        <div className="detail-content-stack">
          {sections.map((section) => (
            <section className="detail-content-card" key={section.label}>
              <span>{section.label}</span>
              <p>{section.value || "-"}</p>
            </section>
          ))}
        </div>
      ) : null}

      <dl className="detail-list compact-detail-list">
        {detailMetaKeys(resource, row).map((key) => (
          <div key={key}>
            <dt>{columnLabel(key)}</dt>
            <dd>{formatDetailValue(key, row[key])}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

function detailSections(resource: ResourceKey, row: AnyRecord): Array<{ label: string; value: string }> {
  switch (resource) {
    case "profiles":
      return [{ label: "Profile", value: [row.profile_name, row.profile_key].filter(Boolean).join(" / ") }];
    case "sites":
      return [{ label: "网站", value: [row.site_name, row.site_key, row.domain].filter(Boolean).join(" / ") }];
    case "surveys":
      return [
        { label: "问卷标题", value: String(row.survey_title ?? "") },
        { label: "问卷链接", value: String(row.survey_url ?? "") }
      ];
    case "snapshots":
      return [
        { label: "题目", value: String(row.question_text ?? "") },
        { label: "选项", value: String(row.options_text ?? "") },
        { label: "页面全文", value: String(row.page_text ?? "") }
      ];
    case "notes":
      return [{ label: "笔记内容", value: String(row.note_text ?? "") }];
    case "translations":
      return [
        { label: "原文", value: String(row.source_text ?? "") },
        { label: "译文", value: String(row.translated_text ?? "") }
      ];
    case "conversations":
      return [
        { label: "用户问题", value: String(row.user_message ?? "") },
        { label: "AI 回复", value: String(row.ai_message ?? "") }
      ];
    default:
      return [];
  }
}

function detailMetaKeys(resource: ResourceKey, row: AnyRecord): string[] {
  const keys: Partial<Record<ResourceKey, string[]>> = {
    profiles: ["id", "profile_key", "profile_name", "status", "remark", "created_at", "updated_at"],
    sites: ["id", "site_key", "site_name", "domain", "created_at"],
    surveys: ["id", "profile_id", "site_key", "status", "created_at", "updated_at"],
    snapshots: ["id", "profile_id", "site_key", "survey_id", "url", "page_title", "language", "created_at"],
    notes: ["id", "profile_id", "site_key", "survey_id", "page_snapshot_id", "created_at"],
    translations: ["id", "profile_id", "site_key", "source_lang", "target_lang", "created_at"],
    conversations: ["id", "profile_id", "site_key", "survey_id", "created_at"]
  };
  return (keys[resource] ?? Object.keys(row)).filter((key) => key in row);
}

function KnowledgeDetail({ row }: { row: AnyRecord }) {
  return (
    <div className="knowledge-detail">
      <section className="knowledge-content">
        <span>知识内容</span>
        <p>{cleanKnowledgeText(String(row.chunk_text ?? "")) || "-"}</p>
      </section>

      <div className="knowledge-meta-grid">
        <MetaItem label="Profile" value={formatCell(row.profile_id)} />
        <MetaItem label="网站" value={formatCell(row.site_key)} />
        <MetaItem label="问卷" value={formatCell(row.survey_id) || "-"} />
        <MetaItem label="范围" value={scopeLabel(row.scope)} />
        <MetaItem label="来源" value={sourceTypeLabel(row.source_type)} />
        <MetaItem label="来源 ID" value={formatCell(row.source_id) || "-"} />
        <MetaItem label="创建时间" value={formatDateTime(row.created_at)} />
      </div>
    </div>
  );
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function FilterBar({
  active,
  filters,
  query,
  limit,
  offset,
  busy,
  onChange,
  onQueryChange,
  onLimitChange,
  onSearch,
  onReset,
  onPrev,
  onNext
}: {
  active: ResourceConfig;
  filters: Record<string, string>;
  query: string;
  limit: number;
  offset: number;
  busy: boolean;
  onChange: (filters: Record<string, string>) => void;
  onQueryChange: (query: string) => void;
  onLimitChange: (limit: number) => void;
  onSearch: () => void;
  onReset: () => void;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <section className="filters">
      {active.filters.map((filter) => (
        <label key={filter}>
          <span>{columnLabel(filter)}</span>
          <input value={filters[filter] ?? ""} onChange={(event) => onChange({ ...filters, [filter]: event.target.value })} />
        </label>
      ))}

      <label>
        <span>页面内搜索</span>
        <input value={query} placeholder="搜索已加载记录" onChange={(event) => onQueryChange(event.target.value)} />
      </label>

      <label className="small-input">
        <span>limit</span>
        <input
          value={limit}
          inputMode="numeric"
          onChange={(event) => {
            const next = Number(event.target.value);
            if (Number.isInteger(next) && next > 0 && next <= 200) onLimitChange(next);
          }}
        />
      </label>

      <div className="filter-actions">
        <button type="button" onClick={onSearch} disabled={busy}>
          {busy ? <RefreshCw size={16} aria-hidden="true" /> : <Search size={16} aria-hidden="true" />}
          查询
        </button>
        <button className="secondary" type="button" onClick={onReset} disabled={busy}>
          <XCircle size={16} aria-hidden="true" />
          清空
        </button>
        <button className="secondary icon-only" type="button" onClick={onPrev} disabled={busy || offset === 0} title="上一页">
          <ChevronLeft size={16} aria-hidden="true" />
        </button>
        <button className="secondary icon-only" type="button" onClick={onNext} disabled={busy} title="下一页">
          <ChevronRight size={16} aria-hidden="true" />
        </button>
        <button className="secondary icon-only" type="button" onClick={onSearch} disabled={busy} title="刷新">
          <RotateCcw size={16} aria-hidden="true" />
        </button>
      </div>
    </section>
  );
}

function rowMatchesQuery(row: AnyRecord, query: string): boolean {
  const value = query.trim().toLowerCase();
  if (!value) return true;
  return JSON.stringify(row).toLowerCase().includes(value);
}

function columnsForResource(active: ResourceConfig): string[] {
  const columns: Partial<Record<ResourceKey, string[]>> = {
    profiles: ["id", "profile_key", "profile_name", "status", "remark", "created_at"],
    sites: ["site_key", "site_name", "domain", "created_at"],
    surveys: ["id", "profile_id", "site_key", "survey_title", "status", "created_at"],
    snapshots: ["id", "profile_id", "site_key", "page_title", "question_text", "created_at"],
    notes: ["id", "profile_id", "site_key", "survey_id", "note_text", "created_at"],
    translations: ["id", "profile_id", "site_key", "source_lang", "target_lang", "translated_text", "created_at"],
    conversations: ["id", "profile_id", "site_key", "user_message", "ai_message", "created_at"],
    knowledge: ["id", "profile_id", "site_key", "scope", "source_type", "chunk_text", "created_at"]
  };
  return columns[active.key] ?? active.columns;
}

function columnLabel(column: string): string {
  const labels: Record<string, string> = {
    id: "ID",
    profile_id: "Profile",
    profile_key: "Profile Key",
    profile_name: "Profile 名称",
    site_key: "网站",
    site_name: "网站名称",
    survey_id: "问卷",
    survey_title: "问卷标题",
    scope: "范围",
    source_type: "来源",
    source_id: "来源 ID",
    chunk_text: "内容",
    note_text: "笔记",
    user_message: "用户问题",
    ai_message: "AI 回复",
    translated_text: "译文",
    created_at: "时间",
    status: "状态",
    remark: "备注"
  };
  const overrides: Record<string, string> = {
    domain: "域名",
    survey_url: "问卷链接",
    source_text: "原文",
    source_lang: "原语言",
    target_lang: "目标语言",
    page_title: "页面标题",
    question_text: "题目",
    options_text: "选项",
    page_text: "页面全文",
    language: "语言",
    url: "链接",
    updated_at: "更新时间"
  };
  return overrides[column] ?? labels[column] ?? column;
}

function formatResourceCell(resource: ResourceKey, column: string, value: unknown, row: AnyRecord): string {
  if (column === "chunk_text") return previewText(cleanKnowledgeText(String(value ?? "")), 180);
  if (["note_text", "translated_text", "user_message", "ai_message", "question_text", "page_text", "remark"].includes(column)) {
    return previewText(String(value ?? ""), resource === "snapshots" ? 150 : 180);
  }
  if (column === "scope") return scopeLabel(value);
  if (column === "source_type") return sourceTypeLabel(value);
  if (column === "created_at" || column === "updated_at") return formatDateTime(value);
  if (column === "profile_id") return value ? `Profile ${String(value)}` : "-";
  if (column === "site_key") return String(value || row.site_name || "-");
  if (column === "survey_id") return value ? String(value) : "-";
  if (column === "status") return statusLabel(value);
  if (column === "source_lang" || column === "target_lang" || column === "language") return String(value || "-");
  return formatCell(value);
}

function statusLabel(value: unknown): string {
  const key = String(value ?? "");
  if (key === "active") return "正常";
  if (key === "disabled") return "已禁用";
  if (key === "archived") return "已归档";
  return key || "-";
}

function cleanKnowledgeText(value: string): string {
  return value
    .replace(/^User:\s*/i, "用户：")
    .replace(/\sAI:\s*/i, "\n\nAI：")
    .trim();
}

function previewText(value: string, maxLength: number): string {
  const normalized = value.replace(/\s+/g, " ").trim();
  if (!normalized) return "-";
  return normalized.length > maxLength ? `${normalized.slice(0, maxLength)}...` : normalized;
}

function scopeLabel(value: unknown): string {
  const key = String(value ?? "");
  if (key === "survey") return "问卷";
  if (key === "site") return "网站";
  if (key === "profile") return "Profile";
  if (key === "team") return "团队";
  return key || "-";
}

function sourceTypeLabel(value: unknown): string {
  const key = String(value ?? "");
  if (key === "ai_conversation") return "AI 对话";
  if (key === "note") return "笔记";
  if (key === "page_snapshot") return "页面快照";
  if (key === "manual") return "手动录入";
  return key || "-";
}

function formatDateTime(value: unknown): string {
  if (!value) return "-";
  const date = new Date(String(value));
  if (Number.isNaN(date.getTime())) return String(value);
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  });
}

function formatCell(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value.length > 140 ? `${value.slice(0, 140)}...` : value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}

function formatDetail(value: unknown): string {
  if (value === null || value === undefined) return "-";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value, null, 2);
}

function formatDetailValue(key: string, value: unknown): string {
  if (key === "created_at" || key === "updated_at") return formatDateTime(value);
  if (key === "status") return statusLabel(value);
  if (key === "scope") return scopeLabel(value);
  if (key === "source_type") return sourceTypeLabel(value);
  if (key === "profile_id") return value ? `Profile ${String(value)}` : "-";
  if (key === "survey_id" || key === "page_snapshot_id") return value ? String(value) : "-";
  return formatDetail(value);
}

function exportFilename(path: string): string {
  if (path.includes("notes.csv")) return "survey-notes.csv";
  if (path.includes("daily-report")) return "survey-daily-report.md";
  return "survey-export.md";
}

function inferDefaultApiBaseUrl(): string {
  if (typeof window === "undefined") return "http://localhost:8080";
  const { protocol, hostname } = window.location;
  if (hostname.includes("-admin.")) return `${protocol}//${hostname.replace("-admin.", "-api.")}`;
  if (hostname.startsWith("admin.")) return `${protocol}//api.${hostname.slice("admin.".length)}`;
  if (hostname === "localhost" || hostname === "127.0.0.1") return "http://localhost:8080";
  return `${protocol}//${hostname}`;
}
