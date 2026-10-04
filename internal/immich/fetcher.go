package immich

import (
	"net/http"
	appconfig "pando/internal/config"
	"pando/internal/utils"
	"sync"
)

type Store struct {
	client        *http.Client
	config        *appconfig.Config
	currentCursor struct {
		cursor string
		mu     sync.Mutex
	}
}

func NewStore(cfg *appconfig.Config, httpClient *http.Client) *Store {
	return &Store{config: cfg, client: httpClient}
}

func (s *Store) ImmichFetcher(url string, method string, payload interface{}) (*http.Response, error) {
	return utils.Fetcher(s.client, url, method, map[string]string{"x-api-key": s.config.ImmichApiToken}, payload)

}
