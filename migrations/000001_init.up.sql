CREATE TABLE IF NOT EXISTS profiles (
    id BIGSERIAL PRIMARY KEY,
    profile_key VARCHAR(100) UNIQUE NOT NULL,
    profile_name VARCHAR(255),
    remark TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS survey_sites (
    id BIGSERIAL PRIMARY KEY,
    site_key VARCHAR(100) UNIQUE NOT NULL,
    site_name VARCHAR(255),
    domain VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS profile_sites (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    site_account_email VARCHAR(255),
    site_remark TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (profile_id, site_key)
);

CREATE TABLE IF NOT EXISTS surveys (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    survey_title TEXT,
    survey_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS page_snapshots (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    survey_id BIGINT REFERENCES surveys(id) ON DELETE SET NULL,
    url TEXT,
    page_title TEXT,
    question_text TEXT,
    options_text TEXT,
    page_text TEXT,
    language VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notes (
    id BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    site_key VARCHAR(100) NOT NULL REFERENCES survey_sites(site_key) ON DELETE RESTRICT,
    survey_id BIGINT REFERENCES surveys(id) ON DELETE SET NULL,
    page_snapshot_id BIGINT REFERENCES page_snapshots(id) ON DELETE SET NULL,
    note_text TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profile_sites_profile_site ON profile_sites (profile_id, site_key);
CREATE INDEX IF NOT EXISTS idx_surveys_profile_site ON surveys (profile_id, site_key);
CREATE INDEX IF NOT EXISTS idx_page_snapshots_profile_site ON page_snapshots (profile_id, site_key);
CREATE INDEX IF NOT EXISTS idx_page_snapshots_survey ON page_snapshots (survey_id);
CREATE INDEX IF NOT EXISTS idx_notes_profile_site ON notes (profile_id, site_key);
CREATE INDEX IF NOT EXISTS idx_notes_survey ON notes (survey_id);
CREATE INDEX IF NOT EXISTS idx_survey_sites_domain ON survey_sites (domain);
