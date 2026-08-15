package api

import (
	"net/http"

	"github.com/krishnabhardwaj25/flowithgo/internal/store"
)

type Server struct {
	mux         *http.ServeMux
	jobStore    *store.JobStore
	broadcaster *Broadcaster
}

func NewServer(jobStore *store.JobStore) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		jobStore:    jobStore,
		broadcaster: NewBroadcaster(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /jobs", s.HandleCreateJob)
	s.mux.HandleFunc("GET /jobs/{id}", s.HandleGetJob)
	s.mux.HandleFunc("GET /dlq", s.HandleGetDeadJobs)
	s.mux.HandleFunc("POST /dlq/{id}/requeue", s.HandleRequeueJob)
	s.mux.HandleFunc("GET /stats", s.HandleGetStats)
	s.mux.HandleFunc("GET /events", s.broadcaster.HandleSSE)
	s.mux.HandleFunc("GET /dashboard", s.HandleDashboard)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) GetBroadcaster() *Broadcaster {
	return s.broadcaster
}