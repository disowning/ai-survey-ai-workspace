import { type KeyboardEvent, useEffect, useMemo, useRef, useState } from "react";
import {
  ArrowUp,
  Bookmark,
  ChevronDown,
  Copy,
  Plus,
  RefreshCcw,
  SlidersHorizontal,
  Sparkles
} from "lucide-react";
import {
  bindProfile,
  checkHealth,
  chatWithAI,
  createNote,
  detectSite,
  ensureSite,
  listAIConversations,
  listNotes,
  login,
  saveSnapshot,
  searchKnowledge,
  setAuthToken,
  translateText
} from "./api";
import type { SystemStatus } from "./api";
import { extractFromActiveTab, getSelectionFromActiveTab } from "./chromeTabs";
import { DEFAULT_API_BASE_URL } from "../shared/constants";
import type { AIConversation, ExtractedPage, KnowledgeChunk, LocalProfile, Note, SiteDetection } from "../shared/types";
import { getLastSite, getLocalProfile, getSettings, setLastSite, setLocalProfile, setSettings } from "../storage/localProfile";

type Message = {
  id: string;
  role: "user" | "assistant" | "system";
  title?: string;
  text: string;
  meta?: string;
  actions?: Array<"save-note" | "copy" | "continue">;
};

type Command = {
  id: string;
  icon: string;
  command: string;
  target: string;
  detail: string;
  placeholder?: string;
};

type ScopeMode = "auto" | "survey" | "site" | "profile";
type ApiStatus = "checking" | "online" | "ai-missing" | "db-offline" | "offline";

const suggestionPills = [
  { id: "summary", text: "总结当前问卷页面？", command: "/总结" },
  { id: "explain", text: "这道题目的核心考点是什么？", command: "/解释" },
  { id: "translate", text: "翻译当前题目和选项", command: "/翻译" }
];

const commands: Command[] = [
  { id: "summary", icon: "📝", command: "/总结", target: "当前页面", detail: "生成页面摘要" },
  { id: "explain", icon: "💡", command: "/解释", target: "当前题目", detail: "说明题意和选项" },
  { id: "choose", icon: "✨", command: "/帮我选择", target: "基于真实情况", detail: "辅助判断选项" },
  { id: "translate-page", icon: "🌐", command: "/翻译", target: "当前题目", detail: "题目和选项译成中文" },
  { id: "translate-full", icon: "📄", command: "/翻译全文", target: "当前页", detail: "全文译成中文" },
  { id: "translate-selection", icon: "🅰️", command: "/翻译选中", target: "", detail: "翻译划词内容" },
  { id: "note", icon: "📌", command: "/保存笔记", target: "", detail: "写入当前站点", placeholder: "笔记内容" },
  { id: "similar", icon: "🔎", command: "/相似题", target: "", detail: "查找相似历史题" },
  { id: "search", icon: "🔍", command: "/搜索知识库", target: "", detail: "查阅过往记录", placeholder: "关键词" },
  { id: "snapshot", icon: "📷", command: "/保存快照", target: "", detail: "记录当前页" },
  { id: "notes", icon: "📚", command: "/历史笔记", target: "", detail: "查看站点笔记" },
  { id: "chats", icon: "💬", command: "/历史对话", target: "", detail: "查看 AI 对话" },
  { id: "bind", icon: "🔗", command: "/绑定", target: "profile-001", detail: "绑定当前环境", placeholder: "profile-001" },
  { id: "survey", icon: "◽", command: "/问卷", target: "123", detail: "设置问卷 ID", placeholder: "123" }
];

const scopeModes: Array<{ value: ScopeMode; label: string }> = [
  { value: "auto", label: "自动" },
  { value: "survey", label: "问卷" },
  { value: "site", label: "网站" },
  { value: "profile", label: "Profile" }
];

const emptyProfile: LocalProfile = {
  profileId: null,
  profileKey: "",
  profileName: ""
};

