package health

import (
	"net/http"
	"net/http/httptest"
	"pando/internal/config"
	"pando/internal/immich"
	"path/filepath"
	"testing"
)

func TestStoresKeepIndependentHealthConfig(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-token" {
			t.Error("unexpected API key")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/validateToken":
			_, _ = w.Write([]byte(`{"authStatus":true}`))
		case "/api/api-keys/me":
			_, _ = w.Write([]byte(`{"permissions":["all"]}`))
		case "/api/users/me":
			_, _ = w.Write([]byte(`{"quotaUsageInBytes":0}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	newStore := func(path string) *Store {
		cfg := &config.Config{ImmichUrl: api.URL, ImmichApiToken: "test-token", BackupFolderPath: path}
		return NewStore(cfg, immich.NewStore(cfg, api.Client()))
	}
	valid := newStore(t.TempDir())
	unconfigured := newStore("")
	missing := newStore(filepath.Join(t.TempDir(), "missing"))

	if err := unconfigured.SystemHealthCheck(); err == nil || err.Error() != "backup folder path is not configured" {
		t.Fatalf("unconfigured SystemHealthCheck() = %v", err)
	}
	if err := missing.SystemHealthCheck(); err == nil || err.Error() != "backup folder path does not exist" {
		t.Fatalf("missing SystemHealthCheck() = %v", err)
	}
	if !valid.RunHealthCheck() {
		t.Fatal("valid store failed health check after creating other stores")
	}
	if unconfigured.RunHealthCheck() {
		t.Fatal("unconfigured store passed health check")
	}
}
