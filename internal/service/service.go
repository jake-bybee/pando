package service

import (
	"fmt"
	"pando/internal/peers"
	"time"
)

func (s *Store) Start() {
	if !s.isCanStart() {
		time.Sleep(2 * time.Second) // pause for a fixed time
	}

	allPeers := peers.GetAllPeers()
	peerIds := []string{}
	myId := ""

	for _, peer := range allPeers {
		if peer.Me {
			myId = peer.Id

		}
		peerIds = append(peerIds, peer.Id)
	}

	if myId == "" {
		fmt.Println("Error: myId not found among peers")
		return
	}

	assetIds, errs := s.immichStore.PollAssets()
	if errs != nil {
		fmt.Println("Error polling assets:", errs)
	}

	myPics := WhichPicsAreMine(myId, assetIds, peerIds, s.config.NumReplicas)

	s.immichStore.BulkDownload(myPics)

}
