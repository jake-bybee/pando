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
		data, err := s.immichStore.BulkDownload([]string{
			"87ed1adc-f219-4d5d-9f5b-7e2c413e3d34",
			"c1a5da5b-d7fe-4e88-9f9b-62cde8306272",
			"f4654692-4ee7-4c16-bf6f-eaf4b88c87ef",
			"f2fa941d-d493-47cb-af18-8a3883390203",
			"728b9f87-73e1-4999-a977-f0d427f37c00",
		})
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
