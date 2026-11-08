package service

import (
	"fmt"
	"log"
	"pando/internal/peers"
	"pando/internal/status"
	"time"
)

func (s *Store) waitForPeers() {
	for !s.isCanStart() {
		if s.status != nil {
			s.status.SetState(status.StateWaitingForPeers)
		}
		log.Printf("Waiting for all peers to become healthy")
		time.Sleep(2 * time.Second)
	}
}

func (s *Store) getPeerInfo() (string, []string, bool) {
	allPeers := peers.GetAllPeers()
	peerIDs := make([]string, 0, len(allPeers))
	myID := ""
	for _, peer := range allPeers {
		if peer.Me {
			myID = peer.Id
		}
		peerIDs = append(peerIDs, peer.Id)
	}
	if myID != "" {
		return myID, peerIDs, true
	}

	err := fmt.Errorf("local peer ID not found")
	log.Printf("Cannot poll assets: %v", err)
	s.stopWithError("poll", err)
	return "", nil, false
}

func (s *Store) waitForFreeSpace() {
	for {
		free, err := freeBytes(s.config.BackupFolderPath)
		if err != nil {
			log.Printf("Failed to check free space: %v", err)
		} else if free < s.config.MinFreeBytes {
			log.Printf("Not enough free space (have %d, need %d). Checking again in %s.", free, s.config.MinFreeBytes, time.Duration(s.config.StorageRecheckSeconds)*time.Second)
		} else {
			if s.status != nil {
				s.status.SetState(status.StateScanning)
			}
			return
		}
		if s.status != nil {
			s.status.SetState(status.StateWaitingForDisk)
		}
		time.Sleep(time.Duration(s.config.StorageRecheckSeconds) * time.Second)
	}
}

func (s *Store) startScan() {
	if s.status != nil {
		s.status.SetState(status.StateScanning)
		s.status.MarkScanStarted()
	}
}

func (s *Store) setIdle() {
	if s.status != nil {
		s.status.SetState(status.StateIdle)
	}
}

func (s *Store) stopWithError(source string, err error) {
	if s.status != nil {
		s.status.RecordError(source, err)
		s.status.SetState(status.StateStopped)
	}
}
