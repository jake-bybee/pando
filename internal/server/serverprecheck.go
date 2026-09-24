package server

import "pando/internal/health"

func runServerPrecheck(healthStore *health.Store) bool {

	healthStatus := healthStore.RunHealthCheck()
	return healthStatus

}
