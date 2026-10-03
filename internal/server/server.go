package server

import (
	"fmt"
	"net/http"

	"pando/internal/config"
	"pando/internal/peers"
	"pando/internal/utils"
)

type Server struct {
	client       *http.Client
	envVariables *config.Config
	peersStore   *peers.Store
}

func NewServer(envVariables *config.Config, client *http.Client) *Server {
	peersStore := peers.NewStore(client)
	utils.Init(envVariables.TimeZone)

	return &Server{client: client, envVariables: envVariables, peersStore: peersStore}
}

func (s *Server) Start() error {

	s.RegisterRoutes()

	fmt.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", nil)
}
