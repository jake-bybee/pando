package immich

import (
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

type BatchDownloadInfo struct {
	TotalSize int `json:"totalSize"`
	Archives  []struct {
		Size     int      `json:"size"`
		AssetIds []string `json:"assetIds"`
	} `json:"archives"`
}

func (s *Store) getInfoForBatchDownload() (BatchDownloadInfo, error) {
	endpoint := "/api/download/info"
	fullUrl := s.config.ImmichUrl + endpoint
	log.Printf("Requesting batch download info from %s", endpoint)
	resp, err := s.client.Get(fullUrl)
	if err != nil {
		return BatchDownloadInfo{}, fmt.Errorf("failed to execute request for %s: %v", fullUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BatchDownloadInfo{}, fmt.Errorf("unexpected response status for %s: %v", fullUrl, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return BatchDownloadInfo{}, fmt.Errorf("failed to read response body for %s: %v", fullUrl, err)
	}

	var batchInfo BatchDownloadInfo
	err = json.Unmarshal(body, &batchInfo)
	if err != nil {
		return BatchDownloadInfo{}, fmt.Errorf("failed to unmarshal response body for %s: %v", fullUrl, err)
	}

	return batchInfo, nil
}
