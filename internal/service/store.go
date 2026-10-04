package service

import (
	"pando/internal/config"
)

type Store struct {
	config config.Config
}

func NewStore(config config.Config) *Store {
	return &Store{
		config: config,
	}
}
