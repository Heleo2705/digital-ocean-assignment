package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type Store struct {
	db *sql.DB
}

type StoreUser struct {
	ID           string
	Email        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type StoreJob struct {
	ID             string
	UserID         string
	Type           string
	Name           string
	WebhookURL     string
	Payload        []byte
	Version        int
	MaxRetries     int
	TimeoutSeconds int
	State          string
	Attempts       int
	ScheduledAt    time.Time
	LastError      sql.NullString
	Result         []byte
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type StoreIdempotencyKey struct {
	ID          string
	UserID      string
	RequestHash string
	JobID       sql.NullString
	Method      string
	Path        string
	StatusCode  sql.NullInt32
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type StoreOutboxEvent struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	Published     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StoreCreateJobParams struct {
	UserID         string
	Type           string
	Name           string
	WebhookURL     string
	Payload        []byte
	MaxRetries     int
	TimeoutSeconds int
	Version        int
}

type StoreCreateOutboxEventParams struct {
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	Published     bool
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, nil)
}

func (s *Store) CreateUser(ctx context.Context, email, username, passwordHash string) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
INSERT INTO users (email, username, password_hash)
VALUES ($1, $2, $3)
RETURNING id
`, email, username, passwordHash).Scan(&id)
	return id, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*StoreUser, error) {
	var user StoreUser
	err := s.db.QueryRowContext(ctx, `
SELECT id, email, username, password_hash, created_at, updated_at
FROM users
WHERE email = $1
`, email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetJobByID(ctx context.Context, jobID string) (*StoreJob, error) {
	return s.GetJobByIDTx(ctx, nil, jobID)
}

func (s *Store) GetJobByIDTx(ctx context.Context, tx *sql.Tx, jobID string) (*StoreJob, error) {
	query := `
SELECT id, user_id, type, name, webhook_url, payload, version, max_retries, timeout_seconds, state, attempts, scheduled_at, last_error, result, created_at, updated_at
FROM jobs
WHERE id = $1
`
	row := s.queryRow(ctx, tx, query, jobID)
	var job StoreJob
	err := row.Scan(
		&job.ID,
		&job.UserID,
		&job.Type,
		&job.Name,
		&job.WebhookURL,
		&job.Payload,
		&job.Version,
		&job.MaxRetries,
		&job.TimeoutSeconds,
		&job.State,
		&job.Attempts,
		&job.ScheduledAt,
		&job.LastError,
		&job.Result,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Store) GetJobByIDForUpdateTx(ctx context.Context, tx *sql.Tx, jobID string) (*StoreJob, error) {
	query := `
SELECT id, user_id, type, name, webhook_url, payload, version, max_retries, timeout_seconds, state, attempts, scheduled_at, last_error, result, created_at, updated_at
FROM jobs
WHERE id = $1
FOR UPDATE
`
	row := tx.QueryRowContext(ctx, query, jobID)
	var job StoreJob
	err := row.Scan(
		&job.ID,
		&job.UserID,
		&job.Type,
		&job.Name,
		&job.WebhookURL,
		&job.Payload,
		&job.Version,
		&job.MaxRetries,
		&job.TimeoutSeconds,
		&job.State,
		&job.Attempts,
		&job.ScheduledAt,
		&job.LastError,
		&job.Result,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Store) ListJobsByUser(ctx context.Context, userID string, limit, offset int) ([]StoreJob, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, type, name, webhook_url, payload, version, max_retries, timeout_seconds, state, attempts, scheduled_at, last_error, result, created_at, updated_at
FROM jobs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []StoreJob
	for rows.Next() {
		var job StoreJob
		if err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.Type,
			&job.Name,
			&job.WebhookURL,
			&job.Payload,
			&job.Version,
			&job.MaxRetries,
			&job.TimeoutSeconds,
			&job.State,
			&job.Attempts,
			&job.ScheduledAt,
			&job.LastError,
			&job.Result,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) UpdateJobDetails(ctx context.Context, jobID, userID, name, webhookURL string, maxRetries, timeoutSeconds, version int) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET name = $1, webhook_url = $2, max_retries = $3, timeout_seconds = $4, version = $5, updated_at = now()
WHERE id = $6 AND user_id = $7
`, name, webhookURL, maxRetries, timeoutSeconds, version, jobID, userID)
	return err
}

