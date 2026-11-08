package immich

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"pando/internal/utils"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var MAX_DOWNLOAD_BATCH_SIZE_BYTES = int64((1024 * 1024 * 1024) / 5) // 500 MB
const MAX_CONCURRENT_DOWNLOADS = 5

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
	AssetId          string `json:"id"`
	OriginalFileName string `json:"originalFileName"`
	ExifInfo         struct {
		FileSizeInByte int64 `json:"fileSizeInByte"`
	} `json:"exifInfo"`
}
type OrderBy struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type IDFilter struct {
	Eq string `json:"eq"`
}

type Filter struct {
	ID *IDFilter `json:"id,omitempty"`
	Or []Filter  `json:"or,omitempty"`
}

type Payload struct {
	Size             int      `json:"size"`
	WithExif         bool     `json:"withExif"`
	OrderBy          *OrderBy `json:"orderBy,omitempty"`
	Filter           *Filter  `json:"filter,omitempty"`
	Cursor           string   `json:"cursor,omitempty"`
	OriginalFileName string   `json:"originalFileName,omitempty"`
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
	fileNames map[string]struct{}
}

func (s *Store) BulkDownload(assetIds []string) (bool, error) {
	_, err := s.BulkDownloadWithManifest(assetIds)
	return err == nil, err
}

func (s *Store) BulkDownloadWithManifest(assetIds []string) (map[string]string, error) {
	if len(assetIds) == 0 {
		log.Printf("No asset IDs provided for bulk download")
		return nil, fmt.Errorf("no asset IDs provided for bulk download")
	}
	batchInfo, err := s.getInfoForBatchDownload(BatchDownloadInfoPayload{AssetIds: assetIds})
	if err != nil {
		log.Printf("Failed to get batch download info: %v", err)
		return nil, fmt.Errorf("failed to get batch download info: %w", err)
	}

	chunkedAssetIds, assetFileNames, err := s.getAllDownloadBatches(batchInfo)
	if err != nil {
		log.Printf("Failed to get all download batches: %v", err)
		return nil, fmt.Errorf("failed to get all download batches: %w", err)
	}
	for _, assetId := range assetIds {
		if _, ok := assetFileNames[assetId]; !ok {
			return nil, fmt.Errorf("no filename metadata found for asset %s", assetId)
		}
	}

	err = s.bulkDownloadRoutineSwarm(chunkedAssetIds, s.config.BackupFolderPath, assetFileNames)
	if err != nil {
		return nil, fmt.Errorf("failed to download batches: %w", err)
	}

	return assetFileNames, nil

}

const sizesBatchSize = 100

func (s *Store) GetFilesSizesData(assetIds []string) ([]FileSizeDataResponse, error) {
	fullUrl := s.config.ImmichUrl + "/api/search/metadata"
	allItems := make([]FileSizeDataResponse, 0, len(assetIds))

	for start := 0; start < len(assetIds); start += sizesBatchSize {
		end := min(start+sizesBatchSize, len(assetIds))
		batch := assetIds[start:end]

		or := make([]Filter, 0, len(batch))
		for _, id := range batch {
			or = append(or, Filter{ID: &IDFilter{Eq: id}})
		}

		cursor := ""
		for {
			payload := Payload{
				Size:     len(batch),
				WithExif: true,
				Filter:   &Filter{Or: or},
				Cursor:   cursor,
			}

			resp, err := s.ImmichFetcher(fullUrl, "POST", payload)
			if err != nil {
				return nil, fmt.Errorf("failed to execute request for %s: %w", fullUrl, err)
			}

			var parsed SearchMetadataResponse
			err = json.NewDecoder(resp.Body).Decode(&parsed)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to decode response from %s: %w", fullUrl, err)
			}

			allItems = append(allItems, parsed.Assets.Items...)

			if parsed.Assets.NextCursor == nil || *parsed.Assets.NextCursor == "" {
				break
			}
			cursor = *parsed.Assets.NextCursor
		}
	}

	if len(allItems) != len(assetIds) {
		log.Printf("Warning: requested %d assets, got sizes for %d", len(assetIds), len(allItems))
	}
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

	for _, asset := range assets {
		size := asset.ExifInfo.FileSizeInByte
		added := false

		// An oversized asset can never share a chunk, so skip the scan entirely.
		if size <= MAX_DOWNLOAD_BATCH_SIZE_BYTES {
			for i := range chunks {
				if chunks[i].TotalSize+size > MAX_DOWNLOAD_BATCH_SIZE_BYTES {
					continue
				}
				if _, exists := chunks[i].fileNames[asset.OriginalFileName]; exists {
					continue
				}
				chunks[i].AssetIds = append(chunks[i].AssetIds, asset.AssetId)
				chunks[i].TotalSize += size
				chunks[i].fileNames[asset.OriginalFileName] = struct{}{}
				added = true
				break
			}
		}

		if !added {
			chunks = append(chunks, Chunk{
				AssetIds:  []string{asset.AssetId},
				TotalSize: size,
				fileNames: map[string]struct{}{asset.OriginalFileName: {}},
			})
		}
	}

	return chunks
}