export function App() {
  const [profile, setProfile] = useState<LocalProfile>(emptyProfile);
  const [apiBaseUrl, setApiBaseUrl] = useState(DEFAULT_API_BASE_URL);
  const [site, setSite] = useState<SiteDetection | null>(null);
  const [page, setPage] = useState<ExtractedPage | null>(null);
  const [surveyId, setSurveyId] = useState("");
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [busy, setBusy] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [showAdvancedSettings, setShowAdvancedSettings] = useState(false);
  const [scopeMode, setScopeMode] = useState<ScopeMode>("auto");
  const [apiStatus, setApiStatus] = useState<ApiStatus>("checking");
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [profileDraft, setProfileDraft] = useState("profile-001");
  const [apiDraft, setApiDraft] = useState(DEFAULT_API_BASE_URL);
  const [usernameDraft, setUsernameDraft] = useState("");
  const [passwordDraft, setPasswordDraft] = useState("");
  const [authToken, setAuthTokenState] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const autoSnapshotKeysRef = useRef<Set<string>>(new Set());

  const trimmedApiBaseUrl = useMemo(() => apiBaseUrl.replace(/\/+$/, ""), [apiBaseUrl]);
  const showCommands = input.trimStart().startsWith("/");
  const filteredCommands = useMemo(() => filterCommands(input), [input]);
  const currentScope = useMemo(() => {
    if (scopeMode !== "auto") return scopeMode;
    return surveyId.trim() ? "survey" : "site";
  }, [scopeMode, surveyId]);
  const scopeLabel = scopeModes.find((item) => item.value === scopeMode)?.label ?? "自动";

  useEffect(() => {
    void (async () => {
      const [savedProfile, savedSettings, savedSite] = await Promise.all([getLocalProfile(), getSettings(), getLastSite()]);
      setProfile(savedProfile);
      setProfileDraft(savedProfile.profileKey || "profile-001");
      setApiBaseUrl(savedSettings.apiBaseUrl);
      setApiDraft(savedSettings.apiBaseUrl);
      setUsernameDraft(savedSettings.authUsername || "");
      setAuthTokenState(savedSettings.authToken || "");
      setAuthToken(savedSettings.authToken || "");
      setSite(savedSite);
      await refreshApiStatus(savedSettings.apiBaseUrl.replace(/\/+$/, ""));
      await refreshPageContext(savedSettings.apiBaseUrl.replace(/\/+$/, ""), false);
    })();
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, busy]);

  useEffect(() => {
    if (!textareaRef.current) return;
    textareaRef.current.style.height = "auto";
    textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 120)}px`;
  }, [input]);

  useEffect(() => {
    if (showCommands) setSelectedIndex(0);
  }, [showCommands, input]);

  function addMessage(message: Omit<Message, "id">) {
    if (message.role === "system" && !isImportantSystemMessage(message.text)) return;
    const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
    setMessages((current) => [...current, { id, ...message }].slice(-40));
  }

  async function runTask(task: () => Promise<void>) {
    setBusy(true);
    try {
      await task();
    } catch (error) {
      addMessage({
        role: "system",
        title: "未完成",
        text: error instanceof Error ? error.message : "操作失败"
      });
    } finally {
      setBusy(false);
    }
  }

  async function refreshApiStatus(baseUrl = trimmedApiBaseUrl) {
    setApiStatus("checking");
    const status = await checkHealth(baseUrl.replace(/\/+$/, ""));
    setSystemStatus(status);
    if (!status) {
      setApiStatus("offline");
      return false;
    }
    if (!status.database_ok) {
      setApiStatus("db-offline");
      return false;
    }
    if (!status.ai_configured) {
      setApiStatus("ai-missing");
      return true;
    }
    setApiStatus("online");
    return true;
  }

  async function refreshPageContext(baseUrl = trimmedApiBaseUrl, notify = true) {
    try {
      const extracted = await extractFromActiveTab();
      setPage(extracted);
      const detected = await detectSite(baseUrl, extracted.url);
      setSite(detected.site);
      await setLastSite(detected.site);
      void autoSaveQuestionSnapshot(baseUrl, extracted, detected.site);
      if (notify) {
        addMessage({
          role: "system",
          title: "页面已识别",
          text: `${extracted.title}\n${detected.site.site_name || detected.site.site_key}`
        });
      }
    } catch {
      if (notify) {
        addMessage({ role: "system", title: "页面未识别", text: "当前页面暂时无法提取，打开问卷页后再试。" });
      }
    }
  }

  async function ensurePage(): Promise<ExtractedPage> {
    const extracted = page ?? (await extractFromActiveTab());
    if (!page) setPage(extracted);
    return extracted;
  }

  async function ensureProfile(): Promise<LocalProfile> {
    if (!profile.profileId) {
      throw new Error("请先输入 /绑定 profile-001，或在设置中绑定 Profile");
    }
    return profile;
  }

  async function ensureCurrentSite(targetPage: ExtractedPage): Promise<SiteDetection> {
    const detected = site ?? (await detectSite(trimmedApiBaseUrl, targetPage.url)).site;
    const persisted = await ensureSite(trimmedApiBaseUrl, detected);
    setSite(persisted);
    await setLastSite(persisted);
    return persisted;
  }

  function currentSurveyId(): number | undefined {
    if (!surveyId.trim()) return undefined;
    const parsed = Number(surveyId);
    if (!Number.isInteger(parsed) || parsed <= 0) throw new Error("问卷 ID 必须是正整数");
    return parsed;
  }

  async function context() {
    const [targetProfile, targetPage] = await Promise.all([ensureProfile(), ensurePage()]);
    const targetSite = await ensureCurrentSite(targetPage);
    return { targetProfile, targetPage, targetSite, targetSurveyId: currentSurveyId() };
  }

  async function submitInput() {
    const value = input.trim();
    if (!value || busy) return;
    setInput("");
    addMessage({ role: "user", text: value });
    await runTask(async () => {
      if (value.startsWith("/")) {
        await executeCommand(value);
      } else {
        await askAI(value, "回复");
      }
    });
  }

  async function executeCommand(raw: string) {
    const parsed = parseCommand(raw);
    const args = parsed.args.trim();

    switch (parsed.command) {
      case "/绑定":
        await bindByCommand(args || profileDraft || "profile-001");
        return;
      case "/问卷":
        if (!args) throw new Error("请输入问卷 ID，例如 /问卷 123");
        setSurveyId(args);
        addMessage({ role: "system", title: "已设置问卷", text: `当前问卷 ID：${args}` });
        return;
      case "/总结":
        await askAI("请总结当前问卷页面：页面在问什么、有哪些题目/选项、需要注意哪些条件。不要自动提交，不要编造用户资料。", "总结");
        return;
      case "/解释":
        await askAI("请解释当前题目和选项含义：题目意思、每个选项含义、容易误解的词、是否像筛选题。不要编造用户资料。", "解释");
        return;
      case "/帮我选择":
        await askAI("请基于我已经提供的真实情况、历史笔记和当前页面，帮我判断更合适的选项。先解释题目和选项；如果缺少真实信息，请明确说需要我补充；如果可以建议，请给出“建议选择”和理由，并提醒我最终确认。不要编造身份或经历，不要自动提交。", "辅助选择");
        return;
      case "/翻译":
        await translatePage();
        return;
      case "/翻译全文":
        await translateFullPage();
        return;
      case "/翻译选中":
        await translateSelection();
        return;
      case "/保存笔记":
        await saveNote(args);
        return;
      case "/搜索知识库":
      case "/搜索历史":
      case "/相似题":
        await searchKB(args);
        return;
      case "/保存快照":
        await snapshotPage();
        return;
      case "/历史笔记":
        await loadNotes();
        return;
      case "/历史对话":
        await loadChats();
        return;
      default:
        throw new Error("未知命令。输入 / 查看可用命令");
    }
  }

  async function askAI(userMessage: string, title: string) {
    const { targetProfile, targetPage, targetSite, targetSurveyId } = await context();
    const response = await chatWithAI(trimmedApiBaseUrl, {
      profile_id: targetProfile.profileId!,
      site_key: targetSite.site_key,
      survey_id: targetSurveyId,
      scope: currentScope,
      page_text: targetPage.pageText,
      question_text: targetPage.questionText,
      options_text: targetPage.optionsText,
      progress_text: targetPage.progressText,
      question_type: targetPage.questionType,
      extraction_confidence: targetPage.extractionConfidence,
      extracted_blocks: targetPage.extractedBlocks,
      user_message: userMessage
    });
    addMessage({
      role: "assistant",
      title,
      text: response.ai_message,
      meta: formatDate(response.created_at),
      actions: ["save-note", "copy", "continue"]
    });
  }

  async function translatePage() {
    const { targetProfile, targetPage, targetSite } = await context();
    const sourceText = surveyQuestionText(targetPage);
    const translated = await translateText(trimmedApiBaseUrl, {
      profile_id: targetProfile.profileId!,
      site_key: targetSite.site_key,
      source_text: sourceText,
      source_lang: targetPage.language,
      target_lang: "zh-CN"
    });
    addMessage({ role: "assistant", title: "翻译", text: translated.translated_text, meta: "已保存翻译记录", actions: ["save-note", "copy", "continue"] });
  }

  async function translateFullPage() {
    const { targetProfile, targetPage, targetSite } = await context();
    const translated = await translateText(trimmedApiBaseUrl, {
      profile_id: targetProfile.profileId!,
      site_key: targetSite.site_key,
      source_text: targetPage.pageText,
      source_lang: targetPage.language,
      target_lang: "zh-CN"
    });
    addMessage({ role: "assistant", title: "翻译全文", text: translated.translated_text, meta: "已保存翻译记录", actions: ["save-note", "copy", "continue"] });
  }

  async function translateSelection() {
    const selectedText = await getSelectionFromActiveTab();
    if (!selectedText) throw new Error("请先在页面中选中文本");
    const { targetProfile, targetPage, targetSite } = await context();
    const translated = await translateText(trimmedApiBaseUrl, {
      profile_id: targetProfile.profileId!,
      site_key: targetSite.site_key,
      source_text: selectedText,
      source_lang: targetPage.language,
      target_lang: "zh-CN"
    });
    addMessage({ role: "assistant", title: "翻译选中", text: translated.translated_text, meta: "已保存翻译记录", actions: ["save-note", "copy", "continue"] });
  }

  async function saveNote(noteText: string) {
    const { targetProfile, targetPage, targetSite, targetSurveyId } = await context();
    const text = noteText || targetPage.questionText || "";
    if (!text.trim()) throw new Error("请输入笔记内容，例如 /保存笔记 这题是购买频率问题");
    const note = await createNote(trimmedApiBaseUrl, {
      profile_id: targetProfile.profileId!,
      site_key: targetSite.site_key,
      survey_id: targetSurveyId,
      note_text: text.trim()
    });
    addMessage({ role: "system", title: "笔记已保存", text: note.note_text, meta: formatDate(note.created_at) });
  }

  async function searchKB(query: string) {
    const { targetProfile, targetPage, targetSite, targetSurveyId } = await context();
    const q = query || targetPage.questionText || targetPage.title;
    const results = await searchKnowledge(trimmedApiBaseUrl, {
      profile_id: targetProfile.profileId!,
      site_key: targetSite.site_key,
      survey_id: targetSurveyId,
      scope: currentScope,
      q
    });
    addMessage({ role: "assistant", title: `历史记录 · ${results.length} 条`, text: results.length ? formatKnowledge(results) : "没有找到相关记录。", actions: ["copy", "continue"] });
  }

  async function snapshotPage() {
    const { targetProfile, targetPage, targetSite, targetSurveyId } = await context();
    await saveSnapshot(trimmedApiBaseUrl, {
      profile_id: targetProfile.profileId!,
      site_key: targetSite.site_key,
      survey_id: targetSurveyId,
      url: targetPage.url,
      page_title: targetPage.title,
      question_text: targetPage.questionText,
      options_text: targetPage.optionsText,
      page_text: targetPage.pageText,
      language: targetPage.language
    });
    addMessage({ role: "system", title: "快照已保存", text: targetPage.title, meta: targetSite.site_key });
  }

  async function autoSaveQuestionSnapshot(baseUrl: string, targetPage: ExtractedPage, targetSite: SiteDetection) {
    if (!profile.profileId || !targetPage.questionText || (targetPage.extractionConfidence ?? 0) < 0.45) return;
    const key = `${profile.profileId}:${targetSite.site_key}:${targetPage.url}:${targetPage.questionText}`;
    if (autoSnapshotKeysRef.current.has(key)) return;
    autoSnapshotKeysRef.current.add(key);

    try {
      const persistedSite = await ensureSite(baseUrl, targetSite);
      await saveSnapshot(baseUrl, {
        profile_id: profile.profileId,
        site_key: persistedSite.site_key,
        survey_id: currentSurveyId(),
        url: targetPage.url,
        page_title: targetPage.title,
        question_text: targetPage.questionText,
        options_text: targetPage.optionsText,
        page_text: surveyQuestionText(targetPage),
        language: targetPage.language
      });
    } catch {
      autoSnapshotKeysRef.current.delete(key);
    }
  }

  async function loadNotes() {
    const { targetProfile, targetSite } = await context();
    const notes = await listNotes(trimmedApiBaseUrl, targetProfile.profileId!, targetSite.site_key);
    addMessage({ role: "assistant", title: `历史笔记 · ${notes.length} 条`, text: notes.length ? formatNotes(notes) : "当前站点暂无笔记。", actions: ["copy", "continue"] });
  }

  async function loadChats() {
    const { targetProfile, targetSite } = await context();
    const history = await listAIConversations(trimmedApiBaseUrl, targetProfile.profileId!, targetSite.site_key);
    addMessage({ role: "assistant", title: `AI 对话 · ${history.length} 条`, text: history.length ? formatConversations(history) : "当前站点暂无 AI 对话记录。", actions: ["copy", "continue"] });
  }

  async function bindByCommand(profileKey: string) {
    const key = profileKey.trim();
    if (!key) throw new Error("请输入 Profile Key，例如 /绑定 profile-001");
    const bound = await bindProfile(trimmedApiBaseUrl, key, key);
    setProfile(bound);
    setProfileDraft(bound.profileKey);
    await setLocalProfile(bound);
    addMessage({ role: "system", title: "Profile 已绑定", text: `${bound.profileName} · #${bound.profileId}` });
  }

  async function saveSettings() {
    await runTask(async () => {
      const nextBaseUrl = apiDraft.trim() || DEFAULT_API_BASE_URL;
      const normalizedBaseUrl = nextBaseUrl.replace(/\/+$/, "");
      let nextAuthToken = authToken;
      let nextAuthExpiresAt = "";

      if (usernameDraft.trim() && passwordDraft) {
        const auth = await login(normalizedBaseUrl, usernameDraft.trim(), passwordDraft);
        nextAuthToken = auth.token || "";
        nextAuthExpiresAt = auth.expires_at || "";
        setAuthTokenState(nextAuthToken);
        setAuthToken(nextAuthToken);
        setPasswordDraft("");
      } else {
        setAuthToken(nextAuthToken);
      }

      setApiBaseUrl(nextBaseUrl);
      await setSettings({
        apiBaseUrl: nextBaseUrl,
        authToken: nextAuthToken,
        authExpiresAt: nextAuthExpiresAt,
        authUsername: usernameDraft.trim()
      });
      const statusOK = await refreshApiStatus(normalizedBaseUrl);
      if (!statusOK) throw new Error("API 连接失败，请检查地址或反向代理");
      if (profileDraft.trim()) {
        const bound = await bindProfile(normalizedBaseUrl, profileDraft.trim(), profileDraft.trim());
        setProfile(bound);
        await setLocalProfile(bound);
      }
      setSettingsOpen(false);
      addMessage({ role: "system", title: "设置已保存", text: "Profile 和 API 地址已更新。" });
    });
  }

  async function handleMessageAction(action: "save-note" | "copy" | "continue", message: Message) {
    if (action === "continue") {
      setInput("继续解释 ");
      textareaRef.current?.focus();
      return;
    }
    if (action === "copy") {
      await navigator.clipboard.writeText(message.text);
      addMessage({ role: "system", title: "已复制", text: "内容已复制到剪贴板。" });
      return;
    }
    await runTask(async () => {
      await saveNote(message.text);
    });
  }

  function cycleScopeMode() {
    const index = scopeModes.findIndex((item) => item.value === scopeMode);
    const next = scopeModes[(index + 1) % scopeModes.length];
    setScopeMode(next.value);
    addMessage({ role: "system", title: "范围已切换", text: `当前知识范围：${next.label}` });
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (showCommands) {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setSelectedIndex((index) => (index + 1) % Math.max(filteredCommands.length, 1));
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setSelectedIndex((index) => (index - 1 + Math.max(filteredCommands.length, 1)) % Math.max(filteredCommands.length, 1));
        return;
      }
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        const selected = filteredCommands[selectedIndex] ?? filteredCommands[0];
        if (selected) pickCommand(selected);
        return;
      }
      if (event.key === "Escape") {
        setInput("");
        return;
      }
    }

    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      void submitInput();
    }
  }

  function pickCommand(command: Command) {
    if (command.placeholder) {
      setInput(`${command.command} `);
      textareaRef.current?.focus();
      return;
    }
    setInput("");
    addMessage({ role: "user", text: command.command });
    void runTask(async () => executeCommand(command.command));
  }

  function runSuggestion(command: string) {
    setInput("");
    addMessage({ role: "user", text: command });
    void runTask(async () => executeCommand(command));
  }

  return (
    <div className="extension-shell">
      <div className="side-panel">
        <header className="chrome-header">
          <div className="brand">
            <span>
              <Sparkles size={14} />
            </span>
            <strong>Survey Assistant</strong>
          </div>
          <div className="header-spacer" />
        </header>

        <main className="messages custom-scrollbar">
          {messages.length === 0 ? (
            <section className="greeting">
              <div className="hello">
                <span>Hello, John</span>
                <strong>今天需要我做些什么？</strong>
              </div>
              {!profile.profileId ? (
                <div className="setup-card">
                  <div>
                    <strong>先绑定当前 Profile</strong>
                    <span>绑定后，快照、笔记、翻译和对话都会写入独立工作空间。</span>
                  </div>
                  <button type="button" onClick={() => void runTask(async () => bindByCommand(profileDraft || "profile-001"))} disabled={busy || apiStatus === "offline" || apiStatus === "db-offline"}>
                    绑定并开始
                  </button>
                </div>
              ) : null}
              <div className="suggestions">
                {suggestionPills.map((pill) => (
                  <button key={pill.id} type="button" onClick={() => runSuggestion(pill.command)}>
                    {pill.text}
                  </button>
                ))}
              </div>
            </section>
          ) : (
            <section className="conversation">
              {messages.map((message) => (
                <MessageBubble key={message.id} message={message} onAction={handleMessageAction} />
              ))}
              {busy ? <ThinkingIndicator /> : null}
              <div ref={messagesEndRef} className="message-end" />
            </section>
          )}
        </main>

        <section className="input-dock">
          {showCommands && (
            <CommandMenu
              commands={filteredCommands}
              selectedIndex={selectedIndex}
              onHover={setSelectedIndex}
              onPick={pickCommand}
            />
          )}

          <div className={`compound-input ${busy ? "is-busy" : ""}`}>
            {busy ? <div className="input-progress" aria-hidden="true" /> : null}
            <div className="context-pill">
              <div title={contextStatusTitle(site, page, profile, systemStatus)}>
                <span className={`live-dot ${apiStatus}`} />
                <span>{apiStatusLabel(apiStatus)}</span>
                <em>{pageStatusLabel(page)}</em>
                {busy ? <em>处理中</em> : null}
              </div>
              <button type="button" onClick={() => void refreshPageContext(trimmedApiBaseUrl, true)} aria-label="刷新页面上下文">
                <RefreshCcw size={14} />
              </button>
            </div>

            <textarea
              ref={textareaRef}
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入 “/” 即可使用“技能”"
              rows={1}
            />

            <div className="input-controls">
              <div className="left-tools">
                <button type="button" aria-label="打开命令" onClick={() => {
                  setInput("/");
                  textareaRef.current?.focus();
                }}>
                  <Plus size={22} />
                </button>
                <button type="button" aria-label="设置" onClick={() => setSettingsOpen((open) => !open)}>
                  <SlidersHorizontal size={18} />
                </button>
              </div>

              <div className="right-tools">
                <button type="button" className="mode-button" onClick={cycleScopeMode}>
                  {scopeLabel}
                  <ChevronDown size={14} />
                </button>
                <button type="button" className="send-button" disabled={busy || !input.trim()} onClick={() => void submitInput()}>
                  <ArrowUp size={20} />
                </button>
              </div>
            </div>
          </div>
        </section>

        {settingsOpen && (
          <aside className="settings-panel">
            <div>
              <strong>连接设置</strong>
              <span>只在这里处理绑定，不占主界面</span>
            </div>
            <button className="settings-link-button" type="button" onClick={() => setShowAdvancedSettings((value) => !value)}>
              {showAdvancedSettings ? "隐藏 API 设置" : "显示 API 设置"}
            </button>
            {showAdvancedSettings ? (
              <label>
                API
                <input value={apiDraft} onChange={(event) => setApiDraft(event.target.value)} />
              </label>
            ) : null}
            <label>
              Username
              <input value={usernameDraft} autoComplete="username" onChange={(event) => setUsernameDraft(event.target.value)} />
            </label>
            <label>
              Password
              <input value={passwordDraft} type="password" autoComplete="current-password" placeholder="输入密码" onChange={(event) => setPasswordDraft(event.target.value)} />
            </label>
            <label>
              Profile
              <input value={profileDraft} onChange={(event) => setProfileDraft(event.target.value)} />
            </label>
            <label>
              Survey ID
              <input value={surveyId} placeholder="可选" inputMode="numeric" onChange={(event) => setSurveyId(event.target.value)} />
            </label>
            <button type="button" onClick={() => void saveSettings()} disabled={busy}>
              保存
            </button>
          </aside>
        )}
      </div>
    </div>
  );
}

