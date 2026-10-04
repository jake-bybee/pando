package config

import (
	"os"
	"strconv"
)

func ParseNumPeers() int {
	numPeersStr := os.Getenv("NUM_PEERS")
	if numPeersStr == "" {
		return 0
	}
	numPeers, err := strconv.Atoi(numPeersStr)
	if err != nil {
		return 0
	}
	return numPeers
}
