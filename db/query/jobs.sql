-- name: CreateJob :one
INSERT INTO jobs (
  user_id,
  type,
  name,
  webhook_url,
  payload,
  state,
  max_retries,
  attempts,
  scheduled_at,
  last_error,
  result
) VALUES (
  @user_id,
  @type,
  @name,
  @webhook_url,
  @payload,
  @state,
  @max_retries,
  @attempts,
  @scheduled_at,
  @last_error,
  @result
)
RETURNING *;

-- name: GetJobByID :one
SELECT * FROM jobs WHERE id = $1;

-- name: ListJobsByUserID :many
SELECT * FROM jobs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateJob :one
UPDATE jobs SET
  type = COALESCE(@type, type),
  name = COALESCE(@name, name),
  webhook_url = COALESCE(@webhook_url, webhook_url),
  payload = COALESCE(@payload, payload),
  state = COALESCE(@state, state),
  max_retries = COALESCE(@max_retries, max_retries),
  attempts = COALESCE(@attempts, attempts),
  scheduled_at = COALESCE(@scheduled_at, scheduled_at),
  last_error = COALESCE(@last_error, last_error),
  result = COALESCE(@result, result),
  updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteJob :exec
DELETE FROM jobs WHERE id = $1;
