package immich

import (
	"fmt"
	"sync"
)

var currentCursor struct {
	cursor string
	mu     sync.Mutex
}

func PollAssets() ([]string, error) {
	currentCursor.mu.Lock()
	defer currentCursor.mu.Unlock()
	assets, nextCursor, err := GetChunkAssetIds(currentCursor.cursor)
	if err != nil {
		fmt.Printf("Polling assets failed due to error %v\n", err)
		return nil, fmt.Errorf("Polling assets failed due to error %v", err)
	}

	if nextCursor == "" {
		fmt.Println("Successfully polled all pictures")
	}

	currentCursor.cursor = nextCursor

	fmt.Printf("Retrieved %d assets with next cursor=%q\n", len(assets), currentCursor.cursor)

	return assets, nil
}
