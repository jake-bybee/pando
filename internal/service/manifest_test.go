package service

import (
	"reflect"
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