function MessageBubble({
  message,
  onAction
}: {
  message: Message;
  onAction: (action: "save-note" | "copy" | "continue", message: Message) => Promise<void>;
}) {
  if (message.role === "user") {
    return (
      <div className="message-row user-row">
        <div className="user-bubble">{message.text}</div>
      </div>
    );
  }

  return (
    <article className={`message-row ai-row ${message.role}`}>
      <div className="ai-avatar">
        <Sparkles size={20} />
      </div>
      <div className="ai-body">
        {message.title ? <strong>{message.title}</strong> : null}
        <p>{message.text}</p>
        {message.meta ? <time>{message.meta}</time> : null}
        <div className="message-actions">
          <button type="button" title="复制" onClick={() => void onAction("copy", message)}>
            <Copy size={16} />
          </button>
          <button type="button" title="保存" onClick={() => void onAction("save-note", message)}>
            <Bookmark size={16} />
          </button>
          <button type="button" title="继续" onClick={() => void onAction("continue", message)}>
            <RefreshCcw size={16} />
          </button>
        </div>
      </div>
    </article>
  );
}

function ThinkingIndicator() {
  return (
    <article className="message-row ai-row thinking-row" aria-live="polite">
      <div className="ai-avatar">
        <Sparkles size={20} />
      </div>
      <div className="ai-body thinking-body">
        <strong>正在处理</strong>
        <div className="thinking-card">
          <span>AI 正在读取当前页面和历史记录</span>
          <div className="thinking-progress">
            <i />
          </div>
        </div>
      </div>
    </article>
  );
}

