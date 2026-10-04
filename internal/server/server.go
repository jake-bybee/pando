package server

import (
	"fmt"
	"net/http"

	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/immich"
	"pando/internal/peers"
	"pando/internal/utils"
)

type Server struct {
	client       *http.Client
	envVariables *config.Config
	peersStore   *peers.Store
	immichStore  *immich.Store
	healthStore  *health.Store
	utilsStore   *utils.Store
}

func NewServer(envVariables *config.Config, client *http.Client, immichStore *immich.Store) *Server {
	peersStore := peers.NewStore(client)
	healthStore := health.NewStore(envVariables, immichStore)
	utilsStore := utils.NewStore(envVariables.TimeZone)

	return &Server{
		client:       client,
		envVariables: envVariables,
		peersStore:   peersStore,
		immichStore:  immichStore,
		healthStore:  healthStore,
		utilsStore:   utilsStore,
	}
}

func (s *Server) Start() error {

	s.RegisterRoutes()

	fmt.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", nil)
}
