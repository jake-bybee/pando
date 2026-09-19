package main

import (
	"log"
	"net/http"
	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/immich"
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

	immichStore := immich.NewStore(envVariables, client)

	payload := immich.BatchDownloadInfoPayload{AssetIds: []string{
		"fdde3012-3d1a-4bbb-b3d2-918565d2be66",
	}} // Replace with actual asset IDs

	batchInfo, err := immichStore.GetInfoForBatchDownload(payload)
	if err != nil {
		log.Fatalf("Failed to get batch download info: %v", err)
	}
	log.Printf("Batch download info: %+v", batchInfo)

}
