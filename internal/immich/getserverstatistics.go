package immich

import (
	"encoding/json"
	"io"
)

type ServerStatisticsResponse struct {
	Photos      int64       `json:"photos"`
	Videos      int64       `json:"videos"`
	Usage       int64       `json:"usage"`
	UsagePhotos int64       `json:"usagePhotos"`
	UsageVideos int64       `json:"usageVideos"`
	UsageByUser []UserStats `json:"usageByUser"`
}

type UserStats struct {
	UserId           string `json:"userId"`
	UserName         string `json:"userName"`
	Photos           int64  `json:"photos"`
	Videos           int64  `json:"videos"`
	Usage            int64  `json:"usage"`
	UsagePhotos      int64  `json:"usagePhotos"`
	UsageVideos      int64  `json:"usageVideos"`
	QuotaSizeInBytes *int64 `json:"quotaSizeInBytes"`
}

func (s *Store) GetServerStatistics() (*ServerStatisticsResponse, error) {
	endpoint := "/api/server/statistics"
	fullUrl := s.config.ImmichUrl + endpoint

	resp, err := s.ImmichFetcher(fullUrl, "GET", nil)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var result ServerStatisticsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
