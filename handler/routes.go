package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Heleo2705/assignment/db"
	"github.com/Heleo2705/assignment/middleware"
	"github.com/Heleo2705/assignment/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler struct {
	store *db.Store
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerJobRequest struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	WebhookURL     string `json:"webhook_url"`
	MaxRetries     *int   `json:"max_retries,omitempty"`
	TimeoutSeconds *int   `json:"timeout_seconds,omitempty"`
}

type jsonResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func New(store *db.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Get("/health", h.healthHandler)
	r.Post("/register", h.registerHandler)
	r.Post("/login", h.loginHandler)

	r.Route("/jobs", func(r chi.Router) {
		r.Post("/", h.createJobHandler)
		r.Get("/", h.listJobsHandler)
		r.Get("/{jobID}", h.getJobHandler)
		r.Put("/{jobID}", h.updateJobHandler)
		r.Delete("/{jobID}", h.deleteJobHandler)
	})
}

func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	logger.Info("health check succeeded")
	writeJSON(w, http.StatusOK, jsonResponse{Message: "ok"})
}

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

	hashedPassword, err := service.HashPassword(req.Password)
	if err != nil {
		logger.Error("failed to hash password", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to create user"})
		return
	}

	userID, err := h.store.CreateUser(r.Context(), req.Email, string(hashedPassword))
	if err != nil {
		logger.Error("failed to create user", zap.Error(err), zap.String("email", req.Email))
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Message: "failed to create user"})
		return
	}

	logger.Info("user registered", zap.String("user_id", userID), zap.String("email", req.Email))
	writeJSON(w, http.StatusCreated, jsonResponse{Message: "registration successful", Data: map[string]string{"user_id": userID, "access_token": "TODO_ACCESS_TOKEN", "refresh_token": "TODO_REFRESH_TOKEN"}})
}

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

	logger.Info("login succeeded", zap.String("user_id", user.ID), zap.String("email", req.Email))
	writeJSON(w, http.StatusOK, jsonResponse{Message: "login successful", Data: map[string]string{"user_id": user.ID, "access_token": "TODO_ACCESS_TOKEN", "refresh_token": "TODO_REFRESH_TOKEN"}})
}

func (h *Handler) createJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		logger.Warn("create job failed: missing user id header")
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Message: "missing user id"})
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

	jobPayload := service.NewJobPayload(req.Type, req.Name, req.WebhookURL, req.MaxRetries, req.TimeoutSeconds)
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

	job, existing, err := h.store.CreateJobWithIdempotency(r.Context(), userID, requestHash, "POST", "/jobs", db.CreateJobParams{
		UserID:     userID,
		Type:       req.Type,
		Name:       req.Name,
		WebhookURL: req.WebhookURL,
		Payload:    payloadBytes,
		MaxRetries: jobPayload.MaxRetries,
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

func (h *Handler) listJobsHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	logger.Info("list jobs requested")
	writeJSON(w, http.StatusNotImplemented, jsonResponse{Message: "list jobs endpoint placeholder"})
}

func (h *Handler) getJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	jobID := chi.URLParam(r, "jobID")
	logger.Info("get job requested", zap.String("job_id", jobID))
	writeJSON(w, http.StatusNotImplemented, jsonResponse{Message: "get job endpoint placeholder", Data: map[string]string{"jobID": jobID}})
}

func (h *Handler) updateJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	jobID := chi.URLParam(r, "jobID")
	logger.Info("update job requested", zap.String("job_id", jobID))
	writeJSON(w, http.StatusNotImplemented, jsonResponse{Message: "update job endpoint placeholder", Data: map[string]string{"jobID": jobID}})
}

func (h *Handler) deleteJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	jobID := chi.URLParam(r, "jobID")
	logger.Info("delete job requested", zap.String("job_id", jobID))
	writeJSON(w, http.StatusNotImplemented, jsonResponse{Message: "delete job endpoint placeholder", Data: map[string]string{"jobID": jobID}})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