function CommandMenu({
  commands,
  selectedIndex,
  onHover,
  onPick
}: {
  commands: Command[];
  selectedIndex: number;
  onHover: (index: number) => void;
  onPick: (command: Command) => void;
}) {
  return (
    <div className="command-menu">
      <div className="command-list custom-scrollbar">
        {commands.map((item, index) => (
          <button
            key={item.id}
            type="button"
            onClick={() => onPick(item)}
            onMouseEnter={() => onHover(index)}
            className={index === selectedIndex ? "selected" : ""}
          >
            <span className="command-icon">{item.icon}</span>
            <span className="command-main">
              <strong>{item.command}</strong>
              {item.target ? <em>{item.target}</em> : null}
            </span>
            <span className="command-detail">{item.placeholder || item.detail}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function parseCommand(raw: string): { command: string; args: string } {
  const value = raw.trim();
  const matched = [...commands].sort((a, b) => b.command.length - a.command.length).find((item) => {
    return value === item.command || value.startsWith(`${item.command} `);
  });
  if (!matched) {
    const [command, ...rest] = value.split(/\s+/);
    return { command, args: rest.join(" ") };
  }
  return { command: matched.command, args: value.slice(matched.command.length) };
}

function filterCommands(input: string): Command[] {
  const query = input.trim().replace(/^\//, "").toLowerCase();
  if (!query) return commands;
  return commands.filter((item) => `${item.command} ${item.target} ${item.detail}`.toLowerCase().includes(query));
}

function isImportantSystemMessage(text: string): boolean {
  const value = text.toLowerCase();
  return value.includes("error") || value.includes("fail") || value.includes("failed") || value.includes("失败") || value.includes("异常");
}

function formatKnowledge(results: KnowledgeChunk[]): string {
  return results
    .slice(0, 6)
    .map((item, index) => `${index + 1}. [${sourceTypeLabel(item.source_type)}] ${cleanKnowledgeText(item.chunk_text)}`)
    .join("\n\n");
}

function formatNotes(notes: Note[]): string {
  return notes
    .slice(0, 8)
    .map((note, index) => `${index + 1}. ${note.note_text}`)
    .join("\n\n");
}

function formatConversations(history: AIConversation[]): string {
  return history
    .slice(0, 6)
    .map((item, index) => `${index + 1}. ${item.user_message}\n${item.ai_message}`)
    .join("\n\n");
}

function apiStatusLabel(status: ApiStatus): string {
  if (status === "online") return "在线";
  if (status === "ai-missing") return "在线";
  if (status === "db-offline") return "异常";
  if (status === "offline") return "离线";
  return "检查中";
}

function pageStatusLabel(page: ExtractedPage | null): string {
  if (!page) return "";
  if ((page.extractionConfidence ?? 0) >= 0.45 && page.questionText) return "题目已识别";
  if (page.questionText) return "页面可读";
  return "非问卷页";
}

function surveyQuestionText(page: ExtractedPage): string {
  const parts = [
    page.progressText ? `进度：${page.progressText}` : "",
    page.questionType ? `题型：${page.questionType}` : "",
    page.questionText ? `题目：${page.questionText}` : "",
    page.optionsText ? `选项：${page.optionsText}` : ""
  ].filter(Boolean);
  return parts.join("\n") || page.pageText;
}

function sourceTypeLabel(value: string): string {
  if (value === "note") return "笔记";
  if (value === "page_snapshot") return "页面";
  if (value === "ai_conversation") return "AI";
  return value;
}

function cleanKnowledgeText(value: string): string {
  return value.replace(/^User:\s*/i, "用户：").replace(/\sAI:\s*/i, "\nAI：").trim();
}

function contextStatusTitle(
  site: SiteDetection | null,
  page: ExtractedPage | null,
  profile: LocalProfile,
  systemStatus: SystemStatus | null
): string {
  const siteName = site?.site_name || site?.site_key || page?.domain || "未识别";
  const profileName = profile.profileName || profile.profileKey || "未绑定";
  const model = systemStatus?.ai_model || "-";
  const embedding = systemStatus?.ai_embedding_model || "-";
  return `网站：${siteName}\nProfile：${profileName}\nAI：${model}\nEmbedding：${embedding}`;
}

function formatDate(value: string): string {
  try {
    return new Date(value).toLocaleString();
  } catch {
    return value;
  }
}
