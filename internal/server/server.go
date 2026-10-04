package server

import (
	"fmt"
	"net"
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

	addr := ":" + s.envVariables.ListenPort
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Println("Starting server on", addr)

	go func() {
		if err := s.relayMeToPeer(s.envVariables.MasterPandoUrl, peers.GetMe().Id); err != nil {
			fmt.Printf("Failed to relay to master peer: %v\n", err)
		}
	}()

	return http.Serve(ln, nil)
}
