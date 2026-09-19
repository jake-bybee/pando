package health

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"pando/internal/config"
	"slices"
)

var (
	apiKeyPermissionsMethod = "GET"
	healthCheck             = "/health"
)

type validateTokenResponse struct {
	AuthStatus bool `json:"authStatus"`
}

type apiKeyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
}

var client = &http.Client{}

func RunHealthCheck(config config.Config) bool {
	if !isImmichReachable(config.ImmichUrl, config.ImmichApiToken) {
		log.Printf("%s is not reachable", config.ImmichUrl)
		return false
	}

	log.Printf("%s is reachable and healthy", config.ImmichUrl)
	return true
}

func isImmichReachable(immichUrl, apiKey string) bool {

	valid, err := immichApiKeyValid(immichUrl, apiKey)
	if err != nil {
		log.Printf("Failed to get API key permissions for %s: %v", immichUrl, err)
		return false
	}
	if !valid {
		return false
	}

	sufficient, err := immichApiKeyPermissionsSufficient(immichUrl, apiKey)
	if err != nil {
		log.Printf("Failed to get API key permissions for %s: %v", immichUrl, err)
		return false
	}
	if !sufficient {
		return false
	}
	return true
}

func immichApiKeyValid(immichUrl string, apiKey string) (bool, error) {
	apiKeyPermissions := "/api/auth/validateToken"
	fullUrl := immichUrl + apiKeyPermissions
	header := http.Header{"x-api-key": []string{apiKey}}

	req, err := http.NewRequest("POST", fullUrl, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", immichUrl, err)
		return false, err
	}
	req.Header = header

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", immichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected response status for %s: %v", immichUrl, resp.Status)
		return false, fmt.Errorf("unexpected response status: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for %s: %v", immichUrl, err)
		return false, err
	}
	var validateResp validateTokenResponse
	err = json.Unmarshal(body, &validateResp)
	if err != nil {
		log.Printf("Failed to unmarshal response body for %s: %v", immichUrl, err)
		return false, err
	}
	if !validateResp.AuthStatus {
		log.Printf("Invalid API key for %s", immichUrl)
		return false, fmt.Errorf("invalid API key")
	}

	log.Printf("Api key for %s is valid", immichUrl)

	return true, nil

}

func immichApiKeyPermissionsSufficient(immichUrl string, apiKey string) (bool, error) {
	endpoint := "/api/api-keys/me"
	fullUrl := immichUrl + endpoint
	header := http.Header{"x-api-key": []string{apiKey}}

	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", immichUrl, err)
		return false, err
	}
	req.Header = header

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", immichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected response status for %s: %v", immichUrl, resp.Status)
		return false, fmt.Errorf("unexpected response status: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for %s: %v", immichUrl, err)
		return false, err
	}
	var permissionsResp apiKeyPermissionsResponse
	err = json.Unmarshal(body, &permissionsResp)
	if err != nil {
		log.Printf("Failed to unmarshal response body for %s: %v", immichUrl, err)
		return false, err
	}

	hasPermissions := checkIfPermissionsSufficient(permissionsResp, []string{""}) // Replace with actual required permissions
	if !hasPermissions {
		log.Printf("Insufficient API key permissions for %s", immichUrl)
		return false, fmt.Errorf("insufficient API key permissions")
	}

	log.Printf("API key permissions for %s are sufficient", immichUrl)

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
