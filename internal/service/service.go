package service

import (
	"log"
	"time"
)

const assetPollInterval = time.Minute

func (s *Store) Start() {
	downloadedAssetFiles, err := s.initializeAssetManifest()
	if err != nil {
		log.Printf("Cannot initialize asset manifest: %v", err)
		s.stopWithError("manifest-save", err)
		return
	}

	for {
		s.waitForPeers()
		myId, peerIds, ok := s.getPeerInfo()
		if !ok {
			return
		}

		s.startScan()
		for {
			assetIds, scanComplete, err := s.immichStore.PollAssets()
			if err != nil {
				log.Printf("Polling assets failed: %v", err)
				if s.status != nil {
					s.status.RecordError("poll", err)
				}
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
				s.waitForFreeSpace()
				assetFiles, err := s.immichStore.BulkDownloadWithManifest(pendingPics)
				if err != nil {
					log.Printf("Downloading assets failed: %v", err)
					if s.status != nil {
						s.status.RecordError("download", err)
					}
					s.immichStore.ResetAssetCursor()
					break
				}
				if s.status != nil {
					s.status.MarkDownloadSuccess()
				}
				for assetId, fileName := range assetFiles {
					downloadedAssetFiles[assetId] = fileName
				}
				if err := saveAssetManifest(s.config.BackupFolderPath, downloadedAssetFiles); err != nil {
					log.Printf("Saving asset manifest failed: %v", err)
					s.stopWithError("manifest-save", err)
					return
				}
			}

			if scanComplete {
				log.Printf("Completed asset scan")
				if s.status != nil {
					s.status.MarkScanCompleted()
				}
				break
			}
		}

		s.setIdle()
		time.Sleep(assetPollInterval)
	}

}
