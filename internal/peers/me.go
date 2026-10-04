package peers

import (
	"pando/internal/utils"

	"github.com/google/uuid"
)

func loadOrGenerateId(backupFolderPath string) string {
	if id, err := utils.LoadId(backupFolderPath); err == nil && id != "" {
		return id
	}
	uuid := uuid.NewString()
	utils.WriteId(backupFolderPath, uuid)
	return uuid
}

func Me(selfUrl string, healthStatus PeerStatus, utilsStore *utils.Store, backupFolderPath string) string {

	id := loadOrGenerateId(backupFolderPath)
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
