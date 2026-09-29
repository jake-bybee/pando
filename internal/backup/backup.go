package backup

import (
	"fmt"
	"pando/internal/immich"
)

func Backup(assetIds []string) (bool, error) {
	wasSuccessful, err := immich.BulkDownload(assetIds)
	if err != nil {
		return false, fmt.Errorf("Failed to download asset batch %v", err)
	}

	return wasSuccessful, nil

}
