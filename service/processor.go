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

	job, err := store.GetJobByID(ctx, payload.JobID)
	if err != nil {
		logger.Error("failed to load job", zap.Error(err), zap.String("job_id", payload.JobID))
		return nil
	}

	logger = logger.With(zap.String("job_id", job.ID), zap.String("outbox_id", payload.OutboxID))
	logger.Info("starting job execution")

	if err := store.UpdateJobState(ctx, job.ID, "running", sql.NullString{}, nil, job.Attempts); err != nil {
		logger.Error("failed to update job state to running", zap.Error(err))
		return nil
	}

	job.Attempts++
	result, err := executeJobWebhook(ctx, job)
	if err != nil {
		logger.Error("job execution failed", zap.Error(err), zap.Int("attempt", job.Attempts), zap.Int("max_retries", job.MaxRetries))
		if job.Attempts >= job.MaxRetries {
			logger.Info("marking job failed after max retries")
			return store.UpdateJobState(ctx, job.ID, "failed", sql.NullString{String: err.Error(), Valid: true}, nil, job.Attempts)
		}

		if err := store.UpdateJobState(ctx, job.ID, "queued", sql.NullString{String: err.Error(), Valid: true}, nil, job.Attempts); err != nil {
			logger.Error("failed to update job retry state", zap.Error(err))
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

		if _, err := store.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			AggregateType: "job",
			AggregateID:   job.ID,
			EventType:     "job_retry",
			Payload:       payloadBytes,
			Published:     false,
		}); err != nil {
			logger.Error("failed to create retry outbox event", zap.Error(err))
		}
		return nil
	}

	logger.Info("job execution completed")
	return store.UpdateJobState(ctx, job.ID, "completed", sql.NullString{}, result, job.Attempts)
}

func executeJobWebhook(ctx context.Context, job *db.Job) ([]byte, error) {
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
