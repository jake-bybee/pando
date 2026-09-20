package server

import (
	"fmt"
	"net/http"
	"pando/internal/health"
	"pando/internal/peers"
	"pando/internal/utils"
)

type Server struct {
	healthStore *health.Store
	utils       *utils.Store
	peersStore  *peers.Store
}

func NewServer(healthStore *health.Store) *Server {

	return &Server{healthStore: healthStore}
}

func (s *Server) Start() error {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		s.HealthHandler(w, r)
	})

	http.HandleFunc("/registerPeer", func(w http.ResponseWriter, r *http.Request) {
		s.RegisterPeerHandler(w, r)
	})

	fmt.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", nil)
}
