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

var MAX_DOWNLOAD_BATCH_SIZE_BYTES = int64((1024 * 1024 * 1024) / 5) // 500 MB
const MAX_CONCURRENT_DOWNLOADS = 5

type Store struct {
	client *http.Client
	config *config.Config
}

type BatchDownloadInfoResponse struct {
	TotalSize int64      `json:"totalSize"`
	Archives  []Archives `json:"archives"`
}

type Archives struct {
	Size     int64    `json:"size"`
	AssetIds []string `json:"assetIds"`
}

type BatchDownloadInfoPayload struct {
	AssetIds []string `json:"assetIds"`
}

type SearchMetadataResponse struct {
	Assets struct {
		Items      []FileSizeDataResponse `json:"items"`
		NextCursor *string                `json:"nextCursor"`
	} `json:"assets"`
}

type FileSizeDataResponse struct {
	AssetId  string `json:"id"`
	ExifInfo struct {
		FileSizeInByte int64 `json:"fileSizeInByte"`
	} `json:"exifInfo"`
}
type Payload struct {
	Filter   Filter `json:"filter"`
	Size     int    `json:"size"`
	WithExif bool   `json:"withExif"`
	Cursor   string `json:"cursor,omitempty"`
}

type Filter struct {
	Or []OrCondition `json:"or"`
}

type OrCondition struct {
	ID IDMatch `json:"id"`
}

type IDMatch struct {
	Eq string `json:"eq"`
}

type Chunk struct {
	AssetIds  []string
	TotalSize int64
}

func NewStore(config *config.Config, client *http.Client) *Store {
	return &Store{
		client: client,
		config: config,
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

	chunkedAssetIds, err := s.getAllDownloadBatches(batchInfo)
	if err != nil {
		log.Printf("Failed to get all download batches: %v", err)
		return false, fmt.Errorf("failed to get all download batches: %v", err)
	}

	err = s.bulkDownloadRoutineSwarm(chunkedAssetIds, s.config.BackupFolderPath)
	if err != nil {
		return false, fmt.Errorf("failed to download batches: %v", err)
	}

	return true, nil

}

func NewPayload(assetIds []string, size int, withExif bool) Payload {
	orConditions := make([]OrCondition, len(assetIds))
	for i, id := range assetIds {
		orConditions[i] = OrCondition{ID: IDMatch{Eq: id}}
	}

	return Payload{
		Filter:   Filter{Or: orConditions},
		Size:     size,
		WithExif: withExif,
	}
}

func (s *Store) GetFilesSizesData(assetIds []string) ([]FileSizeDataResponse, error) {
	endpoint := "/api/search/metadata"
	fullUrl := s.config.ImmichUrl + endpoint

	var allItems []FileSizeDataResponse
	cursor := ""

	for {
		payload := NewPayload(assetIds, len(assetIds), true)
		if cursor != "" {
			payload.Cursor = cursor
		}

		log.Printf("Requesting file sizes data for %v files from %s (cursor=%q)", len(assetIds), fullUrl, cursor)

		resp, err := s.ImmichFetcher(fullUrl, "POST", payload)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request for %s: %v", fullUrl, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body for %s: %v", fullUrl, err)
		}

		var searchMetadataResponse SearchMetadataResponse
		if err := json.Unmarshal(body, &searchMetadataResponse); err != nil {
			log.Printf("Failed to unmarshal file sizes data for %v files from %s: %v", len(assetIds), fullUrl, err)
			return nil, fmt.Errorf("failed to unmarshal response body for %s: %v", fullUrl, err)
		}

		allItems = append(allItems, searchMetadataResponse.Assets.Items...)

		if searchMetadataResponse.Assets.NextCursor == nil || *searchMetadataResponse.Assets.NextCursor == "" {
			break
		}
		cursor = *searchMetadataResponse.Assets.NextCursor
	}

	log.Printf("Successfully retrieved file sizes data for %v files from %s (%d total items)", len(assetIds), fullUrl, len(allItems))

	return allItems, nil
}

func formatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func ChunkAssetsBySize(assets []FileSizeDataResponse) []Chunk {
	var chunks []Chunk
	var currentChunk []string
	var currentSize int64

	for _, asset := range assets {
		size := asset.ExifInfo.FileSizeInByte

		if size > MAX_DOWNLOAD_BATCH_SIZE_BYTES {
			if len(currentChunk) > 0 {
				chunks = append(chunks, Chunk{AssetIds: currentChunk, TotalSize: currentSize})
				currentChunk = nil
				currentSize = 0
			}
			chunks = append(chunks, Chunk{AssetIds: []string{asset.AssetId}, TotalSize: size})
			continue
		}

		if currentSize+size > MAX_DOWNLOAD_BATCH_SIZE_BYTES {
			chunks = append(chunks, Chunk{AssetIds: currentChunk, TotalSize: currentSize})
			currentChunk = nil
			currentSize = 0
		}

		currentChunk = append(currentChunk, asset.AssetId)
		currentSize += size
	}

	if len(currentChunk) > 0 {
		chunks = append(chunks, Chunk{AssetIds: currentChunk, TotalSize: currentSize})
	}

	return chunks
}

func (s *Store) bulkDownloadRoutineSwarm(chunks []Chunk, dest string) error {
	var wg sync.WaitGroup

	type errorResult struct {
		AssetIds []string
		Err      error
	}

	downloadsChannel := make(chan struct{}, MAX_CONCURRENT_DOWNLOADS)
	errorsChannel := make(chan errorResult)

	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunk Chunk) {
			defer wg.Done()
			downloadsChannel <- struct{}{}
			defer func() { <-downloadsChannel }()
			numFilesDownloaded, err := s.bulkDownload(chunk, dest)
			if err != nil {
				log.Printf("Failed to bulk download assets %v: %v", chunk.AssetIds, err)
				errorsChannel <- errorResult{AssetIds: chunk.AssetIds, Err: err}
			}
			log.Printf("Successfully downloaded %d files to %s", numFilesDownloaded, dest)

		}(chunk)
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

	resp, err := s.ImmichFetcher(fullUrl, "POST", payload)
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

func (s *Store) bulkDownload(chunk Chunk, dest string) (int, error) {
	endpoint := "/api/download/archive"
	fullUrl := s.config.ImmichUrl + endpoint

	log.Printf("Preparing to bulk download %d assets (%s) from %s", len(chunk.AssetIds), formatBytes(chunk.TotalSize), fullUrl)
	resp, err := s.ImmichFetcher(fullUrl, "POST", BatchDownloadInfoPayload{AssetIds: chunk.AssetIds})
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

	fileName := fmt.Sprintf("%x.zip", sha256.Sum256([]byte(strings.Join(chunk.AssetIds, ","))))
	filePath := s.config.BackupFolderPath + fileName

	err = os.WriteFile(filePath, body, 0644)
	if err != nil {
		log.Printf("Failed to write bulk download file %s: %v", filePath, err)
		return 0, err
	}

	if dest == "" {
		dest = s.config.BackupFolderPath
	}

	err = utils.Unzip(filePath, dest)
	if err != nil {
		log.Printf("Failed to unzip bulk download file %s: %v", filePath, err)
		return 0, err
	}
	log.Printf("Successfully wrote bulk download file %s", filePath)

	os.Remove(filePath) // Clean up the zip file after extraction

	return len(chunk.AssetIds), nil
}

func (s *Store) getAllDownloadBatches(batchInfo BatchDownloadInfoResponse) ([]Chunk, error) {
	var allAssetIds []string
	for _, archive := range batchInfo.Archives {
		allAssetIds = append(allAssetIds, archive.AssetIds...)
	}

	sizesData, err := s.GetFilesSizesData(allAssetIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get files sizes data: %v", err)
	}

	sizeByAssetId := make(map[string]FileSizeDataResponse, len(sizesData))
	for _, s := range sizesData {
		sizeByAssetId[s.AssetId] = s
	}

	var allChunks []Chunk
	for _, archive := range batchInfo.Archives {
		archiveAssets := make([]FileSizeDataResponse, 0, len(archive.AssetIds))
		for _, id := range archive.AssetIds {
			if data, ok := sizeByAssetId[id]; ok {
				archiveAssets = append(archiveAssets, data)
			} else {
				log.Printf("Warning: no size data found for asset %s, skipping", id)
			}
		}

		archiveChunks := ChunkAssetsBySize(archiveAssets)
		allChunks = append(allChunks, archiveChunks...)
	}

	return allChunks, nil
}
