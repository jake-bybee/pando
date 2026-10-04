package service

import (
	"pando/internal/config"
	"pando/internal/immich"
)

type Store struct {
	config      *config.Config
	immichStore *immich.Store
}

func NewStore(config *config.Config, immichStore *immich.Store) *Store {
	return &Store{
		config:      config,
		immichStore: immichStore,
	}
}
