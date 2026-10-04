package peers

import (
	"crypto/sha256"
	"fmt"
	"pando/internal/utils"
)

func Me(selfUrl string, healthStatus string, utilsStore *utils.Store) string {

	id := fmt.Sprintf("%x", sha256.Sum256([]byte(selfUrl)))
	now := utilsStore.TimeNow()

	me := Peer{
		Me:        true,
		Url:       selfUrl,
		Id:        id,
		FirstSeen: now,
		LastSeen:  now,
		Status:    healthStatus,
	}

	RegisterPeer(me)
	return id
}