func (s *Store) DeleteJob(ctx context.Context, jobID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
DELETE FROM jobs
WHERE id = $1 AND user_id = $2
`, jobID, userID)
	return err
}

func (s *Store) IncrementJobAttemptsTx(ctx context.Context, tx *sql.Tx, jobID string) (int, error) {
	var attempts int
	err := tx.QueryRowContext(ctx, `
UPDATE jobs
SET attempts = attempts + 1, updated_at = now()
WHERE id = $1
RETURNING attempts
`, jobID).Scan(&attempts)
	return attempts, err
}

func (s *Store) UpdateJobStateTx(ctx context.Context, tx *sql.Tx, jobID, state string, lastError sql.NullString, result []byte, attempts int) error {
	_, err := tx.ExecContext(ctx, `
UPDATE jobs
SET state = $1, last_error = $2, result = $3, attempts = $4, updated_at = now()
WHERE id = $5
`, state, lastError, result, attempts, jobID)
	return err
}

func (s *Store) queryRow(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) *sql.Row {
	if tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *Store) CreateJobWithIdempotency(
	ctx context.Context,
	userID, requestHash, method, path string,
	params StoreCreateJobParams,
) (*StoreJob, bool, error) {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	var existingJobID string
	err = tx.QueryRowContext(ctx, `
SELECT job_id FROM idempotency_keys
WHERE user_id = $1 AND request_hash = $2
`, userID, requestHash).Scan(&existingJobID)
	if err == nil {
		job, err := s.GetJobByIDTx(ctx, tx, existingJobID)
		if err != nil {
			return nil, false, err
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, false, commitErr
		}
		return job, true, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}

	var jobID string
	maxRetries := params.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}
	timeoutSeconds := params.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 15
	}
	version := params.Version
	if version <= 0 {
		version = 1
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO jobs (user_id, type, name, webhook_url, payload, version, max_retries, timeout_seconds, state, attempts, scheduled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id
`, params.UserID, params.Type, params.Name, params.WebhookURL, params.Payload, version, maxRetries, timeoutSeconds, "pending", 0, time.Now()).Scan(&jobID)
	if err != nil {
		return nil, false, err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO idempotency_keys (user_id, request_hash, job_id, method, path, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
`, userID, requestHash, jobID, method, path, time.Now().Add(24*time.Hour))
	if err != nil {
		return nil, false, err
	}

	outboxPayload, err := json.Marshal(map[string]interface{}{
		"job_id":      jobID,
		"type":        params.Type,
		"name":        params.Name,
		"webhook_url": params.WebhookURL,
		"payload":     json.RawMessage(params.Payload),
	})
	if err != nil {
		return nil, false, err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO outbox (aggregate_type, aggregate_id, event_type, payload, published)
VALUES ($1, $2, $3, $4, $5)
`, "job", jobID, "job_created", outboxPayload, false)
	if err != nil {
		return nil, false, err
	}

	job, err := s.GetJobByIDTx(ctx, tx, jobID)
	if err != nil {
		return nil, false, err
	}

	if err := tx.Commit(); err != nil {
		return nil, false, err
	}

	return job, false, nil
}

func (s *Store) GetUnpublishedOutboxEventsTx(ctx context.Context, tx *sql.Tx, limit int) ([]StoreOutboxEvent, error) {
	query := `
SELECT id, aggregate_type, aggregate_id, event_type, payload, published, created_at, updated_at
FROM outbox
WHERE published = false
ORDER BY created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED
`
	rows, err := tx.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []StoreOutboxEvent
	for rows.Next() {
		var event StoreOutboxEvent
		if err := rows.Scan(&event.ID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.Payload, &event.Published, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) MarkOutboxEventPublishedTx(ctx context.Context, tx *sql.Tx, outboxID string) error {
	_, err := tx.ExecContext(ctx, `
UPDATE outbox
SET published = true, published_at = now(), updated_at = now()
WHERE id = $1
`, outboxID)
	return err
}

func (s *Store) MarkOutboxEventUnpublished(ctx context.Context, outboxID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE outbox
SET published = false, updated_at = now()
WHERE id = $1
`, outboxID)
	return err
}

func (s *Store) UpdateJobState(ctx context.Context, jobID, state string, lastError sql.NullString, result []byte, attempts int) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET state = $1, last_error = $2, result = $3, attempts = $4, updated_at = now()
WHERE id = $5
`, state, lastError, result, attempts, jobID)
	return err
}

func (s *Store) IncrementJobAttempts(ctx context.Context, jobID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE jobs
SET attempts = attempts + 1, updated_at = now()
WHERE id = $1
`, jobID)
	return err
}

func (s *Store) CreateOutboxEvent(ctx context.Context, params StoreCreateOutboxEventParams) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `
INSERT INTO outbox (aggregate_type, aggregate_id, event_type, payload, published)
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`, params.AggregateType, params.AggregateID, params.EventType, params.Payload, params.Published).Scan(&id)
	return id, err
}

func (s *Store) CreateOutboxEventTx(ctx context.Context, tx *sql.Tx, params StoreCreateOutboxEventParams) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
INSERT INTO outbox (aggregate_type, aggregate_id, event_type, payload, published)
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`, params.AggregateType, params.AggregateID, params.EventType, params.Payload, params.Published).Scan(&id)
	return id, err
}
