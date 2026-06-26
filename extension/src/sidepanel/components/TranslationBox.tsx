import { Languages, MousePointer2, WholeWord } from "lucide-react";
import type { Translation } from "../../shared/types";

type Props = {
  targetLang: string;
  translation: Translation | null;
  busy: boolean;
  onTargetLangChange: (value: string) => void;
  onTranslatePage: () => void;
  onTranslateSelection: () => void;
};

export function TranslationBox({
  targetLang,
  translation,
  busy,
  onTargetLangChange,
  onTranslatePage,
  onTranslateSelection
}: Props) {
  return (
    <section className="panel-section">
      <div className="section-title">
        <Languages size={18} aria-hidden="true" />
        <h2>翻译</h2>
      </div>
      <label>
        <span>目标语言</span>
        <input value={targetLang} onChange={(event) => onTargetLangChange(event.target.value)} />
      </label>
      <div className="toolbar two">
        <button type="button" onClick={onTranslatePage} disabled={busy}>
          <WholeWord size={16} aria-hidden="true" />
          翻译页面
        </button>
        <button type="button" onClick={onTranslateSelection} disabled={busy}>
          <MousePointer2 size={16} aria-hidden="true" />
          翻译选中
        </button>
      </div>
      {translation ? (
        <div className="translation-result">
          <div>
            <span>结果</span>
            <time>{new Date(translation.created_at).toLocaleString()}</time>
          </div>
          <textarea readOnly value={translation.translated_text} />
        </div>
      ) : (
        <div className="empty-state small">暂无翻译结果</div>
      )}
    </section>
  );
}
