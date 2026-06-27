CREATE TABLE IF NOT EXISTS answer_records (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    survey_id BIGINT REFERENCES surveys(id) ON DELETE SET NULL,
    page_snapshot_id BIGINT REFERENCES page_snapshots(id) ON DELETE SET NULL,
    question_text TEXT NOT NULL,
    options_text TEXT,
    suggested_answer TEXT,
    final_answer TEXT NOT NULL,
    reason TEXT,
    persona_matched TEXT,
    confidence NUMERIC(4,3) NOT NULL DEFAULT 1.000,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_answer_records_profile_site
ON answer_records (profile_id, site_key);

CREATE INDEX IF NOT EXISTS idx_answer_records_survey
ON answer_records (survey_id);

CREATE INDEX IF NOT EXISTS idx_answer_records_created_at
ON answer_records (created_at DESC);
