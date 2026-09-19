package main

import (
	"log"
	"pando/internal/config"
	"pando/internal/health"
)

func main() {
	envVariables, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}

	log.Println("Loaded environment variables")

	reachable := health.RunHealthCheck(envVariables)
	if !reachable {
		log.Fatalf("%s is not reachable", envVariables.ImmichUrl)
	}
}
