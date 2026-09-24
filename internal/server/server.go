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
	healthStore  *health.Store
	peersStore   *peers.Store
	immichStore  *immich.Store
}

func NewServer(envVariables *config.Config, client *http.Client) *Server {
	immichStore := immich.NewStore(envVariables, client)
	healthStore := health.NewStore(envVariables, client, immichStore)
	peersStore := peers.NewStore(client)
	utils.Init(envVariables.TimeZone)

	return &Server{client: client, envVariables: envVariables, healthStore: healthStore, peersStore: peersStore, immichStore: immichStore}
}

func (s *Server) Start() error {
	if !runServerPrecheck(s.healthStore) {
		return fmt.Errorf("server precheck failed")
	}
	fmt.Println("Server precheck passed")

	s.RegisterRoutes()

	fmt.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", nil)
}
