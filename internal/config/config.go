package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ImmichUrl        string
	ImmichApiToken   string
	MasterPandoPort  string
	MasterPandoUrl   string
	BackupFolderPath string
}

func LoadEnv() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("godotenv error:", err) // print the ACTUAL error
	}

	cfg, err := loadFromEnvironment()
	if err != nil {
		return Config{}, err
	}
	return *cfg, nil
}

func loadFromEnvironment() (*Config, error) {
	config := &Config{
		ImmichUrl:        os.Getenv("IMMICH_URL"),
		ImmichApiToken:   os.Getenv("IMMICH_API_TOKEN"),
		MasterPandoPort:  os.Getenv("MASTER_PANDO_PORT"),
		MasterPandoUrl:   os.Getenv("MASTER_PANDO_URL"),
		BackupFolderPath: os.Getenv("BACKUP_FOLDER_PATH"),
	}

	if config.ImmichUrl == "" {
		return nil, fmt.Errorf("IMMICH_URL is not set")
	}
	if config.ImmichApiToken == "" {
		return nil, fmt.Errorf("IMMICH_API_TOKEN is not set")
	}
	if config.MasterPandoPort == "" {
		log.Println("MASTER_PANDO_PORT is not set, using default 8080")
		config.MasterPandoPort = "8080" // default port if not set
	}
	if config.MasterPandoUrl == "" {
		var err error
		config.MasterPandoUrl, err = getMasterPandoUrlFromImmichUrl(config.ImmichUrl, config.MasterPandoPort)
		if err != nil {
			return nil, err
		}
	}
	if config.BackupFolderPath == "" {
		return nil, fmt.Errorf("BACKUP_FOLDER_PATH is not set")
	}
	config.BackupFolderPath = strings.TrimRight(config.BackupFolderPath, "/") + "/"

	return config, nil
}

func getMasterPandoUrlFromImmichUrl(immichUrl string, masterPandoPort string) (string, error) {
	// If immich uses a port, replace that port to the master pando port
	if strings.Contains(immichUrl, ":") {
		hostUrl := strings.Split(immichUrl, ":")[0]
		immichUrl = fmt.Sprintf("%s:%s", hostUrl, masterPandoPort)
		return immichUrl, nil
	}
	log.Println("Can't parse master pando URL from immich url, please provide an master pando URL in the environment variable MASTER_PANDO_URL")
	return "", fmt.Errorf("can't parse master pando URL from immich url")

}
