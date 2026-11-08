package server

import (
	"embed"
	"net/http"
)

// The dashboard is intentionally unauthenticated because Pando nodes are
// expected to be reachable only through the private NetBird network.
//
//go:embed dashboard/index.html
var dashboardFiles embed.FS

func (s *Server) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	data, err := dashboardFiles.ReadFile("dashboard/index.html")
	if err != nil {
		http.Error(w, "Dashboard unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
