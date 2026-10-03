package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/immich"
	"pando/internal/peers"
	"pando/internal/server"
	"pando/internal/system"
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
	immich.Init(envVariables, client)
	system.Init(envVariables)

	if !health.RunHealthCheck(envVariables) {
		log.Fatalf("Health check failed")
	}
	fmt.Println("Health check passed")
	peers.Me(envVariables.SelfUrl, "healthy")

	server := server.NewServer(envVariables, client)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
