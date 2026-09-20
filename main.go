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

	immichStore := immich.NewStore(envVariables, client, utils)

	payload := immich.BatchDownloadInfoPayload{AssetIds: []string{

		"00012798-03fa-478a-a79e-e54641afdfd6",
		"000212d9-2eb7-41e9-907b-dc819d139e9b",
		"0002ef04-6445-4440-9c65-61fdc18c352d",
		"000590ed-3f9c-40ca-a255-94dc56b9ec7f",
		"00063a38-0e4a-45ba-bf17-a5cfec3261eb",
		"00082c7a-345d-44aa-b443-52b29feea35e",
		"0008477c-2c48-40c2-9946-ee9057e31b78",
		"0008cf23-2a90-4302-847f-621f8ea8fc61",
		"001071c0-1f0b-4b05-b872-3b88851d6cb1",
		"00112712-8872-4a69-b0ae-31e409f2437c",
		"0011411f-f8d8-4404-9b6e-ce3905bff16a",
	}} // Replace with actual asset IDs

	_, err = immichStore.BulkDownload(payload.AssetIds)
	if err != nil {
		log.Fatalf("Failed to get batch download info: %v", err)
	}

}
