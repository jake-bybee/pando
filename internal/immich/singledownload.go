package immich

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var missingAssetMetadata = struct {
	sync.Mutex
	ids map[*Store]map[string]struct{}
}{ids: make(map[*Store]map[string]struct{})}

func logMissingAssetMetadata(s *Store, assetID string) {
	missingAssetMetadata.Lock()
	defer missingAssetMetadata.Unlock()
	ids := missingAssetMetadata.ids[s]
	if ids == nil {
		ids = make(map[string]struct{})
		missingAssetMetadata.ids[s] = ids
	}
	if _, ok := ids[assetID]; ok {
		return
	}
	ids[assetID] = struct{}{}
	log.Printf("No filename metadata returned for missing asset %s", assetID)
}

func (s *Store) singleDownload(assetID, originalFileName, dest string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(idleTimeout, cancel)
	defer timer.Stop()

	url := fmt.Sprintf("%s/api/assets/%s/original", s.config.ImmichUrl, assetID)
	resp, err := s.ImmichDownloadFetcher(ctx, url, "GET", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp(dest, ".pando-single-download-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	n, err := io.Copy(tempFile, &idleReader{r: resp.Body, t: timer})
	if closeErr := tempFile.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("downloaded zero bytes")
	}

	fileName := assetStorageFileName(assetID, originalFileName)
	if err := os.Rename(tempPath, filepath.Join(dest, fileName)); err != nil {
		return err
	}
	return nil
}
