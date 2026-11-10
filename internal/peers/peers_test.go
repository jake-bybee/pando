package peers

import (
	"fmt"
	"sync"
	"testing"
)

func TestUpsertPeerRaceFree(t *testing.T) {
	id := fmt.Sprintf("race-peer-%s", t.Name())
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			UpsertPeer(id, fmt.Sprintf("http://peer-%d.example", i), fmt.Sprintf("time-%d", i))
		}(i)
	}
	wg.Wait()

	peer := GetPeerById(id)
	if peer == nil {
		t.Fatal("peer was not registered")
	}
	if peer.FirstSeen == "" || peer.LastSeen == "" || peer.Url == "" {
		t.Fatalf("peer = %+v, want populated timestamps and URL", peer)
	}
}
