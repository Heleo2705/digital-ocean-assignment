CREATE TABLE IF NOT EXISTS idempotency_keys (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  request_hash text NOT NULL,
  job_id uuid REFERENCES jobs(id) ON DELETE CASCADE,
  method text NOT NULL,
  path text NOT NULL,
  response_payload jsonb,
  status_code int,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL DEFAULT now() + INTERVAL '24 hours',
  UNIQUE (user_id, request_hash)
);
