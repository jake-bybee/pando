package immich

import (
	"context"
	"net/http"
	appconfig "pando/internal/config"
	"pando/internal/status"
	"pando/internal/utils"
	"sync"
)

type Store struct {
	client         *http.Client
	downloadClient *http.Client
	config         *appconfig.Config
	status         *status.Tracker
	currentCursor  struct {
		cursor string
		mu     sync.Mutex
	}
}

func NewStore(cfg *appconfig.Config, httpClient *http.Client, downloadClient *http.Client, trackers ...*status.Tracker) *Store {
	var tracker *status.Tracker
	if len(trackers) > 0 {
		tracker = trackers[0]
	}
	return &Store{config: cfg, client: httpClient, downloadClient: downloadClient, status: tracker}
}

func (s *Store) ImmichDownloadFetcher(ctx context.Context, url string, method string, payload interface{}) (*http.Response, error) {
	return utils.FetcherWithContext(ctx, s.downloadClient, url, method, map[string]string{"x-api-key": s.config.ImmichApiToken}, payload)
}

func (s *Store) ImmichFetcher(url string, method string, payload interface{}) (*http.Response, error) {
	return utils.Fetcher(s.client, url, method, map[string]string{"x-api-key": s.config.ImmichApiToken}, payload)
}
