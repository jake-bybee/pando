package utils

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const CONFIG_FILE_NAME = "config.txt"
const SETTINGS_FOLDER_NAME = "settings"

type Store struct {
	timezone string
}

func NewStore(timezone string) *Store {
	return &Store{timezone: timezone}
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := uniquePath(filepath.Join(dest, f.Name))

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, os.ModePerm)
			continue
		}

		os.MkdirAll(filepath.Dir(path), os.ModePerm)

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()
	}
	return nil
}

func uniquePath(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)

	for i := 1; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = fmt.Sprintf("%s(%d)%s", base, i, ext)
	}
}

func (s *Store) TimeNow() string {
	if s.timezone == "" {
		fmt.Println("Timezone is not set, defaulting to UTC")
	}
	loc, _ := time.LoadLocation(s.timezone)
	return time.Now().In(loc).Format(time.RFC3339)
}

func LoadId(backupFolderPath string) (string, error) {
	filePath := filepath.Join(backupFolderPath, SETTINGS_FOLDER_NAME, CONFIG_FILE_NAME)

	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var cfg struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return "", err
	}
	if cfg.ID == "" {
		return "", errors.New("id missing in config file")
	}
	return cfg.ID, nil
}

func WriteId(backupFolderPath, id string) error {
	filePath := filepath.Join(backupFolderPath, SETTINGS_FOLDER_NAME, CONFIG_FILE_NAME)

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	cfg := map[string]any{}

	b, err := os.ReadFile(filePath)
	switch {
	case err == nil:
		if len(bytes.TrimSpace(b)) > 0 {
			if err := json.Unmarshal(b, &cfg); err != nil {
				return fmt.Errorf("existing config is not valid JSON: %w", err)
			}
		}
	case errors.Is(err, os.ErrNotExist):
		// no file yet, start with an empty config
	default:
		return err
	}

	cfg["id"] = id

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, out, 0o644)
}
