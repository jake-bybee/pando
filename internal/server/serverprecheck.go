package server

import (
	"pando/internal/config"
	"pando/internal/health"
	"pando/internal/immich"
)

func runServerPrecheck(envVariables *config.Config, immichStore *immich.Store) bool {

	healthStatus := health.RunHealthCheck(envVariables, immichStore)
	return healthStatus

}
