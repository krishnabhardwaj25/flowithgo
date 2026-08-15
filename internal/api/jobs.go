package api

import (
	"encoding/json"
	"net/http"
    "log"
	"github.com/krishnabhardwaj25/flowithgo/internal/models"
)

func (s *Server) HandleCreateJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type        string          `json:"type"`
		Payload     json.RawMessage `json:"payload"`
		MaxAttempts int             `json:"max_attempts"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Type == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}

	if body.MaxAttempts == 0 {
		body.MaxAttempts = 3
	}

	job := models.Job{
		Type:        body.Type,
		Payload:     []byte(body.Payload),
		MaxAttempts: body.MaxAttempts,
	}

	inserted, err := s.jobStore.InsertJob(job)
	if err != nil {
		log.Printf("failed to create job: %v", err)
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inserted)
}

func (s *Server) HandleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	job, err := s.jobStore.GetJobByID(id)
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}