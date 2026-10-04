package immich

import (
	"encoding/json"
	"fmt"
	"io"
)

const NUM_ASSETS_PER_PAGE = 500

func (s *Store) GetChunkAssetIds(cursor string) ([]string, string, error) {
	endpoint := "/api/search/metadata"
	fullUrl := s.config.ImmichUrl + endpoint

	var allIds []string

	payload := Payload{
		Size:     NUM_ASSETS_PER_PAGE,
		WithExif: false,

		OrderBy: struct {
			Field     string `json:"field"`
			Direction string `json:"direction"`
		}{
			Field:     "fileCreatedAt",
			Direction: "asc",
		},
		Cursor: cursor,
	}

	fmt.Printf("Requesting chunk of asset IDs with payload=%v", payload)

	resp, err := s.ImmichFetcher(fullUrl, "POST", payload)
	if err != nil {
		return nil, "", err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, "", err
	}

	var result SearchMetadataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", err
	}
	for _, item := range result.Assets.Items {
		allIds = append(allIds, item.AssetId)
	}

	return allIds, *result.Assets.NextCursor, nil
}
