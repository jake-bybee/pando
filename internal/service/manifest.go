package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const assetManifestPath = "settings/asset-manifest.json"

const DefaultManifestLimit = 100
const MaxManifestLimit = 1000

type AssetManifestEntry struct {
	AssetID  string `json:"assetId"`
	FileName string `json:"fileName"`
}

type AssetManifestPage struct {
	Items      []AssetManifestEntry `json:"items"`
	Total      int                  `json:"total"`
	NextCursor *string              `json:"nextCursor"`
}

func GetAssetManifestPage(backupFolderPath string, limit int, after string) (AssetManifestPage, error) {
	if limit < 1 || limit > MaxManifestLimit {
		return AssetManifestPage{}, fmt.Errorf("limit must be between 1 and %d", MaxManifestLimit)
	}
	manifest, err := loadAssetManifest(backupFolderPath)
	if err != nil {
		return AssetManifestPage{}, err
	}

	assetIDs := make([]string, 0, len(manifest))
	for assetID := range manifest {
		if assetID > after {
			assetIDs = append(assetIDs, assetID)
		}
	}
	sort.Strings(assetIDs)

	page := AssetManifestPage{
		Items: make([]AssetManifestEntry, 0, min(limit, len(assetIDs))),
		Total: len(manifest),
	}
	for _, assetID := range assetIDs[:min(limit, len(assetIDs))] {
		page.Items = append(page.Items, AssetManifestEntry{AssetID: assetID, FileName: manifest[assetID]})
	}
	if len(assetIDs) > limit {
		cursor := page.Items[len(page.Items)-1].AssetID
		page.NextCursor = &cursor
	}
	return page, nil
}

func loadAssetManifest(backupFolderPath string) (map[string]string, error) {
	path := filepath.Join(backupFolderPath, assetManifestPath)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read asset manifest: %w", err)
	}

	var manifest map[string]string
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode asset manifest: %w", err)
	}
	if manifest == nil {
		manifest = make(map[string]string)
	}
	return manifest, nil
}

func saveAssetManifest(backupFolderPath string, manifest map[string]string) error {
	path := filepath.Join(backupFolderPath, assetManifestPath)
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create asset manifest directory: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode asset manifest: %w", err)
	}

	tempFile, err := os.CreateTemp(directory, ".asset-manifest-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary asset manifest: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("write temporary asset manifest: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("sync temporary asset manifest: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary asset manifest: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace asset manifest: %w", err)
	}
	return nil
}
