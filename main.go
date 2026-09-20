package main

import (
	"log"
	"net/http"
	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/utils"
)

func main() {
	var client *http.Client = &http.Client{}

	envVariables, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
	log.Println("Loaded environment variables")
	var utils *utils.Store = utils.NewStore(envVariables, client)

	healthStore := health.NewStore(envVariables, client, utils)
	reachable := healthStore.RunHealthCheck()
	if !reachable {
		log.Fatalf("%s is not reachable", envVariables.ImmichUrl)
	}

	log.Printf("%s is reachable", envVariables.ImmichUrl)

}
