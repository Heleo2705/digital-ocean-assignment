package service

import (
	"context"
	"time"

	"github.com/Heleo2705/assignment/db"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func StartOutboxPollers(ctx context.Context, store *db.Store, client *asynq.Client, logger *zap.Logger, count int, limit int, interval time.Duration) {
	for i := 0; i < count; i++ {
		pollerLogger := logger.With(zap.Int("poller", i))
		go func() {
			pollerLogger.Info("starting outbox poller")
			for {
				if err := pollOnce(ctx, store, client, pollerLogger, limit); err != nil {
					pollerLogger.Error("outbox poll failed", zap.Error(err))
				}

				select {
				case <-ctx.Done():
					pollerLogger.Info("stopping outbox poller")
					return
				case <-time.After(interval):
				}
			}
		}()
	}
}

func pollOnce(ctx context.Context, store *db.Store, client *asynq.Client, logger *zap.Logger, limit int) error {
	tx, err := store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	events, err := store.GetUnpublishedOutboxEventsTx(ctx, tx, limit)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return tx.Commit()
	}

	for _, event := range events {
		if err := store.MarkOutboxEventPublishedTx(ctx, tx, event.ID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	for _, event := range events {
		task, err := NewJobProcessTask(event.AggregateID, event.ID)
		if err != nil {
			logger.Error("failed to build task", zap.Error(err), zap.String("outbox_id", event.ID), zap.String("aggregate_id", event.AggregateID))
			if cleanupErr := store.MarkOutboxEventUnpublished(ctx, event.ID); cleanupErr != nil {
				logger.Error("failed to unpublish outbox event", zap.Error(cleanupErr), zap.String("outbox_id", event.ID))
			}
			continue
		}

		_, err = client.Enqueue(task, asynq.MaxRetry(0))
		if err != nil {
			logger.Error("failed to enqueue task", zap.Error(err), zap.String("outbox_id", event.ID), zap.String("aggregate_id", event.AggregateID))
			if cleanupErr := store.MarkOutboxEventUnpublished(ctx, event.ID); cleanupErr != nil {
				logger.Error("failed to unpublish outbox event", zap.Error(cleanupErr), zap.String("outbox_id", event.ID))
			}
			continue
		}

		logger.Info("enqueued job task", zap.String("job_id", event.AggregateID), zap.String("outbox_id", event.ID))
	}

	return nil
}
