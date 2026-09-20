package immich

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"pando/internal/config"
	"pando/internal/utils"
	"strings"
	"sync"
)

var MAX_DOWNLOAD_BATCH_SIZE_BYTES = (1024 * 1024 * 1024) / 2 // 500 MB
const MAX_CONCURRENT_DOWNLOADS = 5

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

func (s *Store) BulkDownload(assetIds []string) (bool, error) {
	if len(assetIds) == 0 {
		log.Printf("No asset IDs provided for bulk download")
		return false, fmt.Errorf("no asset IDs provided for bulk download")
	}
	batchInfo, err := s.getInfoForBatchDownload(BatchDownloadInfoPayload{AssetIds: assetIds})
	if err != nil {
		log.Printf("Failed to get batch download info: %v", err)
		return false, fmt.Errorf("failed to get batch download info: %v", err)
	}

	batchInfo, err = s.getAllDownloadBatches(batchInfo)
	if err != nil {
		log.Printf("Failed to get all download batches: %v", err)
		return false, fmt.Errorf("failed to get all download batches: %v", err)
	}

	err = s.bulkDownloadRoutineSwarm(batchInfo, s.config.BackupFolderPath)
	if err != nil {
		return false, fmt.Errorf("failed to download batches: %v", err)
	}

	return true, nil

}

func (s *Store) bulkDownloadRoutineSwarm(batches BatchDownloadInfoResponse, dest string) error {
	var wg sync.WaitGroup

	type errorResult struct {
		AssetIds []string
		Err      error
	}

	downloadsChannel := make(chan struct{}, MAX_CONCURRENT_DOWNLOADS)
	errorsChannel := make(chan errorResult)

	for _, archive := range batches.Archives {
		wg.Add(1)
		go func(assetIds []string) {
			defer wg.Done()
			downloadsChannel <- struct{}{}
			defer func() { <-downloadsChannel }()
			numFilesDownloaded, err := s.bulkDownload(assetIds, dest)
			if err != nil {
				log.Printf("Failed to bulk download assets %v: %v", assetIds, err)
				errorsChannel <- errorResult{AssetIds: assetIds, Err: err}
			}
			log.Printf("Successfully downloaded %d files to %s", numFilesDownloaded, dest)

		}(archive.AssetIds)
	}
	wg.Wait()
	close(errorsChannel)
	for errResult := range errorsChannel {
		log.Printf("Error downloading assets %v: %v", errResult.AssetIds, errResult.Err)
	}
	return nil
}

func (s *Store) getInfoForBatchDownload(payload BatchDownloadInfoPayload) (BatchDownloadInfoResponse, error) {
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

func determineHowManyBatches(batchInfo BatchDownloadInfoResponse) int {
	return len(batchInfo.Archives)
}

func (s *Store) bulkDownload(assetIds []string, dest string) (int, error) {
	endpoint := "/api/download/archive"
	fullUrl := s.config.ImmichUrl + endpoint

	log.Printf("Preparing to bulk download %d assets from %s", len(assetIds), fullUrl)
	resp, err := s.utils.ImmichFetcher(fullUrl, "POST", BatchDownloadInfoPayload{AssetIds: assetIds})
	if err != nil {
		log.Printf("Failed to execute bulk download request for %s: %v", fullUrl, err)
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for bulk download request for %s: %v", fullUrl, err)
		return 0, err
	}

	fileName := fmt.Sprintf("%x.zip", sha256.Sum256([]byte(strings.Join(assetIds, ","))))
	filePath := s.config.BackupFolderPath + fileName

	err = os.WriteFile(filePath, body, 0644)
	if err != nil {
		log.Printf("Failed to write bulk download file %s: %v", filePath, err)
		return 0, err
	}
	err = s.utils.Unzip(filePath, s.config.BackupFolderPath)
	if err != nil {
		log.Printf("Failed to unzip bulk download file %s: %v", filePath, err)
		return 0, err
	}
	log.Printf("Successfully wrote bulk download file %s", filePath)

	os.Remove(filePath) // Clean up the zip file after extraction

	return len(assetIds), nil
}

func (s *Store) getAllDownloadBatches(batchInfo BatchDownloadInfoResponse) (BatchDownloadInfoResponse, error) {
	numBatches := determineHowManyBatches(batchInfo)

	for i := 0; i < numBatches; i++ {
		archive := batchInfo.Archives[i]
		if archive.Size > MAX_DOWNLOAD_BATCH_SIZE_BYTES {
			splitBatches, err := s.splitAssetBatch(archive)
			if err != nil {
				log.Printf("Failed to split asset batch: %v", err)
				return BatchDownloadInfoResponse{}, fmt.Errorf("failed to split asset batch")
			}
			// Replace the original archive with the split batches in the batchInfo
			batchInfo.Archives = append(batchInfo.Archives[:i], append(splitBatches, batchInfo.Archives[i+1:]...)...)
			i = i + len(splitBatches) - 1
		}

	}
	return batchInfo, nil
}

func (s *Store) splitAssetBatch(archive Archives) ([]Archives, error) {
	var result []Archives

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
				batches = append(batches[:i], batches[i+1:]...)
				result = append(result, Archives{
					AssetIds: batch.AssetIds,
					Size:     batch.Size,
				})
				i--
				continue
			}

			if len(batch.AssetIds) <= 1 {
				result = append(result, Archives{
					AssetIds: batch.AssetIds,
					Size:     batch.Size,
				})
				batch.Approved = true
				batches = append(batches[:i], batches[i+1:]...)
				i--
				continue
			}

			batchInfo, err := s.getInfoForBatchDownload(BatchDownloadInfoPayload{AssetIds: batch.AssetIds})
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
				result = append(result, Archives{
					AssetIds: batch.AssetIds,
					Size:     batch.Size,
				})
				batches = append(batches[:i], batches[i+1:]...)
				i--
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

	if len(result) == 0 {
		result = append(result, archive)
		log.Printf("Remaining batches: %v", len(result))
	}

	return result, nil
}
