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

func ParseNumReplicas() int {
	numReplicasStr := os.Getenv("NUM_REPLICAS")
	if numReplicasStr == "" {
		return 0
	}
	numReplicas, err := strconv.Atoi(numReplicasStr)
	if err != nil {
		return 0
	}
	return numReplicas
}
