package server

import (
	"fmt"
	"net/http"

	"pando/internal/config"
	"pando/internal/immich"
	"pando/internal/peers"
)

type Server struct {
	client       *http.Client
	envVariables *config.Config
	peersStore   *peers.Store
	immichStore  *immich.Store
}

func NewServer(envVariables *config.Config, client *http.Client, immichStore *immich.Store) *Server {
	peersStore := peers.NewStore(client)

	return &Server{client: client, envVariables: envVariables, peersStore: peersStore, immichStore: immichStore}
}

func (s *Server) Start() error {

	s.RegisterRoutes()

	fmt.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", nil)
}
