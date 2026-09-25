package immich

import (
	"encoding/json"
	"io"
)

func (s *Store) GetAllAssetIds() ([]string, error) {
	endpoint := "/api/search/metadata"
	fullUrl := s.config.ImmichUrl + endpoint

	var allIds []string
	cursor := ""

	for {
		payload := Payload{
			Size:     1000, // page size
			WithExif: false,
		}
		if cursor != "" {
			payload.Cursor = cursor
		}
		// omit Filter entirely — no ID constraint means "all assets"

		resp, err := s.ImmichFetcher(fullUrl, "POST", payload)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var result SearchMetadataResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		for _, item := range result.Assets.Items {
			allIds = append(allIds, item.AssetId)
		}

		if result.Assets.NextCursor == nil || *result.Assets.NextCursor == "" {
			break
		}
		cursor = *result.Assets.NextCursor
	}

	return allIds, nil
}
