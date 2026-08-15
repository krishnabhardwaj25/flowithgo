package api

import (
	"net/http"
)

func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/dashboard.html")
}