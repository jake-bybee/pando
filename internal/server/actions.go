package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"pando/internal/peers"
)

func (s *Server) RelayNewPeer() {
	endpoint := "/registerPeersReceived"
	for _, peer := range peers.GetAllPeers() {
		url := peer.Url + endpoint
		data, err := json.Marshal(url)

		if err != nil {
			fmt.Printf("Failed to create request for %s: %v\n", peer.Url, err)
			continue
		}
		req, err := http.NewRequest("POST", peer.Url, bytes.NewReader(data))
		if err != nil {
			fmt.Printf("Failed to create request for %s: %v\n", peer.Url, err)
			continue
		}
		_, err = s.client.Do(req)
		if err != nil {
			fmt.Printf("Failed to relay to %s: %v\n", peer.Url, err)
			continue
		}

		fmt.Printf("Successfully relayed new peer to %s\n", peer.Url)
	}
}
