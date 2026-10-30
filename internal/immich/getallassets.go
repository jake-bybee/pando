package immich

import (
	"encoding/json"
	"io"
)

const NUM_ASSETS_PER_PAGE = 500

func (s *Store) GetChunkAssetIds(cursor string) ([]string, string, bool, error) {
	endpoint := "/api/search/metadata"
	fullUrl := s.config.ImmichUrl + endpoint

	var allIds []string
	payload := Payload{
		Size:     NUM_ASSETS_PER_PAGE,
		WithExif: false,
		OrderBy:  &OrderBy{Field: "fileCreatedAt", Direction: "asc"},
		Cursor:   cursor,
	}

	resp, err := s.ImmichFetcher(fullUrl, "POST", payload)
	if err != nil {
		return nil, "", false, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, "", false, err
	}

	var result SearchMetadataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", false, err
	}
	for _, item := range result.Assets.Items {
		allIds = append(allIds, item.AssetId)
	}

	nextCursor := ""
	if result.Assets.NextCursor != nil {
		nextCursor = *result.Assets.NextCursor
	}

	return allIds, nextCursor, nextCursor == "", nil
}
