package utils

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestUnzipRenamesFilesByAssetID(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "assets.zip")
	createZip(t, archivePath, map[string]string{
		"photos/one.jpg": "one",
		"photos/two.png": "two",
	})
	destination := t.TempDir()

	err := Unzip(archivePath, destination, map[string]string{
		"asset-one": "one.jpg",
		"asset-two": "two.png",
	})
	if err != nil {
		t.Fatalf("Unzip() error = %v", err)
	}

	for _, test := range []struct {
		name string
		want string
	}{
		{"asset-one.jpg", "one"},
		{"asset-two.png", "two"},
	} {
		data, err := os.ReadFile(filepath.Join(destination, test.name))
		if err != nil {
			t.Fatalf("read %s: %v", test.name, err)
		}
		if string(data) != test.want {
			t.Errorf("%s = %q, want %q", test.name, data, test.want)
		}
	}
}

func TestUnzipRejectsUnsafePaths(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "assets.zip")
	createZip(t, archivePath, map[string]string{"../../outside.jpg": "unsafe"})

	err := Unzip(archivePath, t.TempDir(), map[string]string{"asset": "outside.jpg"})
	if err == nil {
		t.Fatal("Unzip() error = nil, want unsafe path error")
	}
}

func createZip(t *testing.T, path string, files map[string]string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for name, contents := range files {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
