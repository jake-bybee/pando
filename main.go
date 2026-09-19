package main

import (
	"log"
	"net/http"
	"pando/internal/config"
	"pando/internal/health"
)

func main() {
	envVariables, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
	log.Println("Loaded environment variables")

	healthStore := health.NewStore(envVariables, &http.Client{})
	reachable := healthStore.RunHealthCheck()
	if !reachable {
		log.Fatalf("%s is not reachable", envVariables.ImmichUrl)
	}

	log.Printf("%s is reachable", envVariables.ImmichUrl)

}
