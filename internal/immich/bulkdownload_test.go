package immich

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	appconfig "pando/internal/config"
)

func TestChunkAssetsBySizeSeparatesDuplicateFileNames(t *testing.T) {
	assets := []FileSizeDataResponse{
		{AssetId: "file-1", OriginalFileName: "file.jpg"},
		{AssetId: "pic-1", OriginalFileName: "pic.jpg"},
		{AssetId: "file-2", OriginalFileName: "file.jpg"},
		{AssetId: "pic-2", OriginalFileName: "pic.jpg"},
	}
	for i := range assets {
		assets[i].ExifInfo.FileSizeInByte = 1
	}

	chunks := ChunkAssetsBySize(assets)
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2: %+v", len(chunks), chunks)
	}
	if len(chunks[0].AssetIds) != 2 || len(chunks[1].AssetIds) != 2 {
		t.Fatalf("chunks = %+v, want two assets per chunk", chunks)
	}
	if chunks[0].AssetIds[0] != "file-1" || chunks[0].AssetIds[1] != "pic-1" ||
		chunks[1].AssetIds[0] != "file-2" || chunks[1].AssetIds[1] != "pic-2" {
		t.Fatalf("chunks = %+v, want duplicate names split across chunks", chunks)
	}
}

func TestBulkDownloadWithManifestDownloadsMissingAssetsIndividually(t *testing.T) {
	cases := []struct {
		name       string
		singleBody string
		singleCode int
		wantAssets int
		wantSingle bool
	}{
		{name: "downloads missing asset", singleBody: "motion video", singleCode: http.StatusOK, wantAssets: 3, wantSingle: true},
		{name: "single download failure is non-fatal", singleCode: http.StatusInternalServerError, wantAssets: 2},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			backupFolder := t.TempDir()
			archiveData := testDownloadArchive(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/download/info":
					json.NewEncoder(w).Encode(BatchDownloadInfoResponse{
						Archives: []Archives{{AssetIds: []string{"asset-1", "asset-2"}}},
					})
				case "/api/search/metadata":
					var payload Payload
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Errorf("decode metadata payload: %v", err)
						return
					}
					items := []FileSizeDataResponse{}
					if payload.Size == 2 {
						items = []FileSizeDataResponse{
							{AssetId: "asset-1", OriginalFileName: "one.jpg", ExifInfo: struct {
								FileSizeInByte int64 `json:"fileSizeInByte"`
							}{FileSizeInByte: 1}},
							{AssetId: "asset-2", OriginalFileName: "two.jpg", ExifInfo: struct {
								FileSizeInByte int64 `json:"fileSizeInByte"`
							}{FileSizeInByte: 1}},
						}
					} else {
						items = []FileSizeDataResponse{{AssetId: "asset-3", OriginalFileName: "motion.mp4", ExifInfo: struct {
							FileSizeInByte int64 `json:"fileSizeInByte"`
						}{FileSizeInByte: 1}}}
					}
					json.NewEncoder(w).Encode(SearchMetadataResponse{Assets: struct {
						Items      []FileSizeDataResponse `json:"items"`
						NextCursor *string                `json:"nextCursor"`
					}{Items: items}})
				case "/api/download/archive":
					w.Write(archiveData)
				case "/api/assets/asset-3/original":
					w.WriteHeader(testCase.singleCode)
					io.WriteString(w, testCase.singleBody)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			cfg := &appconfig.Config{ImmichUrl: server.URL, BackupFolderPath: backupFolder}
			store := NewStore(cfg, server.Client(), server.Client())
			assetFileNames, err := store.BulkDownloadWithManifest([]string{"asset-1", "asset-2", "asset-3"})
			if err != nil {
				t.Fatalf("BulkDownloadWithManifest() error = %v", err)
			}
			if len(assetFileNames) != testCase.wantAssets {
				t.Fatalf("got %d asset names, want %d: %v", len(assetFileNames), testCase.wantAssets, assetFileNames)
			}
			if _, ok := assetFileNames["asset-1"]; !ok {
				t.Error("asset-1 missing from returned asset names")
			}
			if _, ok := assetFileNames["asset-2"]; !ok {
				t.Error("asset-2 missing from returned asset names")
			}
			_, err = os.Stat(filepath.Join(backupFolder, "asset-3.mp4"))
			if testCase.wantSingle && err != nil {
				t.Fatalf("single-download file missing: %v", err)
			}
			if !testCase.wantSingle && !os.IsNotExist(err) {
				t.Fatalf("single-download file exists after failure, stat error = %v", err)
			}
		})
	}
}

func testDownloadArchive(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, file := range []struct {
		name string
		body string
	}{{name: "one.jpg", body: "one"}, {name: "two.jpg", body: "two"}} {
		fileWriter, err := writer.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(fileWriter, file.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
