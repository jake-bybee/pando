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
	"pando/internal/service"
	"pando/internal/utils"
)

func main() {
	var transport = &http.Transport{
		IdleConnTimeout:     30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	}
	var client = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	envVariables, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
	immichStore := immich.NewStore(envVariables, client)
	healthStore := health.NewStore(envVariables, immichStore)
	utilsStore := utils.NewStore(envVariables.TimeZone)

	if !healthStore.RunHealthCheck() {
		log.Fatalf("Health check failed")
	}
	fmt.Println("Health check passed")
	peers.Me(envVariables.SelfUrl, "healthy", utilsStore, envVariables.BackupFolderPath)

	srv := server.NewServer(envVariables, client, immichStore)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Println("Starting service...")

	service.NewStore(envVariables, immichStore).Start()

}
