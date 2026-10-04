package service

import (
	"cmp"
	"hash/fnv"
	"slices"
)

func WhichPicsAreMine(myId string, assetIds []string, peerIDs []string, replicas int) []string {

	myPics := []string{}
	for _, assetID := range assetIds {
		storingPeers := whichPeersShouldStore(assetID, peerIDs, replicas)

		if slices.Contains(storingPeers, myId) {
			myPics = append(myPics, assetID)
		}
	}
	return myPics
}

func score(assetID, peerID string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(peerID + ":" + assetID))
	return h.Sum64()
}

type scored struct {
	peerID string
	score  uint64
}

func whichPeersShouldStore(assetID string, peerIDs []string, replicas int) []string {
	scores := make([]scored, 0, len(peerIDs))
	for _, id := range peerIDs {
		scores = append(scores, scored{id, score(assetID, id)})
	}

	slices.SortFunc(scores, func(a, b scored) int {
		if c := cmp.Compare(b.score, a.score); c != 0 {
			return c
		}
		return cmp.Compare(a.peerID, b.peerID)
	})

	n := min(replicas, len(scores))
	out := make([]string, n)
	for i := range n {
		out[i] = scores[i].peerID
	}
	return out
}
