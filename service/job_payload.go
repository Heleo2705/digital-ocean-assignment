package service

type JobPayload struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	WebhookURL     string `json:"webhook_url"`
	Version        int    `json:"version"`
	MaxRetries     int    `json:"max_retries"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func NewJobPayload(jobType, name, webhookURL string, version, maxRetries, timeoutSeconds *int) JobPayload {
	payload := JobPayload{
		Type:           jobType,
		Name:           name,
		WebhookURL:     webhookURL,
		Version:        1,
		MaxRetries:     5,
		TimeoutSeconds: 15,
	}
	if version != nil && *version > 0 {
		payload.Version = *version
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
