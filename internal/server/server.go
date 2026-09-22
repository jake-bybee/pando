package server

import (
	"fmt"
	"net/http"
	"pando/internal/health"
	"pando/internal/immich"
	"pando/internal/peers"
	"pando/internal/utils"
)

type Server struct {
	healthStore *health.Store
	utils       *utils.Store
	peersStore  *peers.Store
	immichStore *immich.Store
}

func NewServer(healthStore *health.Store, utils *utils.Store, peersStore *peers.Store, immichStore *immich.Store) *Server {

	return &Server{healthStore: healthStore, utils: utils, peersStore: peersStore, immichStore: immichStore}
}

func (s *Server) Start() error {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		s.HealthHandler(w, r)
	})

	http.HandleFunc("/registerPeer", func(w http.ResponseWriter, r *http.Request) {
		s.RegisterPeerHandler(w, r)
	})

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		data, err := s.immichStore.BulkDownload(
			[]string{

				"00012798-03fa-478a-a79e-e54641afdfd6",
				"000212d9-2eb7-41e9-907b-dc819d139e9b",
				"0002ef04-6445-4440-9c65-61fdc18c352d",
			},
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get file sizes data: %v", err), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "File sizes data: %v", data)
	})

	fmt.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", nil)
}
