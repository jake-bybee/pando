package immich

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"pando/internal/config"
	"pando/internal/utils"
)

var MAX_DOWNLOAD_BATCH_SIZE_BYTES = 1024 * 1024 * 1024 // 1 GB

type Store struct {
	client *http.Client
	config config.Config
	utils  *utils.Store
}

type BatchDownloadInfoResponse struct {
	TotalSize int        `json:"totalSize"`
	Archives  []Archives `json:"archives"`
}

type Archives struct {
	Size     int      `json:"size"`
	AssetIds []string `json:"assetIds"`
}

type BatchDownloadInfoPayload struct {
	AssetIds []string `json:"assetIds"`
}

func NewStore(config config.Config, client *http.Client, utils *utils.Store) *Store {
	return &Store{
		client: client,
		config: config,
		utils:  utils,
	}
}

func (s *Store) GetInfoForBatchDownload(payload BatchDownloadInfoPayload) (BatchDownloadInfoResponse, error) {
	endpoint := "/api/download/info"
	fullUrl := s.config.ImmichUrl + endpoint
	log.Printf("Requesting batch download info from %s", endpoint)

	resp, err := s.utils.ImmichFetcher(fullUrl, "POST", payload)
	if err != nil {
		return BatchDownloadInfoResponse{}, fmt.Errorf("failed to execute request for %s: %v", fullUrl, err)
	}
	defer resp.Body.Close()

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

func DetermineHowManyBatches(batchInfo BatchDownloadInfoResponse) int {
	return len(batchInfo.Archives)
}

func (s *Store) bulkDownload(assetIds []string) {

	endpoint := "/api/download/archive"
	fullUrl := s.config.ImmichUrl + endpoint

	log.Printf("Preparing to bulk download %d assets from %s", len(assetIds), fullUrl)
	resp, err := s.utils.ImmichFetcher(fullUrl, "POST", BatchDownloadInfoPayload{AssetIds: assetIds})
	if err != nil {
		log.Printf("Failed to execute bulk download request for %s: %v", fullUrl, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for bulk download request for %s: %v", fullUrl, err)
		return
	}

	os.WriteFile("test.zip", body, 0644)

}

func (s *Store) DownloadAllBatches(batchInfo BatchDownloadInfoResponse) {
	numBatches := DetermineHowManyBatches(batchInfo)

	for i := 0; i < numBatches; i++ {
		archive := batchInfo.Archives[i]
		if archive.Size > MAX_DOWNLOAD_BATCH_SIZE_BYTES {
			1 == 1
		}

		for i := 0; i < numBatches; i++ {
			archive := batchInfo.Archives[i]
			log.Printf("Downloading archive %d with size %d and asset IDs: %+v", i, archive.Size, archive.AssetIds)
			s.bulkDownload(archive.AssetIds)
		}
	}
}

func (s *Store) splitAssetBatch(archive Archives) ([]Archives, error) {
	var batches []struct {
		AssetIds []string
		Size     int
		Approved bool
	}

	maxSize := archive.Size
	batches = append(batches, struct {
		AssetIds []string
		Size     int
		Approved bool
	}{AssetIds: archive.AssetIds, Size: 0, Approved: false})

	for maxSize > MAX_DOWNLOAD_BATCH_SIZE_BYTES {
		maxSize = 0
		for i := 0; i < len(batches); i++ {
			batch := &batches[i]

			if batch.Approved {
				continue
			}

			batchInfo, err := s.GetInfoForBatchDownload(BatchDownloadInfoPayload{AssetIds: batch.AssetIds})
			if err != nil {
				log.Printf("Failed to get info for batch: %v", err)
				return nil, fmt.Errorf("failed to get info for batch: %v", err)
			}

			batch.Size = batchInfo.TotalSize

			if batchInfo.TotalSize > maxSize {
				maxSize = batchInfo.TotalSize
			}

			if batch.Size <= MAX_DOWNLOAD_BATCH_SIZE_BYTES {
				batch.Approved = true
				continue
			} else {
				batch.Approved = false
			}

			mid := len(batch.AssetIds) / 2

			batches = append(batches, struct {
				AssetIds []string
				Size     int
				Approved bool
			}{AssetIds: batch.AssetIds[:mid], Size: 0, Approved: false})

			batches = append(batches, struct {
				AssetIds []string
				Size     int
				Approved bool
			}{AssetIds: batch.AssetIds[mid:], Size: 0, Approved: false})

			batches = append(batches[:i], batches[i+1:]...)
			i--

		}

	}

	var result []Archives
	for _, batch := range batches {
		result = append(result, Archives{
			AssetIds: batch.AssetIds,
			Size:     batch.Size,
		})
	}

	return result, nil
}
