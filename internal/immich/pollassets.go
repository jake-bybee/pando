package immich

import (
	"fmt"
	"log"
)

func (s *Store) PollAssets() ([]string, bool, error) {
	s.currentCursor.mu.Lock()
	defer s.currentCursor.mu.Unlock()
	assets, nextCursor, scanComplete, err := s.GetChunkAssetIds(s.currentCursor.cursor)
	if err != nil {
		return nil, false, fmt.Errorf("polling assets failed: %w", err)
	}

	s.currentCursor.cursor = nextCursor

	log.Printf("Retrieved %d assets; scan complete=%t", len(assets), scanComplete)

	return assets, scanComplete, nil
}

func (s *Store) ResetAssetCursor() {
	s.currentCursor.mu.Lock()
	defer s.currentCursor.mu.Unlock()
	s.currentCursor.cursor = ""
}
