CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    survey_id BIGINT REFERENCES surveys(id) ON DELETE SET NULL,
    scope VARCHAR(50) NOT NULL DEFAULT 'site',
    source_type VARCHAR(50) NOT NULL DEFAULT 'manual',
    source_id BIGINT,
    chunk_text TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_profile_site ON knowledge_chunks (profile_id, site_key);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_survey ON knowledge_chunks (survey_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_scope ON knowledge_chunks (scope);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_created_at ON knowledge_chunks (created_at DESC);
