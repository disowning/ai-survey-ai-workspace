CREATE TABLE IF NOT EXISTS site_personas (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    category VARCHAR(50) NOT NULL DEFAULT 'preference',
    persona_key VARCHAR(120) NOT NULL,
    persona_value TEXT NOT NULL,
    confidence NUMERIC(4,3) NOT NULL DEFAULT 1.000,
    source_type VARCHAR(50) NOT NULL DEFAULT 'manual',
    source_id BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (profile_id, site_key, persona_key)
);

CREATE INDEX IF NOT EXISTS idx_site_personas_profile_site
ON site_personas (profile_id, site_key);

CREATE INDEX IF NOT EXISTS idx_site_personas_category
ON site_personas (category);

CREATE INDEX IF NOT EXISTS idx_site_personas_updated_at
ON site_personas (updated_at DESC);
