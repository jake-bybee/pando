package health

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"pando/internal/config"
	"pando/internal/immich"
	"pando/internal/system"
	"slices"
)

type validateTokenResponse struct {
	AuthStatus bool `json:"authStatus"`
}

type apiKeyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
}

func RunHealthCheck(config *config.Config) bool {
	if !isImmichReachable(config) {
		log.Printf("%s is not reachable", config.ImmichUrl)
		return false
	}
	if err := system.SystemHealthCheck(); err != nil {
		log.Printf("System health check failed: %v", err)
		return false
	}

	return true
}

func isImmichReachable(config *config.Config) bool {

	valid, err := immichApiKeyValid(config)
	if err != nil {
		log.Printf("Failed to get API key permissions for %s: %v", config.ImmichUrl, err)
		return false
	}
	if !valid {
		return false
	}

	sufficient, err := immichApiKeyPermissionsSufficient(config)
	if err != nil {
		log.Printf("Failed to get API key permissions for %s: %v", config.ImmichUrl, err)
		return false
	}
	if !sufficient {
		return false
	}
	return true
}

func immichApiKeyValid(config *config.Config) (bool, error) {
	apiKeyPermissions := "/api/auth/validateToken"
	fullUrl := config.ImmichUrl + apiKeyPermissions
	log.Printf("Checking if API key for %s is valid", fullUrl)

	resp, err := immich.ImmichFetcher(fullUrl, "POST", nil)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", config.ImmichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for %s: %v", config.ImmichUrl, err)
		return false, err
	}
	var validateResp validateTokenResponse
	err = json.Unmarshal(body, &validateResp)
	if err != nil {
		log.Printf("Failed to unmarshal response body for %s: %v", config.ImmichUrl, err)
		return false, err
	}
	if !validateResp.AuthStatus {
		log.Printf("Invalid API key for %s", config.ImmichUrl)
		return false, fmt.Errorf("invalid API key")
	}

	log.Printf("API key for %s is valid", config.ImmichUrl)

	return true, nil

}

func immichApiKeyPermissionsSufficient(config *config.Config) (bool, error) {
	endpoint := "/api/api-keys/me"
	fullUrl := config.ImmichUrl + endpoint

	resp, err := immich.ImmichFetcher(fullUrl, "GET", nil)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", config.ImmichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for %s: %v", config.ImmichUrl, err)
		return false, err
	}
	var permissionsResp apiKeyPermissionsResponse
	err = json.Unmarshal(body, &permissionsResp)
	if err != nil {
		log.Printf("Failed to unmarshal response body for %s: %v", config.ImmichUrl, err)
		return false, err
	}

	hasPermissions := checkIfPermissionsSufficient(permissionsResp, []string{""}) // Replace with actual required permissions
	if !hasPermissions {
		log.Printf("Insufficient API key permissions for %s", config.ImmichUrl)
		return false, fmt.Errorf("insufficient API key permissions")
	}

	log.Printf("API key permissions for %s are sufficient", config.ImmichUrl)

	return true, nil
}

func checkIfPermissionsSufficient(permissionsResp apiKeyPermissionsResponse, requiredPermission []string) bool {
	if slices.Contains(permissionsResp.Permissions, "all") {
		return true
	}

	for _, reqPerm := range requiredPermission {
		if !slices.Contains(permissionsResp.Permissions, reqPerm) {
			return false
		}

	}
	return true
}
