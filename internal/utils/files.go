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

func Unzip(src, dest string, assetFileNames map[string]string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	usedAssetIDs := make(map[string]bool, len(assetFileNames))
	extracted := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		relativePath, err := safeZipPath(f.Name)
		if err != nil {
			return err
		}

		assetID := findAssetID(filepath.Base(relativePath), assetFileNames, usedAssetIDs)
		if assetID == "" {
			return fmt.Errorf("archive entry %q does not match an asset filename", f.Name)
		}
		usedAssetIDs[assetID] = true
		path := filepath.Join(dest, assetID+filepath.Ext(relativePath))

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return err
		}
		if err := srcFile.Close(); err != nil {
			dstFile.Close()
			return err
		}
		if err := dstFile.Close(); err != nil {
			return err
		}
		extracted++
	}

	if extracted != len(assetFileNames) {
		return fmt.Errorf("archive contained %d files, expected %d", extracted, len(assetFileNames))
	}
	return nil
}

func safeZipPath(name string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if cleaned == "." || filepath.IsAbs(cleaned) || cleaned == ".." ||
		strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive entry path %q", name)
	}
	return cleaned, nil
}

func findAssetID(fileName string, assetFileNames map[string]string, used map[string]bool) string {
	for assetID, originalFileName := range assetFileNames {
		if !used[assetID] && filepath.Base(originalFileName) == fileName {
			return assetID
		}
	}
	return ""
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
