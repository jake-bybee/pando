package system

import (
	"fmt"
	"os"
	"pando/internal/config"
)

// test
type Store struct {
	config *config.Config
}

func NewStore(config *config.Config) *Store {
	return &Store{
		config: config,
	}
}

func (s *Store) SystemHealthCheck() error {

	if s.config.BackupFolderPath == "" {
		return fmt.Errorf("backup folder path is not configured")
	}
	if _, err := os.Stat(s.config.BackupFolderPath); os.IsNotExist(err) {
		return fmt.Errorf("backup folder path does not exist")
	}
	return nil
}
