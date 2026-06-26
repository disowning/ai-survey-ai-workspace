CREATE TABLE IF NOT EXISTS ai_conversations (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    survey_id BIGINT REFERENCES surveys(id) ON DELETE SET NULL,
    user_message TEXT NOT NULL,
    ai_message TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_conversations_profile_site ON ai_conversations (profile_id, site_key);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_survey ON ai_conversations (survey_id);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_created_at ON ai_conversations (created_at DESC);
