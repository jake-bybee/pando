package service

import (
	"pando/internal/config"
	"pando/internal/immich"
	"pando/internal/status"
)

type Store struct {
	config      *config.Config
	immichStore *immich.Store
	status      *status.Tracker
}

func NewStore(config *config.Config, immichStore *immich.Store, trackers ...*status.Tracker) *Store {
	var tracker *status.Tracker
	if len(trackers) > 0 {
		tracker = trackers[0]
	}
	return &Store{
		config:      config,
		immichStore: immichStore,
		status:      tracker,
	}
}
