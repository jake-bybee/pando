package immich

import "testing"

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
