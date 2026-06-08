package service

type JobPayload struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	WebhookURL     string `json:"webhook_url"`
	MaxRetries     int    `json:"max_retries"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func NewJobPayload(jobType, name, webhookURL string, maxRetries, timeoutSeconds *int) JobPayload {
	payload := JobPayload{
		Type:           jobType,
		Name:           name,
		WebhookURL:     webhookURL,
		MaxRetries:     5,
		TimeoutSeconds: 15,
	}
	if maxRetries != nil && *maxRetries > 0 {
		payload.MaxRetries = *maxRetries
	}
	if timeoutSeconds != nil && *timeoutSeconds > 0 {
		payload.TimeoutSeconds = *timeoutSeconds
	}
	return payload
}

func (p JobPayload) CanonicalJSON() ([]byte, error) {
	return CanonicalJSON(p)
}
