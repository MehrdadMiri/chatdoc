-- Migration: create initial tables for the patient waitroom chatbot
-- This file is identical to internal/db/schema.sql to make it easy to
-- integrate with migration tools such as golang-migrate.

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    message_cap INT NOT NULL DEFAULT 50,
    patient_phone TEXT,
    patient_national_id TEXT,
    client_ip INET,
    user_agent TEXT
);

CREATE TABLE IF NOT EXISTS users (
    national_id TEXT PRIMARY KEY,
    phone TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE IF NOT EXISTS message_role AS ENUM ('patient', 'bot');

CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID REFERENCES sessions(id) ON DELETE CASCADE,
    national_id TEXT REFERENCES users(national_id),
    role message_role NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS summaries (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE UNIQUE,
    key_points JSONB NOT NULL,
    structured JSONB NOT NULL,
    free_text TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);