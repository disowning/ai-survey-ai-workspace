import { Search } from "lucide-react";
import type { KnowledgeChunk } from "../../shared/types";

type Props = {
  query: string;
  results: KnowledgeChunk[];
  busy: boolean;
  onQueryChange: (value: string) => void;
  onSearch: () => void;
};

export function KnowledgeSearch({ query, results, busy, onQueryChange, onSearch }: Props) {
  return (
    <section className="panel-section">
      <div className="section-title">
        <Search size={18} aria-hidden="true" />
        <h2>搜索知识库</h2>
      </div>
      <div className="inline-search">
        <input value={query} placeholder="输入关键词" onChange={(event) => onQueryChange(event.target.value)} />
        <button type="button" onClick={onSearch} disabled={busy || !query.trim()} aria-label="搜索知识库">
          <Search size={16} aria-hidden="true" />
        </button>
      </div>
      {results.length ? (
        <div className="knowledge-list">
          {results.map((item) => (
            <article key={item.id}>
              <div>
                <strong>{item.source_type}</strong>
                <time>{new Date(item.created_at).toLocaleString()}</time>
              </div>
              <p>{item.chunk_text}</p>
            </article>
          ))}
        </div>
      ) : (
        <div className="empty-state small">暂无搜索结果</div>
      )}
    </section>
  );
}
