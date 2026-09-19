package immich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pando/internal/config"
)

type Store struct {
	client *http.Client
	config config.Config
}

type BatchDownloadInfoResponse struct {
	TotalSize int `json:"totalSize"`
	Archives  []struct {
		Size     int      `json:"size"`
		AssetIds []string `json:"assetIds"`
	} `json:"archives"`
}

type BatchDownloadInfoPayload struct {
	AssetIds []string `json:"assetIds"`
}

func NewStore(config config.Config, client *http.Client) *Store {
	return &Store{
		client: client,
		config: config,
	}
}

func (s *Store) GetInfoForBatchDownload(payload BatchDownloadInfoPayload) (BatchDownloadInfoResponse, error) {
	endpoint := "/api/download/info"
	fullUrl := s.config.ImmichUrl + endpoint
	log.Printf("Requesting batch download info from %s", endpoint)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return BatchDownloadInfoResponse{}, fmt.Errorf("failed to marshal payload for %s: %v", fullUrl, err)
	}

	req, err := http.NewRequest("POST", fullUrl, bytes.NewReader(payloadBytes))
	if err != nil {
		return BatchDownloadInfoResponse{}, fmt.Errorf("failed to create request for %s: %v", fullUrl, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.config.ImmichApiToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return BatchDownloadInfoResponse{}, fmt.Errorf("failed to execute request for %s: %v", fullUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return BatchDownloadInfoResponse{}, fmt.Errorf("unexpected response status for %s: %v", fullUrl, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return BatchDownloadInfoResponse{}, fmt.Errorf("failed to read response body for %s: %v", fullUrl, err)
	}

	var batchInfo BatchDownloadInfoResponse
	err = json.Unmarshal(body, &batchInfo)
	if err != nil {
		return BatchDownloadInfoResponse{}, fmt.Errorf("failed to unmarshal response body for %s: %v", fullUrl, err)
	}

	return batchInfo, nil
}

func determineHowManyBatches(batchInfo BatchDownloadInfoResponse) int {
	return len(batchInfo.Archives)
}

func (s *Store) bulkDownload(assetIds []string, numBatches int) {

	endpoint := "/api/download/archive"
	fullUrl := s.config.ImmichUrl + endpoint

	log.Printf("Preparing to bulk download %d assets in %d batches from %s", len(assetIds), numBatches, fullUrl)

}

func (s *Store) DownloadAllBatches(batchInfo BatchDownloadInfoResponse, numBatches int) {

	for i := 0; i < numBatches; i++ {
		archive := batchInfo.Archives[i]
		log.Printf("Downloading archive %d with size %d and asset IDs: %+v", i, archive.Size, archive.AssetIds)
		// Implement the actual download logic here
	}
}