func (s *Store) bulkDownloadRoutineSwarm(chunks []Chunk, dest string, assetFileNames map[string]string) error {
	var wg sync.WaitGroup

	type errorResult struct {
		AssetIds []string
		Err      error
	}

	downloadsChannel := make(chan struct{}, MAX_CONCURRENT_DOWNLOADS)
	errorsChannel := make(chan errorResult, len(chunks))

	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunk Chunk) {
			defer wg.Done()
			downloadsChannel <- struct{}{}
			if s.status != nil {
				s.status.ChunkStarted()
				defer s.status.ChunkFinished()
			}
			defer func() { <-downloadsChannel }()
			chunkAssetFileNames := make(map[string]string, len(chunk.AssetIds))
			for _, assetID := range chunk.AssetIds {
				fileName, ok := assetFileNames[assetID]
				if !ok {
					errorsChannel <- errorResult{
						AssetIds: chunk.AssetIds,
						Err:      fmt.Errorf("no filename metadata found for asset %s", assetID),
					}
					return
				}
				chunkAssetFileNames[assetID] = fileName
			}
			numFilesDownloaded, err := s.bulkDownload(chunk, dest, chunkAssetFileNames)
			if err != nil {
				log.Printf("Failed to bulk download assets %v", err)
				errorsChannel <- errorResult{AssetIds: chunk.AssetIds, Err: err}
				return
			}
			log.Printf("Successfully downloaded %d files to %s", numFilesDownloaded, dest)

		}(chunk)
	}
	wg.Wait()
	close(errorsChannel)
	var downloadErrors []error
	for errResult := range errorsChannel {
		log.Printf("Error downloading assets %v: %v", errResult.AssetIds, errResult.Err)
		downloadErrors = append(downloadErrors, fmt.Errorf("assets %v: %w", errResult.AssetIds, errResult.Err))
	}
	return errors.Join(downloadErrors...)
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

func (s *Store) bulkDownload(chunk Chunk, dest string, assetFileNames map[string]string) (int, error) {
	endpoint := "/api/download/archive"
	fullUrl := s.config.ImmichUrl + endpoint
	if dest == "" {
		dest = s.config.BackupFolderPath
	}

	log.Printf("Preparing to bulk download %d assets (%s) from %s", len(chunk.AssetIds), formatBytes(chunk.TotalSize), fullUrl)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(idleTimeout, cancel) // stop button
	defer timer.Stop()

	resp, err := s.ImmichDownloadFetcher(ctx, fullUrl, "POST",
		BatchDownloadInfoPayload{AssetIds: chunk.AssetIds})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	fileName := fmt.Sprintf("%x.zip", sha256.Sum256([]byte(strings.Join(chunk.AssetIds, ","))))
	filePath := filepath.Join(dest, fileName)

	f, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	_, err = io.Copy(f, &idleReader{r: resp.Body, t: timer})
	f.Close()
	if err != nil {
		os.Remove(filePath)
		return 0, err
	}

	tempDir := filepath.Join(dest, strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	if err := os.Mkdir(tempDir, 0o755); err != nil {
		os.Remove(filePath)
		return 0, fmt.Errorf("create extraction directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	err = utils.Unzip(filePath, tempDir, assetFileNames)
	if err != nil {
		log.Printf("Failed to unzip bulk download file %s: %v", filePath, err)
		os.Remove(filePath) // Clean up the zip file if unzipping fails
		return 0, err
	}

	for assetID, originalFileName := range assetFileNames {
		fileName := assetStorageFileName(assetID, originalFileName)
		if err := os.Rename(filepath.Join(tempDir, fileName), filepath.Join(dest, fileName)); err != nil {
			os.Remove(filePath)
			return 0, fmt.Errorf("move downloaded asset %s: %w", assetID, err)
		}
	}
	if err := os.Remove(filePath); err != nil {
		return 0, fmt.Errorf("remove downloaded archive: %w", err)
	}
	log.Printf("Successfully downloaded and stored %d assets from %s", len(assetFileNames), fileName)

	return len(chunk.AssetIds), nil
}

func assetStorageFileName(assetID, originalFileName string) string {
	return assetID + filepath.Ext(filepath.Base(originalFileName))
}

func (s *Store) getAllDownloadBatches(batchInfo BatchDownloadInfoResponse) ([]Chunk, map[string]string, error) {
	var allAssetIds []string
	for _, archive := range batchInfo.Archives {
		allAssetIds = append(allAssetIds, archive.AssetIds...)
	}

	sizesData, err := s.GetFilesSizesData(allAssetIds)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get files sizes data: %w", err)
	}

	sizeByAssetId := make(map[string]FileSizeDataResponse, len(sizesData))
	assetFileNames := make(map[string]string, len(sizesData))
	for _, s := range sizesData {
		sizeByAssetId[s.AssetId] = s
		if s.OriginalFileName == "" {
			return nil, nil, fmt.Errorf("no filename metadata found for asset %s", s.AssetId)
		}
		assetFileNames[s.AssetId] = s.OriginalFileName
	}

	var allChunks []Chunk
	for _, archive := range batchInfo.Archives {
		archiveAssets := make([]FileSizeDataResponse, 0, len(archive.AssetIds))
		for _, id := range archive.AssetIds {
			if data, ok := sizeByAssetId[id]; ok {
				archiveAssets = append(archiveAssets, data)
			} else {
				return nil, nil, fmt.Errorf("no size metadata found for asset %s", id)
			}
		}

		archiveChunks := ChunkAssetsBySize(archiveAssets)
		allChunks = append(allChunks, archiveChunks...)
	}

	return allChunks, assetFileNames, nil
}

const idleTimeout = 2 * time.Minute

type idleReader struct {
	r io.Reader
	t *time.Timer
}

func (i *idleReader) Read(p []byte) (int, error) {
	n, err := i.r.Read(p)
	if n > 0 {
		i.t.Reset(idleTimeout) // data arrived, restart the clock
	}
	return n, err
}
