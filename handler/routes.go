package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Heleo2705/assignment/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerJobRequest struct {
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	WebhookURL string          `json:"webhook_url"`
	Payload    json.RawMessage `json:"payload"`
}

type jsonResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func RegisterRoutes(r *chi.Mux) {
	r.Get("/health", healthHandler)
	r.Post("/register", registerHandler)
	r.Post("/login", loginHandler)

	r.Route("/jobs", func(r chi.Router) {
		r.Post("/", createJobHandler)
		r.Get("/", listJobsHandler)
		r.Get("/{jobID}", getJobHandler)
		r.Put("/{jobID}", updateJobHandler)
		r.Delete("/{jobID}", deleteJobHandler)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	logger.Info("health check succeeded")
	writeJSON(w, http.StatusOK, jsonResponse{Message: "ok"})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
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

	logger.Info("registration requested", zap.String("email", req.Email))
	writeJSON(w, http.StatusCreated, jsonResponse{Message: "registration accepted", Data: map[string]string{"email": req.Email, "access_token": "TODO_ACCESS_TOKEN", "refresh_token": "TODO_REFRESH_TOKEN"}})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
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

	logger.Info("login requested", zap.String("email", req.Email))
	writeJSON(w, http.StatusOK, jsonResponse{Message: "login accepted", Data: map[string]string{"email": req.Email, "access_token": "TODO_ACCESS_TOKEN", "refresh_token": "TODO_REFRESH_TOKEN"}})
}

func createJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	var req registerJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("create job failed: invalid request payload", zap.Error(err))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "invalid request payload"})
		return
	}
	if req.Type == "" || req.Name == "" || req.WebhookURL == "" || len(req.Payload) == 0 {
		logger.Warn("create job failed: missing required fields", zap.String("type", req.Type), zap.String("name", req.Name), zap.String("webhook_url", req.WebhookURL))
		writeJSON(w, http.StatusBadRequest, jsonResponse{Message: "type, name, webhook_url, and payload are required"})
		return
	}

	logger.Info("job registration requested", zap.String("job_type", req.Type), zap.String("job_name", req.Name), zap.String("webhook_url", req.WebhookURL))
	writeJSON(w, http.StatusAccepted, jsonResponse{Message: "job registration accepted", Data: map[string]string{"job_id": "TODO_JOB_ID", "status": "pending"}})
}

func listJobsHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	logger.Info("list jobs requested")
	writeJSON(w, http.StatusNotImplemented, jsonResponse{Message: "list jobs endpoint placeholder"})
}

func getJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	jobID := chi.URLParam(r, "jobID")
	logger.Info("get job requested", zap.String("job_id", jobID))
	writeJSON(w, http.StatusNotImplemented, jsonResponse{Message: "get job endpoint placeholder", Data: map[string]string{"jobID": jobID}})
}

func updateJobHandler(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	jobID := chi.URLParam(r, "jobID")
	logger.Info("update job requested", zap.String("job_id", jobID))
	writeJSON(w, http.StatusNotImplemented, jsonResponse{Message: "update job endpoint placeholder", Data: map[string]string{"jobID": jobID}})
}

func deleteJobHandler(w http.ResponseWriter, r *http.Request) {
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
