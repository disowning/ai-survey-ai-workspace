import { History, NotebookPen, Save } from "lucide-react";
import type { Note } from "../../shared/types";

type Props = {
  noteDraft: string;
  notes: Note[];
  busy: boolean;
  onNoteDraftChange: (value: string) => void;
  onSaveNote: () => void;
  onLoadNotes: () => void;
};

export function NotesBox({ noteDraft, notes, busy, onNoteDraftChange, onSaveNote, onLoadNotes }: Props) {
  return (
    <section className="panel-section">
      <div className="section-title">
        <NotebookPen size={18} aria-hidden="true" />
        <h2>当前笔记</h2>
      </div>
      <textarea
        className="note-input"
        value={noteDraft}
        placeholder="记录当前页面理解、上下文、需要复查的信息"
        onChange={(event) => onNoteDraftChange(event.target.value)}
      />
      <div className="toolbar two">
        <button type="button" onClick={onSaveNote} disabled={busy || !noteDraft.trim()}>
          <Save size={16} aria-hidden="true" />
          保存笔记
        </button>
        <button type="button" onClick={onLoadNotes} disabled={busy}>
          <History size={16} aria-hidden="true" />
          查看历史
        </button>
      </div>
      {notes.length ? (
        <div className="history-list">
          {notes.map((note) => (
            <article key={note.id}>
              <time>{new Date(note.created_at).toLocaleString()}</time>
              <p>{note.note_text}</p>
            </article>
          ))}
        </div>
      ) : (
        <div className="empty-state small">暂无历史笔记</div>
      )}
    </section>
  );
}
