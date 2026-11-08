package config

import (
	"fmt"
	"log"
	"os"
	"pando/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ImmichUrl        string
	ImmichApiToken   string
	SelfUrl          string
	MasterPandoPort  string
	MasterPandoUrl   string
	BackupFolderPath string
	TimeZone         string
	NumPeers         int
	NumReplicas      int
	ListenPort       string
	MinFreeBytes     uint64
	StorageRecheckSeconds int
}

func LoadEnv() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("godotenv error:", err)
	}

	cfg, err := loadFromEnvironment()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadFromEnvironment() (*Config, error) {
	config := &Config{
		ImmichUrl:        os.Getenv("IMMICH_URL"),
		ImmichApiToken:   os.Getenv("IMMICH_API_TOKEN"),
		SelfUrl:          os.Getenv("SELF_URL"),
		MasterPandoPort:  os.Getenv("MASTER_PANDO_PORT"),
		MasterPandoUrl:   os.Getenv("MASTER_PANDO_URL"),
		BackupFolderPath: os.Getenv("BACKUP_FOLDER_PATH"),
		TimeZone:         os.Getenv("TIME_ZONE"),
		NumPeers:         ParseNumPeers(),
		NumReplicas:      ParseNumReplicas(),
		ListenPort:       os.Getenv("LISTEN_PORT"),
	}

	var err error
	config.MinFreeBytes = 5368709120
	if value := os.Getenv("MIN_FREE_BYTES"); value != "" {
		config.MinFreeBytes, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid MIN_FREE_BYTES: %v", err)
		}
	}
	config.StorageRecheckSeconds = 60
	if value := os.Getenv("STORAGE_RECHECK_SECONDS"); value != "" {
		config.StorageRecheckSeconds, err = strconv.Atoi(value)
		if err != nil || config.StorageRecheckSeconds < 0 {
			return nil, fmt.Errorf("invalid STORAGE_RECHECK_SECONDS: %s", value)
		}
	}

	if config.ImmichUrl == "" {
		return nil, fmt.Errorf("IMMICH_URL is not set")
	} else {
		config.ImmichUrl, err = utils.NormalizeURL(config.ImmichUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize IMMICH_URL: %v", err)
		}
	}
	if config.SelfUrl == "" {
		return nil, fmt.Errorf("SELF_URL is not set")
	} else {
		config.SelfUrl, err = utils.NormalizeURL(config.SelfUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize SELF_URL: %v", err)
		}
	}
	if config.ImmichApiToken == "" {
		return nil, fmt.Errorf("IMMICH_API_TOKEN is not set")
	}
	if config.ListenPort == "" {
		log.Println("LISTEN_PORT is not set, using default 8080")
		config.ListenPort = "8080" // default port if not set
	}
	if config.MasterPandoUrl == "" {
		masterPandoUrl, err := getMasterPandoUrlFromImmichUrl()
		if err != nil {
			return nil, err
		}
		config.MasterPandoUrl, err = utils.NormalizeURL(masterPandoUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize MASTER_PANDO_URL: %v", err)
		}

	}
	if config.BackupFolderPath == "" {
		return nil, fmt.Errorf("BACKUP_FOLDER_PATH is not set")
	}
	config.BackupFolderPath = strings.TrimRight(config.BackupFolderPath, "/") + "/"

	if config.TimeZone == "" || func() bool {
		_, err := time.LoadLocation(config.TimeZone)
		return err != nil
	}() {
		config.TimeZone = "UTC" // default time zone if not set
	}

	return config, nil
}

func getMasterPandoUrlFromImmichUrl() (string, error) {
	log.Println("Can't parse master pando URL from immich url, please provide an master pando URL in the environment variable MASTER_PANDO_URL")
	return "", fmt.Errorf("can't parse master pando URL from immich url")

}
