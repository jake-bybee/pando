package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) RegisterRoutes() {
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

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		data, err := s.immichStore.GetServerStatistics()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get server statistics: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode server statistics: %v", err), http.StatusInternalServerError)
			return
		}

	})
}
