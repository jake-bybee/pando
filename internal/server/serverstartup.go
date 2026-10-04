package server

import (
	"fmt"
	"net/http"
	"pando/internal/utils"
)

func (s *Server) relayMeToPeer(peerUrl string, myId string) error {
	retryCount := 3

	for i := 0; i < retryCount; i++ {
		fmt.Printf("Relaying my ID %v to peer %v attempt %v\n", myId, peerUrl, i+1)
		url := peerUrl + "/registerPeer"
		payload := map[string]string{
			"id":  myId,
			"url": s.envVariables.SelfUrl,
		}
		resp, err := utils.Fetcher(s.client, url, "POST", map[string]string{}, payload)
		if err != nil {
			if i < retryCount-1 {
				fmt.Printf("Failed to relay to peer: %v, retrying...\n", err)
				continue
			}
			return fmt.Errorf("failed to relay to peer: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			if i < retryCount-1 {
				fmt.Printf("Unexpected status code from peer: %v, retrying...\n", resp.StatusCode)
				continue
			}
			return fmt.Errorf("unexpected status code from peer: %v", resp.StatusCode)
		}
		return nil
	}
	return nil
}
