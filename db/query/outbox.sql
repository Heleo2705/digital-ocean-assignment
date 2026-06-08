-- name: CreateOutboxEvent
INSERT INTO outbox (
  aggregate_type,
  aggregate_id,
  event_type,
  payload,
  published
) VALUES (
  :aggregate_type,
  :aggregate_id,
  :event_type,
  :payload,
  :published
)
RETURNING *;

-- name: GetUnpublishedOutboxEvents
SELECT * FROM outbox WHERE published = false ORDER BY created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED;

-- name: MarkOutboxEventPublished
UPDATE outbox SET published = true, published_at = now(), updated_at = now() WHERE id = :id RETURNING *;

-- name: DeleteOutboxEvent
DELETE FROM outbox WHERE id = $1;
