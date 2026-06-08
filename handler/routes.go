package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Heleo2705/assignment/db"
	"github.com/Heleo2705/assignment/middleware"
	"github.com/Heleo2705/assignment/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler struct {
	store *db.Store
}

// authRequest represents registration or login credentials.
// @Description User credentials for registration or login.
type authRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"securePass123"`
	Username string `json:"username,omitempty" example:"johndoe"`
}

// registerJobRequest represents a job registration payload.
// @Description Payload for registering a new async job.
type registerJobRequest struct {
	Type           string `json:"type" example:"email"`
	Name           string `json:"name" example:"welcome-email"`
	WebhookURL     string `json:"webhook_url" example:"http://webhook:9000/echo"`
	Version        *int   `json:"version,omitempty" example:"1"`
	MaxRetries     *int   `json:"max_retries,omitempty" example:"5"`
	TimeoutSeconds *int   `json:"timeout_seconds,omitempty" example:"30"`
}

// updateJobRequest represents a partial job update payload.
// @Description Fields to update on an existing job. Only non-nil fields are applied.
type updateJobRequest struct {
	Name           *string `json:"name,omitempty" example:"rebranded-email"`
	WebhookURL     *string `json:"webhook_url,omitempty" example:"http://webhook:9000/echo"`
	Version        *int    `json:"version,omitempty" example:"2"`
	MaxRetries     *int    `json:"max_retries,omitempty" example:"3"`
	TimeoutSeconds *int    `json:"timeout_seconds,omitempty" example:"60"`
}

// jsonResponse is the standard API response envelope.
// @Description Standard response wrapper used by all endpoints.
type jsonResponse struct {
	Message string      `json:"message" example:"ok"`
	Data    interface{} `json:"data,omitempty"`
}

func getUserID(r *http.Request) string {
	if claims := middleware.GetJWTClaims(r); claims != nil {
		return claims.Subject
	}
	return ""
}

func New(store *db.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(r *chi.Mux, authMiddleware func(http.Handler) http.Handler) {
	r.Get("/health", h.healthHandler)
	r.Post("/register", h.registerHandler)
	r.Post("/login", h.loginHandler)

	r.Route("/jobs", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", h.createJobHandler)
		r.Get("/", h.listJobsHandler)
		r.Get("/{jobID}", h.getJobHandler)
		r.Put("/{jobID}", h.updateJobHandler)
		r.Delete("/{jobID}", h.deleteJobHandler)
	})
}

// healthHandler godoc
// @Summary Health check
// @Description Returns a simple OK response to confirm the API is running.
// @Tags system
// @Produce json
// @Success 200 {object} jsonResponse "API is healthy"
// @Router /health [get]
func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	logger.Info("health check succeeded")
	writeJSON(w, http.StatusOK, jsonResponse{Message: "ok"})
}

// registerHandler godoc
// @Summary Register a new user
// @Description Creates a user account with email and password, returns JWT access and refresh tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body authRequest true "Registration credentials"
// @Success 201 {object} jsonResponse "User created — includes user_id, access_token, refresh_token"
// @Failure 400 {object} jsonResponse "Invalid payload or missing fields"
// @Failure 500 {object} jsonResponse "Server error"
// @Router /register [post]
func (h *Handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("register failed: invalid request payload", zap.Error(err))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "invalid request payload"})
		return
	}
	if req.Email == "" || req.Password == "" {
		logger.Warn("register failed: missing credentials", zap.String("email", req.Email))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "email and password are required"})
		return
	}

	username := req.Username
	if username == "" {
		username = req.Email
	}

	hashedPassword, err := service.HashPassword(req.Password)
	if err != nil {
		logger.Error("failed to hash password", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to create user"})
		return
	}

	userID, err := h.store.CreateUser(r.Context(), req.Email, username, string(hashedPassword))
	if err != nil {
		logger.Error("failed to create user", zap.Error(err), zap.String("email", req.Email))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to create user"})
		return
	}

	accessToken, err := service.GenerateAccessToken(os.Getenv("JWT_SECRET"), userID, req.Email, 15*time.Minute)
	if err != nil {
		logger.Error("failed to generate access token", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to create user"})
		return
	}

	refreshToken, err := service.GenerateRefreshToken(os.Getenv("JWT_SECRET"), userID, 7*24*time.Hour)
	if err != nil {
		logger.Error("failed to generate refresh token", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to create user"})
		return
	}

	logger.Info("user registered", zap.String("user_id", userID), zap.String("email", req.Email))
	writeJSON(w, http.StatusCreated, jsonResponse{Message: "registration successful", Data: map[string]string{"user_id": userID, "access_token": accessToken, "refresh_token": refreshToken}})
}

// loginHandler godoc
// @Summary Authenticate a user
// @Description Authenticates with email and password, returns JWT access and refresh tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body authRequest true "Login credentials"
// @Success 200 {object} jsonResponse "Login OK — includes user_id, access_token, refresh_token"
// @Failure 400 {object} jsonResponse "Invalid payload or missing fields"
// @Failure 401 {object} jsonResponse "Invalid email or password"
// @Router /login [post]
func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("login failed: invalid request payload", zap.Error(err))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "invalid request payload"})
		return
	}
	if req.Email == "" || req.Password == "" {
		logger.Warn("login failed: missing credentials", zap.String("email", req.Email))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "email and password are required"})
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		logger.Error("failed to fetch user", zap.Error(err), zap.String("email", req.Email))
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "invalid credentials"})
		return
	}
	if err := service.ComparePassword(user.PasswordHash, req.Password); err != nil {
		logger.Warn("login failed: invalid credentials", zap.String("email", req.Email))
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "invalid credentials"})
		return
	}

	accessToken, err := service.GenerateAccessToken(os.Getenv("JWT_SECRET"), user.ID, req.Email, 15*time.Minute)
	if err != nil {
		logger.Error("failed to generate access token", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to login"})
		return
	}

	refreshToken, err := service.GenerateRefreshToken(os.Getenv("JWT_SECRET"), user.ID, 7*24*time.Hour)
	if err != nil {
		logger.Error("failed to generate refresh token", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to login"})
		return
	}

	logger.Info("login succeeded", zap.String("user_id", user.ID), zap.String("email", req.Email))
	writeJSON(w, http.StatusOK, jsonResponse{Message: "login successful", Data: map[string]string{"user_id": user.ID, "access_token": accessToken, "refresh_token": refreshToken}})
}

// createJobHandler godoc
// @Summary Create a job
// @Description Registers an async job to be processed. **Idempotent** — sending the same payload twice returns the existing job with HTTP 200. The idempotency hash is computed from canonical (sorted-key) JSON of all fields.
// @Tags jobs
// @Accept json
// @Produce json
// @Param body body registerJobRequest true "Job registration payload"
// @Success 202 {object} jsonResponse "Job accepted for processing — returns job_id and state"
// @Success 200 {object} jsonResponse "Job already exists (idempotent match) — returns existing job_id and state"
// @Failure 400 {object} jsonResponse "Invalid payload or missing required fields"
// @Failure 401 {object} jsonResponse "Missing or invalid Authorization header"
// @Failure 500 {object} jsonResponse "Server error"
// @Router /jobs [post]
func (h *Handler) createJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	userID := getUserID(r)
	if userID == "" {
		logger.Warn("create job failed: missing auth claims")
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "unauthorized"})
		return
	}

	var req registerJobRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		logger.Warn("create job failed: invalid request payload", zap.Error(err))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "invalid request payload"})
		return
	}
	if req.Type == "" || req.Name == "" || req.WebhookURL == "" {
		logger.Warn("create job failed: missing required fields", zap.String("type", req.Type), zap.String("name", req.Name), zap.String("webhook_url", req.WebhookURL))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "type, name, and webhook_url are required"})
		return
	}

	jobPayload := service.NewJobPayload(req.Type, req.Name, req.WebhookURL, req.Version, req.MaxRetries, req.TimeoutSeconds)
	requestHash, err := service.ComputeIdempotencyHash(jobPayload)
	if err != nil {
		logger.Error("failed to compute idempotency hash", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to register job"})
		return
	}

	payloadBytes, err := json.Marshal(jobPayload)
	if err != nil {
		logger.Error("failed to marshal job payload", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to register job"})
		return
	}

	job, existing, err := h.store.CreateJobWithIdempotency(r.Context(), userID, requestHash, "POST", "/jobs", db.StoreCreateJobParams{
		UserID:         userID,
		Type:           req.Type,
		Name:           req.Name,
		WebhookURL:     req.WebhookURL,
		Payload:        payloadBytes,
		MaxRetries:     jobPayload.MaxRetries,
		TimeoutSeconds: jobPayload.TimeoutSeconds,
		Version:        jobPayload.Version,
	})
	if err != nil {
		logger.Error("failed to create job", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to register job"})
		return
	}

	logger.Info("job registration processed", zap.String("job_id", job.ID), zap.Bool("idempotent", existing))
	statusCode := http.StatusAccepted
	message := "job registration accepted"
	if existing {
		statusCode = http.StatusOK
		message = "job already registered"
	}
	writeJSON(w, statusCode, jsonResponse{Message: message, Data: map[string]interface{}{"job_id": job.ID, "state": job.State}})
}

// listJobsHandler godoc
// @Summary List user's jobs
// @Description Returns a paginated list of jobs owned by the authenticated user, newest first.
// @Tags jobs
// @Produce json
// @Param page query int false "Page number (default 1)" minimum(1)
// @Param page_size query int false "Items per page (default 20, max 100)" minimum(1) maximum(100)
// @Success 200 {object} jsonResponse "Paginated list of jobs"
// @Failure 401 {object} jsonResponse "Missing or invalid Authorization header"
// @Failure 500 {object} jsonResponse "Server error"
// @Router /jobs [get]
func (h *Handler) listJobsHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	userID := getUserID(r)
	if userID == "" {
		logger.Warn("list jobs failed: missing auth claims")
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	jobs, err := h.store.ListJobsByUser(r.Context(), userID, pageSize, offset)
	if err != nil {
		logger.Error("failed to list jobs", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to list jobs"})
		return
	}

	response := make([]interface{}, 0, len(jobs))
	for _, job := range jobs {
		response = append(response, map[string]interface{}{
			"id":              job.ID,
			"type":            job.Type,
			"name":            job.Name,
			"webhook_url":     job.WebhookURL,
			"version":         job.Version,
			"max_retries":     job.MaxRetries,
			"timeout_seconds": job.TimeoutSeconds,
			"state":           job.State,
			"attempts":        job.Attempts,
			"scheduled_at":    job.ScheduledAt,
			"payload":         json.RawMessage(job.Payload),
			"last_error":      job.LastError.String,
			"result":          json.RawMessage(job.Result),
		})
	}

	writeJSON(w, http.StatusOK, jsonResponse{Message: "jobs listed", Data: response})
}

// getJobHandler godoc
// @Summary Get a job by ID
// @Description Returns full details of a specific job owned by the authenticated user.
// @Tags jobs
// @Produce json
// @Param jobID path string true "Job UUID" example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} jsonResponse "Job details"
// @Failure 401 {object} jsonResponse "Missing or invalid Authorization header"
// @Failure 403 {object} jsonResponse "Job belongs to another user"
// @Failure 404 {object} jsonResponse "Job not found"
// @Router /jobs/{jobID} [get]
func (h *Handler) getJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	userID := getUserID(r)
	if userID == "" {
		logger.Warn("get job failed: missing auth claims")
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "unauthorized"})
		return
	}

	jobID := chi.URLParam(r, "jobID")
	job, err := h.store.GetJobByID(r.Context(), jobID)
	if err != nil {
		logger.Error("failed to fetch job", zap.Error(err), zap.String("job_id", jobID))
		writeJSON(w, http.StatusNotFound, jsonResponse{Message: "job not found"})
		return
	}
	if job.UserID != userID {
		logger.Warn("get job failed: unauthorized job access", zap.String("job_id", jobID), zap.String("user_id", userID))
		writeJSON(w, http.StatusForbidden, jsonResponse{Message: "forbidden"})
		return
	}

	writeJSON(w, http.StatusOK, jsonResponse{Message: "job fetched", Data: map[string]interface{}{
		"id":              job.ID,
		"type":            job.Type,
		"name":            job.Name,
		"webhook_url":     job.WebhookURL,
		"version":         job.Version,
		"max_retries":     job.MaxRetries,
		"timeout_seconds": job.TimeoutSeconds,
		"state":           job.State,
		"attempts":        job.Attempts,
		"scheduled_at":    job.ScheduledAt,
		"payload":         json.RawMessage(job.Payload),
		"last_error":      job.LastError.String,
		"result":          json.RawMessage(job.Result),
	}})
}

// updateJobHandler godoc
// @Summary Update a job
// @Description Partially updates an existing job. Only the fields included in the request body are changed — omitted fields keep their current values.
// @Tags jobs
// @Accept json
// @Produce json
// @Param jobID path string true "Job UUID" example(550e8400-e29b-41d4-a716-446655440000)
// @Param body body updateJobRequest true "Fields to update (all optional — at least one required)"
// @Success 200 {object} jsonResponse "Job updated successfully"
// @Failure 400 {object} jsonResponse "Invalid payload or no fields to update"
// @Failure 401 {object} jsonResponse "Missing or invalid Authorization header"
// @Failure 403 {object} jsonResponse "Job belongs to another user"
// @Failure 404 {object} jsonResponse "Job not found"
// @Failure 500 {object} jsonResponse "Server error"
// @Router /jobs/{jobID} [put]
func (h *Handler) updateJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	userID := getUserID(r)
	if userID == "" {
		logger.Warn("update job failed: missing auth claims")
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "unauthorized"})
		return
	}

	jobID := chi.URLParam(r, "jobID")
	var req updateJobRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		logger.Warn("update job failed: invalid request payload", zap.Error(err))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "invalid request payload"})
		return
	}
	if req.Name == nil && req.WebhookURL == nil && req.Version == nil && req.MaxRetries == nil && req.TimeoutSeconds == nil {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "no fields to update"})
		return
	}

	job, err := h.store.GetJobByID(r.Context(), jobID)
	if err != nil {
		logger.Error("failed to fetch job", zap.Error(err), zap.String("job_id", jobID))
		writeJSON(w, http.StatusNotFound, jsonResponse{Message: "job not found"})
		return
	}
	if job.UserID != userID {
		logger.Warn("update job failed: unauthorized job access", zap.String("job_id", jobID), zap.String("user_id", userID))
		writeJSON(w, http.StatusForbidden, jsonResponse{Message: "forbidden"})
		return
	}

	if req.Name != nil {
		job.Name = *req.Name
	}
	if req.WebhookURL != nil {
		job.WebhookURL = *req.WebhookURL
	}
	if req.Version != nil && *req.Version > 0 {
		job.Version = *req.Version
	}
	if req.MaxRetries != nil && *req.MaxRetries > 0 {
		job.MaxRetries = *req.MaxRetries
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		job.TimeoutSeconds = *req.TimeoutSeconds
	}

	if err := h.store.UpdateJobDetails(r.Context(), job.ID, job.UserID, job.Name, job.WebhookURL, job.MaxRetries, job.TimeoutSeconds, job.Version); err != nil {
		logger.Error("failed to update job", zap.Error(err), zap.String("job_id", jobID))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to update job"})
		return
	}

	writeJSON(w, http.StatusOK, jsonResponse{Message: "job updated"})
}

// deleteJobHandler godoc
// @Summary Delete a job
// @Description Permanently removes a job owned by the authenticated user.
// @Tags jobs
// @Produce json
// @Param jobID path string true "Job UUID" example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} jsonResponse "Job deleted successfully"
// @Failure 401 {object} jsonResponse "Missing or invalid Authorization header"
// @Failure 403 {object} jsonResponse "Job belongs to another user"
// @Failure 404 {object} jsonResponse "Job not found"
// @Failure 500 {object} jsonResponse "Server error"
// @Router /jobs/{jobID} [delete]
func (h *Handler) deleteJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	userID := getUserID(r)
	if userID == "" {
		logger.Warn("delete job failed: missing auth claims")
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "unauthorized"})
		return
	}

	jobID := chi.URLParam(r, "jobID")
	job, err := h.store.GetJobByID(r.Context(), jobID)
	if err != nil {
		logger.Error("failed to fetch job", zap.Error(err), zap.String("job_id", jobID))
		writeJSON(w, http.StatusNotFound, jsonResponse{Message: "job not found"})
		return
	}
	if job.UserID != userID {
		logger.Warn("delete job failed: unauthorized job access", zap.String("job_id", jobID), zap.String("user_id", userID))
		writeJSON(w, http.StatusForbidden, jsonResponse{Message: "forbidden"})
		return
	}

	if err := h.store.DeleteJob(r.Context(), jobID, userID); err != nil {
		logger.Error("failed to delete job", zap.Error(err), zap.String("job_id", jobID))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to delete job"})
		return
	}

	writeJSON(w, http.StatusOK, jsonResponse{Message: "job deleted"})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
