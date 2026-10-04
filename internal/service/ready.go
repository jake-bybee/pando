package service

import (
	"pando/internal/peers"
)

func (s *Store) isCanStart() bool {
	return peers.IsAllPeersActive(s.config.NumPeers)
}
