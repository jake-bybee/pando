package immich

import "encoding/json"

type User struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	QuotaUsageInBytes int64  `json:"quotaUsageInBytes"`
	StorageSize       int64  `json:"storageSize"`
}

func (s *Store) GetUsers() (User, error) {
	url := s.config.ImmichUrl + "/api/users/me"
	resp, err := s.ImmichFetcher(url, "GET", nil)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	var users User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return User{}, err
	}

	return users, nil
}
