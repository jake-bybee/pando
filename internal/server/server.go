package server

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/immich"
	"pando/internal/peers"
	"pando/internal/status"
	"pando/internal/utils"
)

type Server struct {
	client       *http.Client
	envVariables *config.Config
	peersStore   *peers.Store
	immichStore  *immich.Store
	healthStore  *health.Store
	utilsStore   *utils.Store
	status       *status.Tracker
	version      string
	statusMu     sync.Mutex
	statusCache  []byte
	statusCached time.Time
	healthMu     sync.Mutex
	healthCached time.Time
	healthValid  bool
}

func NewServer(envVariables *config.Config, client *http.Client, immichStore *immich.Store, trackers ...any) *Server {
	peersStore := peers.NewStore(client)
	healthStore := health.NewStore(envVariables, immichStore)
	utilsStore := utils.NewStore(envVariables.TimeZone)
	var tracker *status.Tracker
	version := "dev"
	for _, value := range trackers {
		switch typed := value.(type) {
		case *status.Tracker:
			tracker = typed
		case string:
			version = typed
		}
	}

	return &Server{
		client:       client,
		envVariables: envVariables,
		peersStore:   peersStore,
		immichStore:  immichStore,
		healthStore:  healthStore,
		utilsStore:   utilsStore,
		status:       tracker,
		version:      version,
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
