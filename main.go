package main

import (
	"log"
	"net/http"
	"time"

	"pando/internal/config"
	"pando/internal/server"
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

	server := server.NewServer(envVariables, client)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
