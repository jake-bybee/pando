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
