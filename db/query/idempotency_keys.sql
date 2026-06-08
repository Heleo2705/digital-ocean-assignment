-- name: CreateIdempotencyKey
INSERT INTO idempotency_keys (
  user_id,
  request_hash,
  job_id,
  method,
  path,
  response_payload,
  status_code,
  expires_at
) VALUES (
  :user_id,
  :request_hash,
  :job_id,
  :method,
  :path,
  :response_payload,
  :status_code,
  :expires_at
)
RETURNING *;

-- name: GetIdempotencyKeyByHash
SELECT * FROM idempotency_keys WHERE user_id = $1 AND request_hash = $2;

-- name: UpdateIdempotencyKeyResponse
UPDATE idempotency_keys SET
  response_payload = COALESCE(:response_payload, response_payload),
  status_code = COALESCE(:status_code, status_code),
  updated_at = now()
WHERE id = :id
RETURNING *;

-- name: DeleteIdempotencyKey
DELETE FROM idempotency_keys WHERE id = $1;
