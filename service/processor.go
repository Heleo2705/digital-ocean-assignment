package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Heleo2705/assignment/db"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func ProcessJobTask(ctx context.Context, store *db.Store, task *asynq.Task, logger *zap.Logger) error {
	payload, err := ParseJobProcessPayload(task)
	if err != nil {
		logger.Error("invalid task payload", zap.Error(err))
		return nil
	}

	tx, err := store.BeginTx(ctx)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return nil
	}
	defer tx.Rollback()

	job, err := store.GetJobByIDForUpdateTx(ctx, tx, payload.JobID)
	if err != nil {
		logger.Error("failed to load job", zap.Error(err), zap.String("job_id", payload.JobID))
		return nil
	}

	attempts, err := store.IncrementJobAttemptsTx(ctx, tx, job.ID)
	if err != nil {
		logger.Error("failed to increment job attempts", zap.Error(err), zap.String("job_id", job.ID))
		return nil
	}

	if err := store.UpdateJobStateTx(ctx, tx, job.ID, "running", sql.NullString{}, nil, attempts); err != nil {
		logger.Error("failed to update job state to running", zap.Error(err), zap.String("job_id", job.ID))
		return nil
	}

	if err := tx.Commit(); err != nil {
		logger.Error("failed to commit running state", zap.Error(err), zap.String("job_id", job.ID))
		return nil
	}

	logger = logger.With(zap.String("job_id", job.ID), zap.String("outbox_id", payload.OutboxID), zap.Int("attempt", attempts))
	logger.Info("starting job execution")

	result, err := executeJobWebhook(ctx, job)
	if err != nil {
		logger.Error("job execution failed", zap.Error(err), zap.Int("attempt", attempts), zap.Int("max_retries", job.MaxRetries))

		retryTx, txErr := store.BeginTx(ctx)
		if txErr != nil {
			logger.Error("failed to begin retry transaction", zap.Error(txErr), zap.String("job_id", job.ID))
			return nil
		}
		defer retryTx.Rollback()

		if attempts >= job.MaxRetries {
			logger.Info("marking job failed after max retries")
			if err := store.UpdateJobStateTx(ctx, retryTx, job.ID, "failed", sql.NullString{String: err.Error(), Valid: true}, nil, attempts); err != nil {
				logger.Error("failed to mark job failed", zap.Error(err), zap.String("job_id", job.ID))
				return nil
			}
		} else {
			if err := store.UpdateJobStateTx(ctx, retryTx, job.ID, "queued", sql.NullString{String: err.Error(), Valid: true}, nil, attempts); err != nil {
				logger.Error("failed to update job retry state", zap.Error(err), zap.String("job_id", job.ID))
				return nil
			}

			payloadBytes, marshalErr := json.Marshal(map[string]interface{}{
				"job_id":      job.ID,
				"type":        job.Type,
				"name":        job.Name,
				"webhook_url": job.WebhookURL,
				"payload":     json.RawMessage(job.Payload),
			})
			if marshalErr != nil {
				logger.Error("failed to marshal retry payload", zap.Error(marshalErr))
				return nil
			}

			if _, err := store.CreateOutboxEventTx(ctx, retryTx, db.StoreCreateOutboxEventParams{
				AggregateType: "job",
				AggregateID:   job.ID,
				EventType:     "job_retry",
				Payload:       payloadBytes,
				Published:     false,
			}); err != nil {
				logger.Error("failed to create retry outbox event", zap.Error(err), zap.String("job_id", job.ID))
				return nil
			}
		}

		if err := retryTx.Commit(); err != nil {
			logger.Error("failed to commit retry transaction", zap.Error(err), zap.String("job_id", job.ID))
			return nil
		}

		return nil
	}

	logger.Info("job execution completed")
	return store.UpdateJobState(ctx, job.ID, "completed", sql.NullString{}, result, attempts)
}

func executeJobWebhook(ctx context.Context, job *db.StoreJob) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(job.WebhookURL, "application/json", bytesReader(job.Payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &WebhookError{StatusCode: resp.StatusCode, Body: body}
	}

	return body, nil
}

type WebhookError struct {
	StatusCode int
	Body       []byte
}

func (w *WebhookError) Error() string {
	return string(w.Body)
}

func bytesReader(payload []byte) io.Reader {
	return bytes.NewReader(payload)
}
