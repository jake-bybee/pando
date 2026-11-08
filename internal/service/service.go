package service

import (
	"log"
	"pando/internal/peers"
	"time"
)

const assetPollInterval = time.Minute

func (s *Store) Start() {
	downloadedAssetFiles, err := s.initializeAssetManifest()
	if err != nil {
		log.Printf("Cannot initialize asset manifest: %v", err)
		return
	}

	for {
		for !s.isCanStart() {
			log.Printf("Waiting for all peers to become healthy")
			time.Sleep(2 * time.Second)
		}

		allPeers := peers.GetAllPeers()
		peerIds := make([]string, 0, len(allPeers))
		myId := ""
		for _, peer := range allPeers {
			if peer.Me {
				myId = peer.Id
			}
			peerIds = append(peerIds, peer.Id)
		}

		if myId == "" {
			log.Printf("Cannot poll assets: local peer ID not found")
			return
		}

		for {
			assetIds, scanComplete, err := s.immichStore.PollAssets()
			if err != nil {
				log.Printf("Polling assets failed: %v", err)
				break
			}

			myPics := WhichPicsAreMine(myId, assetIds, peerIds, s.config.NumReplicas)
			pendingPics := myPics[:0]
			for _, assetId := range myPics {
				if _, downloaded := downloadedAssetFiles[assetId]; !downloaded {
					pendingPics = append(pendingPics, assetId)
				}
			}

			if len(pendingPics) > 0 {
				assetFiles, err := s.immichStore.BulkDownloadWithManifest(pendingPics)
				if err != nil {
					log.Printf("Downloading assets failed: %v", err)
					s.immichStore.ResetAssetCursor()
					break
				}
				for assetId, fileName := range assetFiles {
					downloadedAssetFiles[assetId] = fileName
				}
				if err := saveAssetManifest(s.config.BackupFolderPath, downloadedAssetFiles); err != nil {
					log.Printf("Saving asset manifest failed: %v", err)
					return
				}
			}

			if scanComplete {
				log.Printf("Completed asset scan")
				break
			}
		}

		time.Sleep(assetPollInterval)
	}

}
