package main

import (
	"log"

	"pando/internal/config"
)

func main() {
	envVariables, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}

	log.Printf("Loaded environment variables: %+v", envVariables)

}
