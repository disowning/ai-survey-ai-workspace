import { Bot, History, SendHorizontal } from "lucide-react";
import type { AIConversation } from "../../shared/types";

type Props = {
  chatDraft: string;
  scope: string;
  conversations: AIConversation[];
  busy: boolean;
  onChatDraftChange: (value: string) => void;
  onScopeChange: (value: string) => void;
  onSend: () => void;
  onLoadHistory: () => void;
};

export function ChatBox({
  chatDraft,
  scope,
  conversations,
  busy,
  onChatDraftChange,
  onScopeChange,
  onSend,
  onLoadHistory
}: Props) {
  return (
    <section className="panel-section">
      <div className="section-title">
        <Bot size={18} aria-hidden="true" />
        <h2>AI 对话</h2>
      </div>
      <label>
        <span>知识库范围</span>
        <select value={scope} onChange={(event) => onScopeChange(event.target.value)}>
          <option value="survey">当前问卷</option>
          <option value="site">当前网站</option>
          <option value="profile">当前 Profile</option>
          <option value="team">团队公共库</option>
        </select>
      </label>
      <textarea
        className="chat-input"
        value={chatDraft}
        placeholder="这一题是什么意思？"
        onChange={(event) => onChatDraftChange(event.target.value)}
      />
      <div className="toolbar two">
        <button type="button" onClick={onSend} disabled={busy || !chatDraft.trim()}>
          <SendHorizontal size={16} aria-hidden="true" />
          向 AI 提问
        </button>
        <button type="button" onClick={onLoadHistory} disabled={busy}>
          <History size={16} aria-hidden="true" />
          对话历史
        </button>
      </div>
      {conversations.length ? (
        <div className="chat-list">
          {conversations.map((item) => (
            <article key={item.id}>
              <time>{new Date(item.created_at).toLocaleString()}</time>
              <p className="user-message">{item.user_message}</p>
              <p className="ai-message">{item.ai_message}</p>
            </article>
          ))}
        </div>
      ) : (
        <div className="empty-state small">暂无 AI 对话</div>
      )}
    </section>
  );
}
