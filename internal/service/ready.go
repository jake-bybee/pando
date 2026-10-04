package service

import (
	"pando/internal/peers"
)

func (s *Store) isCanStart() bool {
	return peers.IsAllPeersActiveAndHealthy(s.config.NumPeers)
}
