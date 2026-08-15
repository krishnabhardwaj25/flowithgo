package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.jobStore.GetQueueStats()
	if err != nil {
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}