package main

import (
	"log"
	"net/http"
	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/immich"
	"pando/internal/peers"
	"pando/internal/server"
	"pando/internal/utils"
)

func main() {
	var client *http.Client = &http.Client{}

	envVariables, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
	log.Println("Loaded environment variables")
	var utilsStore *utils.Store = utils.NewStore(client, envVariables.TimeZone)
	immichStore := immich.NewStore(envVariables, client, utilsStore)
	var peersStore *peers.Store = peers.NewStore(utilsStore, client)

	healthStore := health.NewStore(envVariables, client, immichStore)
	reachable := healthStore.RunHealthCheck()
	if !reachable {
		log.Fatal("Health check failed!!!")
	}
	log.Print("Health check passed, Immich is reachable")

	server := server.NewServer(healthStore, utilsStore, peersStore, immichStore)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
