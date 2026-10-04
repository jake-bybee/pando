package backup

import (
	"fmt"
	"pando/internal/immich"
)

func Backup(assetIds []string, immichStore *immich.Store) (bool, error) {
	wasSuccessful, err := immichStore.BulkDownload(assetIds)
	if err != nil {
		return false, fmt.Errorf("Failed to download asset batch %v", err)
	}

	return wasSuccessful, nil

}
