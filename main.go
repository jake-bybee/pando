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
		"0011a549-791e-4595-860c-183ca5ed4b1f",
		"0016606a-38b8-4dd0-815d-1967a511faed",
		"0018fa90-a79b-4d0a-85f6-1d4764004d76",
		"0019e523-9040-44b3-a3c0-b48f9419eddf",
		"0019f5d5-28fc-403d-bc8d-4efd425147e3",
		"001d94e8-6ed1-492e-ada6-a8fd262ffc98",
		"001da5d3-4fac-4224-a1d6-9f56096c3e44",
		"001dedc1-961a-44c4-b731-630daffd074c",
		"001df117-ec18-4dc3-b089-58d94abec76b",
		"001fa253-5ba3-4126-9bc7-faacce60f1bc",
		"0020ad3c-0177-421b-90e2-27bb429cfc39",
		"00229792-6f76-4310-927d-e9dc1bcd9a9f",
		"00272955-9a61-4bf9-9f2f-8b40e9cf4082",
		"00285e98-52e4-46e8-9fe2-d3c7a1c94f67",
		"0028a532-1185-438f-9974-2c79fddb7949",
		"002d10a4-a377-4f6f-b2ab-0385cd112544",
		"002d8741-0b1d-4645-88fd-79e79733bedb",
		"002e0fde-146c-41cc-b750-5eb082b00417",
		"002e35a5-17df-48c9-a2bf-320cca6b08ba",
		"0031cd32-6aa6-4a50-a339-bf5a772a0a3b",
		"00372477-4035-41f2-9c75-8e68fe2cf1a0",
		"0038ace5-252f-4b6c-b974-b5abb8ebbed3",
		"003ac0de-6b09-45cf-8abe-1e1a17698d87",
		"003ada98-3f47-4d44-8db0-0b909036cd9f",
		"003b1c5e-861b-4afe-96fc-7a9c353ffd7d",
		"003ce005-5758-47fd-ab08-91d0af6ba8f4",
		"003f13fb-ad31-45d9-b683-c51708c4b665",
		"003ffbbc-9e72-4628-b9d1-91bbd9115e25",
		"00453920-3402-413a-89a0-65e9754cfdea",
		"00479ce7-b281-4492-b9e8-356ffed80c4c",
		"0049eb41-e51f-4fdf-981a-da0c4abcdfb5",
		"004e8a2f-145b-4e2a-9eb3-952917981b10",
		"004f12e5-cda3-4c26-8ab5-42d37c0f88a6",
		"0052aa7e-ba3b-4c61-af57-767992122ab3",
		"0052ace8-aa4f-43bb-8a6b-e7bd7120fd75",
		"005e94b5-1fbd-41bb-a00f-fb8eccfac7eb",
		"005edd9d-9378-4857-b764-9ac5938195b6",
		"005f4b41-60e7-4eb8-848f-60c9b15f5d2e",
		"006115bc-86ea-4c9b-a804-e0b490b37e68",
		"00645bdc-cd54-4a8a-9ce5-05c0f2fafe0a",
		"00647ae3-c48e-439d-b9cd-b9b31729badc",
		"00650dd5-9d4a-4c08-82cc-501e3739dded",
		"00657fcf-d478-4d48-a530-48404c45ff97",
		"00677592-3c8b-4d0d-84af-8cc80a83da66",
		"006a2480-0498-4052-a97a-292224c7715d",
		"006afb5d-e11b-4c85-a6d2-46cf861adf4f",
		"006bcb6a-bbbc-46cc-93d7-38ceabdcd8f1",
		"006df7ab-b209-46f7-84ac-0a4157d959fa",
		"006e18fb-5b44-4006-8a83-c293e019c254",
		"0070e797-6bae-4526-b751-f0ad8bce59e0",
		"00753d66-cd87-4de4-9b63-ad5d06dc168a",
		"00798e4c-2238-4595-91dc-957a0e667d8b",
		"0079d08b-0ced-40e6-b37c-6da43f7973d9",
		"007d6412-8556-4132-9d09-6b2a57f5c6ca",
		"007f3379-b345-42f7-988b-d46519fdb91c",
		"00823baa-ab94-44af-a336-adf926896590",
		"00876f9a-ab39-416d-81a2-aa7b07d08589",
		"008ab6f3-6ed8-4991-9cd7-4ff779bd9fdc",
		"008e1893-cb61-4feb-9250-01bf4c7c6673",
		"00923b6b-f611-4be1-b0dc-09141854ca9a",
		"0092d6e8-f385-4f50-bd6b-c4cf36b75f3a",
		"009488c5-fda5-40fc-978e-12effa6e836e",
		"009976ea-eb07-4ddd-a42c-8bb51a8ad98c",
		"009c7e7f-43dd-4cdf-96bb-cbb28e22adba",
		"00a13e4b-0c35-4566-a9a7-1c1d967b747b",
		"00a28633-3097-41fe-80a4-239fc72d5e6d",
		"00b231a8-2d95-46d2-a20d-4e92f07c91ba",
		"00b37672-577d-4350-afb8-5a6694a2c47b",
		"00b40cbc-81f4-4138-ab35-a0c97f6b9bd8",
		"00b8ce18-db81-4e1d-8411-975964ae30ec",
		"00bae1f0-7d9d-446b-af6f-09357dfcf9af",
		"00bb4251-5dfd-4a17-8c24-fe7dc7113c87",
		"00bb9382-4a51-47b6-9b17-c408003ef89d",
		"00c09464-3efa-4b08-8731-2882137cd985",
		"00c14ba1-a7fe-475a-9a44-a398317d48c0",
		"00c3d8a1-c74c-40d0-beb3-16151972b84b",
		"00c40d31-2802-4fc7-810c-ddb6a5cbe236",
		"00c5756d-87ee-48b6-b073-9be8da759f26",
		"00c587e6-2b40-4f4b-854a-1cd7312cced4",
		"00c5be7b-1f87-4be9-ad7c-9d02cc676038",
		"00c74bbc-5120-4727-b2b2-85285b0822c0",
		"00c80e12-6c87-44d7-93b7-e80b87e4ad72",
		"00c8f318-ac90-4619-bba6-5a14ca415fa0",
		"00c91f31-f0cd-4ddc-a673-8993fba712bb",
		"00cbd43c-9e05-450b-b1d7-52890db7ef13",
		"00d0dd5a-4b6e-4f47-8694-5c0e7ea860e1",
		"00d3d90a-68c4-4497-8b11-5b4c28b7347e",
		"00d4f0e8-06ba-4786-8292-2ddc94c767a3",
		"00d648c2-c033-4f07-94b6-aa43cbf9ec31",
	}} // Replace with actual asset IDs

	batchInfo, err := immichStore.GetInfoForBatchDownload(payload)
	if err != nil {
		log.Fatalf("Failed to get batch download info: %v", err)
	}
	log.Printf("Batch download info: %+v", batchInfo)

	immichStore.DownloadAllBatches(batchInfo, immich.DetermineHowManyBatches(batchInfo))

}
