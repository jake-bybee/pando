package service

import (
	"fmt"
	"pando/internal/peers"
)

func (s *Store) Start() {
	// if !s.isCanStart() {
	// 	return
	// }

	allPeers := peers.GetAllPeers()
	peerIds := []string{}
	myId := ""

	for _, peer := range allPeers {
		if peer.Me {
			myId = peer.Id
			continue
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

	fmt.Println("Got assets")

	myPics := WhichPicsAreMine(myId, assetIds, peerIds, s.config.NumReplicas)

	s.immichStore.BulkDownload(myPics)

}
