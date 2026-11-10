package server

import (
	"net/http"
)

func (s *Server) RegisterRoutes() {
	http.HandleFunc("/dashboard", s.DashboardHandler)
	http.HandleFunc("/storage", s.StorageHandler)
	http.HandleFunc("/cluster/status", s.ClusterStatusHandler)
	http.HandleFunc("/manifest", s.ManifestHandler)
	http.HandleFunc("/status", s.StatusHandler)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		s.HealthHandler(w, r)
	})

	http.HandleFunc("/registerPeer", func(w http.ResponseWriter, r *http.Request) {
		s.RegisterPeerHandler(w, r)
	})

	http.HandleFunc("/relayPeers", func(w http.ResponseWriter, r *http.Request) {
		s.RelayPeersHandler(w, r)
	})

	http.HandleFunc("/registerPeersReceived", func(w http.ResponseWriter, r *http.Request) {
		s.RegisterPeersReceivedHandler(w, r)
	})

	http.HandleFunc("/doesThisNodeHaveImage", func(w http.ResponseWriter, r *http.Request) {
		s.DoesThisNodeHaveImage(w, r)
	})

	http.HandleFunc("/whichNodeHasThisImage", func(w http.ResponseWriter, r *http.Request) {
		s.WhichNodeHasThisImage(w, r)
	})

}
