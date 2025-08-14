-- Schema definition for the patient waitroom chatbot (patient-only MVP)

-- sessions: one per patient visit
CREATE TABLE IF NOT EXISTS sessions (
    id                  UUID PRIMARY KEY,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at           TIMESTAMPTZ,
    message_cap         INT NOT NULL DEFAULT 50,
    patient_name        TEXT,
    patient_address     TEXT,
    patient_phone       TEXT,
    patient_national_id TEXT,
    client_ip           INET,
    user_agent          TEXT
);

-- messages: transcript lines
CREATE TABLE IF NOT EXISTS messages (
    id          BIGSERIAL PRIMARY KEY,
    session_id  UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('patient','bot')),
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_session_id_created_at
    ON messages (session_id, created_at);

-- summaries: one row per session
CREATE TABLE IF NOT EXISTS summaries (
    id          BIGSERIAL PRIMARY KEY,
    session_id  UUID NOT NULL UNIQUE REFERENCES sessions(id) ON DELETE CASCADE,
    key_points  JSONB NOT NULL DEFAULT '[]'::jsonb,
    structured  JSONB NOT NULL DEFAULT '{}'::jsonb,
    free_text   TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_summaries_updated_at
    ON summaries (updated_at DESC);