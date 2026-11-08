package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"pando/internal/config"
	"pando/internal/immich"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAssetManifestRoundTrip(t *testing.T) {
	backupFolder := t.TempDir()
	want := map[string]string{
		"asset-id-1": "photo-one.jpg",
		"asset-id-2": "photo-two.jpg",
	}

	if err := saveAssetManifest(backupFolder, want); err != nil {
		t.Fatalf("save asset manifest: %v", err)
	}

	got, err := loadAssetManifest(backupFolder)
	if err != nil {
		t.Fatalf("load asset manifest: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded manifest = %#v, want %#v", got, want)
	}
}

func TestLoadMissingAssetManifest(t *testing.T) {
	manifest, err := loadAssetManifest(t.TempDir())
	if err != nil {
		t.Fatalf("load missing manifest: %v", err)
	}
	if manifest == nil || len(manifest) != 0 {
		t.Fatalf("manifest = %#v, want an empty map", manifest)
	}
}

func TestGetAssetManifestPage(t *testing.T) {
	backupFolder := t.TempDir()
	if err := saveAssetManifest(backupFolder, map[string]string{
		"asset-c": "three.jpg",
		"asset-a": "one.jpg",
		"asset-b": "two.jpg",
	}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		after      string
		limit      int
		want       []AssetManifestEntry
		nextCursor string
	}{
		{"first page", "", 2, []AssetManifestEntry{{"asset-a", "one.jpg"}, {"asset-b", "two.jpg"}}, "asset-b"},
		{"last page", "asset-b", 2, []AssetManifestEntry{{"asset-c", "three.jpg"}}, ""},
		{"exact page", "", 3, []AssetManifestEntry{{"asset-a", "one.jpg"}, {"asset-b", "two.jpg"}, {"asset-c", "three.jpg"}}, ""},
		{"absent cursor", "asset-ab", 1, []AssetManifestEntry{{"asset-b", "two.jpg"}}, "asset-b"},
		{"past end", "asset-z", 2, []AssetManifestEntry{}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			page, err := GetAssetManifestPage(backupFolder, test.limit, test.after)
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 3 || !reflect.DeepEqual(page.Items, test.want) {
				t.Fatalf("page = %+v, want total 3 and items %+v", page, test.want)
			}
			if test.nextCursor == "" {
				if page.NextCursor != nil {
					t.Fatalf("next cursor = %q, want nil", *page.NextCursor)
				}
			} else if page.NextCursor == nil || *page.NextCursor != test.nextCursor {
				t.Fatalf("next cursor = %v, want %q", page.NextCursor, test.nextCursor)
			}
		})
	}
}

func TestGetMissingAssetManifestPage(t *testing.T) {
	page, err := GetAssetManifestPage(t.TempDir(), DefaultManifestLimit, "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil || len(page.Items) != 0 || page.Total != 0 || page.NextCursor != nil {
		t.Fatalf("page = %+v, want an empty page", page)
	}
}

func TestGetAssetManifestPageInvalidLimit(t *testing.T) {
	for _, limit := range []int{-1, 0, MaxManifestLimit + 1} {
		if _, err := GetAssetManifestPage(t.TempDir(), limit, ""); err == nil {
			t.Fatalf("limit %d: expected an error", limit)
		}
	}
}

func TestInitializeAssetManifestReconcilesFilesAndMetadata(t *testing.T) {
	backupFolder := t.TempDir()
	for _, fileName := range []string{"550e8400-e29b-41d4-a716-446655440000.jpg", "550e8400-e29b-41d4-a716-446655440001.png"} {
		if err := os.WriteFile(filepath.Join(backupFolder, fileName), []byte("asset"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := saveAssetManifest(backupFolder, map[string]string{
		"550e8400-e29b-41d4-a716-446655440001": "old-two.png",
		"550e8400-e29b-41d4-a716-446655440002": "missing.jpg",
	}); err != nil {
		t.Fatal(err)
	}

	immichServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"assets":{"items":[{"id":"550e8400-e29b-41d4-a716-446655440000","originalFileName":"one.jpg","exifInfo":{"fileSizeInByte":1}}]}}`))
	}))
	defer immichServer.Close()

	cfg := &config.Config{BackupFolderPath: backupFolder, ImmichUrl: immichServer.URL}
	store := NewStore(cfg, immich.NewStore(cfg, immichServer.Client(), immichServer.Client()))
	got, err := store.initializeAssetManifest()
	if err != nil {
		t.Fatalf("initializeAssetManifest() error = %v", err)
	}

	want := map[string]string{
		"550e8400-e29b-41d4-a716-446655440000": "one.jpg",
		"550e8400-e29b-41d4-a716-446655440001": "old-two.png",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest = %#v, want %#v", got, want)
	}
}

func TestInitializeAssetManifestSkipsNonAssetFiles(t *testing.T) {
	backupFolder := t.TempDir()
	validID := "550e8400-e29b-41d4-a716-446655440000"
	for _, fileName := range []string{validID + ".jpg", "IMG_0001.jpg", strings.Repeat("a", 64) + ".zip", ".DS_Store"} {
		if err := os.WriteFile(filepath.Join(backupFolder, fileName), []byte("asset"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	immichServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), validID) || strings.Contains(string(body), "IMG_0001") || strings.Contains(string(body), strings.Repeat("a", 64)) {
			t.Fatalf("metadata request body = %s", body)
		}
		_, _ = w.Write([]byte(`{"assets":{"items":[{"id":"550e8400-e29b-41d4-a716-446655440000","originalFileName":"photo.jpg"}]}}`))
	}))
	defer immichServer.Close()

	cfg := &config.Config{BackupFolderPath: backupFolder, ImmichUrl: immichServer.URL}
	store := NewStore(cfg, immich.NewStore(cfg, immichServer.Client(), immichServer.Client()))
	if _, err := store.initializeAssetManifest(); err != nil {
		t.Fatalf("initializeAssetManifest() error = %v", err)
	}
}
