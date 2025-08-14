-- Schema definition for the patient waitroom chatbot (idempotent-ish)

-- sessions: each record represents a patient visit initiated via a QR code.

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    message_cap INT NOT NULL DEFAULT 50,
    patient_phone TEXT,
    patient_national_id TEXT,
    patient_name TEXT,
    patient_address TEXT,
    client_ip INET,
    user_agent TEXT
);

-- messages: transcript lines for a session.
-- role is constrained to 'patient' or 'bot' without creating a custom enum.
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('patient','bot')),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_session_id_created_at
    ON messages (session_id, created_at);

-- summaries: one row per session (upserted/updated by worker).
CREATE TABLE IF NOT EXISTS summaries (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL UNIQUE REFERENCES sessions(id) ON DELETE CASCADE,
    key_points JSONB NOT NULL DEFAULT '[]'::jsonb,
    structured JSONB NOT NULL DEFAULT '{}'::jsonb,
    free_text TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Helpful index if you query summary freshness
CREATE INDEX IF NOT EXISTS idx_summaries_updated_at
    ON summaries (updated_at DESC);

-- one-time quick fix
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  totp_secret TEXT,
  national_id TEXT UNIQUE,
  name TEXT,
  phone TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users
    ALTER COLUMN email DROP NOT NULL,
    ALTER COLUMN password_hash DROP NOT NULL,
    ALTER COLUMN totp_secret DROP NOT NULL,
    ALTER COLUMN created_at DROP NOT NULL;