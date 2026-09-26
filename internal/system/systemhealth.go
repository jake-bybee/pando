package system

import (
	"fmt"
	"os"
	"pando/internal/config"
	"pando/internal/immich"

	"golang.org/x/sys/unix"
)

var configVariables *config.Config

func Init(config *config.Config) {

	configVariables = config

}

func SystemHealthCheck() error {

	if configVariables.BackupFolderPath == "" {
		return fmt.Errorf("backup folder path is not configured")
	}
	if _, err := os.Stat(configVariables.BackupFolderPath); os.IsNotExist(err) {
		return fmt.Errorf("backup folder path does not exist")
	}

	if !isSystemStorageSufficient() {
		return fmt.Errorf("system storage is insufficient")
	}
	return nil
}

func getSystemStorage() (uint64, error) {
	var stat unix.Statfs_t

	if err := unix.Statfs(configVariables.BackupFolderPath, &stat); err != nil {
		return 0, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)

	fmt.Print("Checking system storage for path: ", configVariables.BackupFolderPath)
	fmt.Print(" Total:", total)
	fmt.Printf(" Free: %d\n", free)

	return free, nil
}
func isSystemStorageSufficient() bool {
	user, err := immich.GetUsers()
	if err != nil {
		return false
	}

	free, err := getSystemStorage()
	if err != nil {
		return false
	}
	return free >= uint64(user.QuotaUsageInBytes)
}
