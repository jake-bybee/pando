package main

import (
	"log"
	"net/http"
	"time"

	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/immich"
	"pando/internal/peers"
	"pando/internal/server"
	"pando/internal/utils"
)

func main() {
	var transport *http.Transport = &http.Transport{
		IdleConnTimeout:     30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}
	var client *http.Client = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	envVariables, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
	log.Println("Loaded environment variables")
	utils.Init(envVariables.TimeZone)

	immichStore := immich.NewStore(envVariables, client)
	var peersStore *peers.Store = peers.NewStore(client)

	healthStore := health.NewStore(envVariables, client, immichStore)
	reachable := healthStore.RunHealthCheck()
	if !reachable {
		log.Fatal("Health check failed!!!")
	}
	log.Print("Health check passed, Immich is reachable")

	server := server.NewServer(healthStore, peersStore, immichStore)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
