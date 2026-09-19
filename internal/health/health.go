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

type Store struct {
	config config.Config
	client *http.Client
}

type validateTokenResponse struct {
	AuthStatus bool `json:"authStatus"`
}

type apiKeyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
}

func NewStore(config config.Config, client *http.Client) *Store {
	return &Store{
		config: config,
		client: client,
	}
}

func (s *Store) RunHealthCheck() bool {
	if !s.isImmichReachable() {
		log.Printf("%s is not reachable", s.config.ImmichUrl)
		return false
	}

	return true
}

func (s *Store) isImmichReachable() bool {

	valid, err := s.immichApiKeyValid()
	if err != nil {
		log.Printf("Failed to get API key permissions for %s: %v", s.config.ImmichUrl, err)
		return false
	}
	if !valid {
		return false
	}

	sufficient, err := s.immichApiKeyPermissionsSufficient()
	if err != nil {
		log.Printf("Failed to get API key permissions for %s: %v", s.config.ImmichUrl, err)
		return false
	}
	if !sufficient {
		return false
	}
	return true
}

func (s *Store) immichApiKeyValid() (bool, error) {
	apiKeyPermissions := "/api/auth/validateToken"
	fullUrl := s.config.ImmichUrl + apiKeyPermissions
	header := http.Header{"x-api-key": []string{s.config.ImmichApiToken}}

	req, err := http.NewRequest("POST", fullUrl, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	req.Header = header

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected response status for %s: %v", s.config.ImmichUrl, resp.Status)
		return false, fmt.Errorf("unexpected response status: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	var validateResp validateTokenResponse
	err = json.Unmarshal(body, &validateResp)
	if err != nil {
		log.Printf("Failed to unmarshal response body for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	if !validateResp.AuthStatus {
		log.Printf("Invalid API key for %s", s.config.ImmichUrl)
		return false, fmt.Errorf("invalid API key")
	}

	log.Printf("Api key for %s is valid", s.config.ImmichUrl)

	return true, nil

}

func (s *Store) immichApiKeyPermissionsSufficient() (bool, error) {
	endpoint := "/api/api-keys/me"
	fullUrl := s.config.ImmichUrl + endpoint
	header := http.Header{"x-api-key": []string{s.config.ImmichApiToken}}

	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	req.Header = header

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected response status for %s: %v", s.config.ImmichUrl, resp.Status)
		return false, fmt.Errorf("unexpected response status: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	var permissionsResp apiKeyPermissionsResponse
	err = json.Unmarshal(body, &permissionsResp)
	if err != nil {
		log.Printf("Failed to unmarshal response body for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}

	hasPermissions := checkIfPermissionsSufficient(permissionsResp, []string{""}) // Replace with actual required permissions
	if !hasPermissions {
		log.Printf("Insufficient API key permissions for %s", s.config.ImmichUrl)
		return false, fmt.Errorf("insufficient API key permissions")
	}

	log.Printf("API key permissions for %s are sufficient", s.config.ImmichUrl)

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
