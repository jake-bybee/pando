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

func (s *Store) ResolveFileNameToAssetId(fileName string) (string, error) {
	endpoint := "/api/search/metadata"
	fullUrl := s.config.ImmichUrl + endpoint

	payload := Payload{
		Size:             1,
		OriginalFileName: fileName,
	}

	resp, err := s.ImmichFetcher(fullUrl, "POST", payload)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return "", err
	}

	var result SearchMetadataResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Assets.Items) > 0 {
		return result.Assets.Items[0].AssetId, nil
	}

	return "", nil
}
