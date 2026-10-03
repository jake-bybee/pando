package server

import (
	"pando/internal/config"
	"pando/internal/health"
)

func runServerPrecheck(envVariables *config.Config) bool {

	healthStatus := health.RunHealthCheck(envVariables)
	return healthStatus

}
