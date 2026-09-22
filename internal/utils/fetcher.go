package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pando/internal/config"
)

type Store struct {
	client *http.Client
	config config.Config
}

func NewStore(config config.Config, client *http.Client) *Store {
	return &Store{
		client: client,
		config: config,
	}
}

func (s *Store) ImmichFetcher(url string, method string, payload interface{}) (*http.Response, error) {
	var req *http.Request
	var err error
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload for %s: %v", url, err)
		}
		req, err = http.NewRequest(method, url, bytes.NewReader(payloadBytes))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %v", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.config.ImmichApiToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for %s: %v", url, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected response status for %s: %v, body: %s", url, resp.Status, string(body))
	}

	return resp, nil
}
