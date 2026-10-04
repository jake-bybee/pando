package health

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"pando/internal/config"
	"pando/internal/immich"
	"slices"

	"golang.org/x/sys/unix"
)

type Store struct {
	config      *config.Config
	immichStore *immich.Store
}

func NewStore(cfg *config.Config, immichStore *immich.Store) *Store {
	return &Store{config: cfg, immichStore: immichStore}
}

type validateTokenResponse struct {
	AuthStatus bool `json:"authStatus"`
}

type apiKeyPermissionsResponse struct {
	Permissions []string `json:"permissions"`
}

func (s *Store) RunHealthCheck() bool {
	if !s.isImmichReachable() {
		log.Printf("%s is not reachable", s.config.ImmichUrl)
		return false
	}
	if err := s.SystemHealthCheck(); err != nil {
		log.Printf("System health check failed: %v", err)
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
	log.Printf("Checking if API key for %s is valid", fullUrl)

	resp, err := s.immichStore.ImmichFetcher(fullUrl, "POST", nil)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

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

	log.Printf("API key for %s is valid", s.config.ImmichUrl)

	return true, nil

}

func (s *Store) immichApiKeyPermissionsSufficient() (bool, error) {
	endpoint := "/api/api-keys/me"
	fullUrl := s.config.ImmichUrl + endpoint

	resp, err := s.immichStore.ImmichFetcher(fullUrl, "GET", nil)
	if err != nil {
		log.Printf("Failed to execute request for %s: %v", s.config.ImmichUrl, err)
		return false, err
	}
	defer resp.Body.Close()

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

func (s *Store) SystemHealthCheck() error {
	if s.config.BackupFolderPath == "" {
		return fmt.Errorf("backup folder path is not configured")
	}
	if _, err := os.Stat(s.config.BackupFolderPath); os.IsNotExist(err) {
		return fmt.Errorf("backup folder path does not exist")
	}

	if !s.isSystemStorageSufficient() {
		return fmt.Errorf("system storage is insufficient")
	}
	return nil
}

func (s *Store) getSystemStorage() (uint64, error) {
	var stat unix.Statfs_t

	if err := unix.Statfs(s.config.BackupFolderPath, &stat); err != nil {
		return 0, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)

	fmt.Print("Checking system storage for path: ", s.config.BackupFolderPath)
	fmt.Print(" Total:", total)
	fmt.Printf(" Free: %d\n", free)

	return free, nil
}

func (s *Store) isSystemStorageSufficient() bool {
	user, err := s.immichStore.GetUsers()
	if err != nil {
		return false
	}

	free, err := s.getSystemStorage()
	if err != nil {
		return false
	}
	return free >= uint64(user.QuotaUsageInBytes)
}
