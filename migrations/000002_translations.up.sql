CREATE TABLE IF NOT EXISTS translations (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    source_text TEXT NOT NULL,
    translated_text TEXT NOT NULL,
    source_lang VARCHAR(50),
    target_lang VARCHAR(50) NOT NULL DEFAULT 'zh-CN',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_translations_profile_site ON translations (profile_id, site_key);
CREATE INDEX IF NOT EXISTS idx_translations_created_at ON translations (created_at DESC);
