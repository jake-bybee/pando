package immich

import (
	"fmt"
)

func (s *Store) PollAssets() ([]string, error) {
	s.currentCursor.mu.Lock()
	defer s.currentCursor.mu.Unlock()
	assets, nextCursor, err := s.GetChunkAssetIds(s.currentCursor.cursor)
	if err != nil {
		fmt.Printf("Polling assets failed due to error %v\n", err)
		return nil, fmt.Errorf("Polling assets failed due to error %v", err)
	}

	if nextCursor == "" {
		fmt.Println("Successfully polled all pictures")
	}

	s.currentCursor.cursor = nextCursor

	fmt.Printf("Retrieved %d assets with next cursor=%q\n", len(assets), s.currentCursor.cursor)

	return assets, nil
}
