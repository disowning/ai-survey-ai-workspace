import { Database, FileSearch, Globe2, ScanText } from "lucide-react";
import type { ExtractedPage, SiteDetection } from "../../shared/types";

type Props = {
  page: ExtractedPage | null;
  site: SiteDetection | null;
  busy: boolean;
  onExtract: () => void;
  onDetectSite: () => void;
  onSaveSnapshot: () => void;
};

export function PageExtractBox({ page, site, busy, onExtract, onDetectSite, onSaveSnapshot }: Props) {
  return (
    <section className="panel-section">
      <div className="section-title">
        <ScanText size={18} aria-hidden="true" />
        <h2>当前页面</h2>
      </div>
      <div className="toolbar">
        <button type="button" onClick={onExtract} disabled={busy}>
          <FileSearch size={16} aria-hidden="true" />
          提取页面
        </button>
        <button type="button" onClick={onDetectSite} disabled={busy || !page}>
          <Globe2 size={16} aria-hidden="true" />
          识别网站
        </button>
        <button type="button" onClick={onSaveSnapshot} disabled={busy || !page || !site}>
          <Database size={16} aria-hidden="true" />
          保存快照
        </button>
      </div>
      {page ? (
        <div className="page-summary">
          <div>
            <span>标题</span>
            <strong>{page.title}</strong>
          </div>
          <div>
            <span>域名</span>
            <strong>{page.domain}</strong>
          </div>
          <div>
            <span>网站</span>
            <strong>{site?.site_key || "未识别"}</strong>
          </div>
          <textarea readOnly value={page.questionText || page.pageText.slice(0, 1200)} />
        </div>
      ) : (
        <div className="empty-state">等待提取当前标签页内容</div>
      )}
    </section>
  );
}
