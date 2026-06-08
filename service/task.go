package service

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TypeJobProcess = "job:process"

type JobProcessPayload struct {
	JobID    string `json:"job_id"`
	OutboxID string `json:"outbox_id"`
}

func NewJobProcessTask(jobID, outboxID string) (*asynq.Task, error) {
	payload, err := json.Marshal(JobProcessPayload{JobID: jobID, OutboxID: outboxID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeJobProcess, payload), nil
}

func ParseJobProcessPayload(task *asynq.Task) (*JobProcessPayload, error) {
	var payload JobProcessPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
