-- name: CreateOutboxEvent :one
INSERT INTO outbox (
  aggregate_type,
  aggregate_id,
  event_type,
  payload,
  published
) VALUES (
  @aggregate_type,
  @aggregate_id,
  @event_type,
  @payload,
  @published
)
RETURNING *;

-- name: GetUnpublishedOutboxEvents :many
SELECT * FROM outbox WHERE published = false ORDER BY created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished :one
UPDATE outbox SET published = true, published_at = now(), updated_at = now() WHERE id = @id RETURNING *;

-- name: DeleteOutboxEvent :exec
DELETE FROM outbox WHERE id = $1;
