-- Enable UUID generation and create users table.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username text UNIQUE,
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  keycloak_id text UNIQUE,
  keycloak_access_token text,
  keycloak_refresh_token text,
  keycloak_access_expires_at timestamptz,
  keycloak_refresh_expires_at timestamptz,
  is_active boolean NOT NULL DEFAULT true,
  roles text[] NOT NULL DEFAULT ARRAY['user'],
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
