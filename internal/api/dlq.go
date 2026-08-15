package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) HandleGetDeadJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.jobStore.GetDeadJobs()
	if err != nil {
		http.Error(w, "failed to get dead jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (s *Server) HandleRequeueJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	err := s.jobStore.RequeueJob(id)
	if err != nil {
		http.Error(w, "failed to requeue job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "job requeued successfully",
		"id":      id,
	})
}