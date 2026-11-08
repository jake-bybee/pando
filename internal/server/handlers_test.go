package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"pando/internal/config"
	"pando/internal/peers"
	"pando/internal/service"
	"pando/internal/status"
	"pando/internal/utils"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type registrationTransport func(*http.Request) (*http.Response, error)

func (transport registrationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestRegistrationRequiresIDAndURL(t *testing.T) {
	server := &Server{}
	for _, test := range []struct {
		name    string
		body    string
		handler http.HandlerFunc
	}{
		{"single missing ID", `{"url":"http://peer.example"}`, server.RegisterPeerHandler},
		{"single empty ID", `{"id":" ","url":"http://peer.example"}`, server.RegisterPeerHandler},
		{"single missing URL", `{"id":"peer-id"}`, server.RegisterPeerHandler},
		{"batch missing ID", `{"peers":[{"url":"http://peer.example"}]}`, server.RegisterPeersReceivedHandler},
		{"batch missing URL", `{"peers":[{"id":"peer-id"}]}`, server.RegisterPeersReceivedHandler},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			test.handler(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body)))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.Code)
			}
		})
	}
}

func TestRegistrationPreservesIDAndRelaysURL(t *testing.T) {
	var relayed RegisterPeersReceivedRequest
	client := &http.Client{Transport: registrationTransport(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/registerPeersReceived" {
			if err := json.NewDecoder(request.Body).Decode(&relayed); err != nil {
				t.Fatalf("decode relay payload: %v", err)
			}
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}
	server := &Server{peersStore: peers.NewStore(client), utilsStore: utils.NewStore("UTC")}
	id := uuid.NewString()
	body := `{"id":"` + id + `","url":"HTTP://PEER.EXAMPLE/"}`
	response := httptest.NewRecorder()
	server.RegisterPeerHandler(response, httptest.NewRequest(http.MethodPost, "/registerPeer", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var registered peers.Peer
	if err := json.Unmarshal(response.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode registered peer: %v", err)
	}
	if registered.Id != id || registered.Url != "http://peer.example" {
		t.Fatalf("registered peer = %+v", registered)
	}
	found := false
	for _, peer := range relayed.Peers {
		if peer.Id == id && peer.Url == registered.Url {
			found = true
		}
	}
	if !found {
		t.Fatalf("relay payload does not contain peer ID and URL: %+v", relayed)
	}
	response = httptest.NewRecorder()
	server.RegisterPeerHandler(response, httptest.NewRequest(http.MethodPost, "/registerPeer", strings.NewReader(body)))
	if response.Code != http.StatusConflict {
		t.Fatalf("duplicate registration status = %d, want 409", response.Code)
	}

	batchID := uuid.NewString()
	body = `{"peers":[{"id":"` + batchID + `","url":"HTTP://BATCH.EXAMPLE/"}]}`
	response = httptest.NewRecorder()
	server.RegisterPeersReceivedHandler(response, httptest.NewRequest(http.MethodPost, "/registerPeersReceived", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("batch status = %d, want 200", response.Code)
	}
	batchPeer := peers.GetPeerById(batchID)
	if batchPeer == nil || batchPeer.Url != "http://batch.example" || batchPeer.Status != peers.StatusHealthy {
		t.Fatalf("registered batch peer = %+v", batchPeer)
	}
}

func TestBatchRegistrationValidatesBeforeRegistering(t *testing.T) {
	id := uuid.NewString()
	body := `{"peers":[{"id":"` + id + `","url":"http://valid.example"},{"url":"http://missing-id.example"}]}`
	response := httptest.NewRecorder()
	server := &Server{}
	server.RegisterPeersReceivedHandler(response, httptest.NewRequest(http.MethodPost, "/registerPeersReceived", strings.NewReader(body)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if peers.GetPeerById(id) != nil {
		t.Fatal("invalid batch partially registered a peer")
	}
}

func TestManifestEndpoint(t *testing.T) {
	backupFolder := t.TempDir()
	settingsFolder := filepath.Join(backupFolder, "settings")
	if err := os.MkdirAll(settingsFolder, 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(settingsFolder, "asset-manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"asset-c":"three.jpg","asset-a":"one.jpg","asset-b":"two.jpg"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	server := &Server{envVariables: &config.Config{BackupFolderPath: backupFolder}}
	originalMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = originalMux })
	server.RegisterRoutes()

	for _, test := range []struct {
		name       string
		query      string
		wantIDs    []string
		nextCursor string
	}{
		{"default", "", []string{"asset-a", "asset-b", "asset-c"}, ""},
		{"first page", "?limit=2", []string{"asset-a", "asset-b"}, "asset-b"},
		{"last page", "?limit=2&after=asset-b", []string{"asset-c"}, ""},
		{"past end", "?after=asset-z", []string{}, ""},
		{"maximum limit", "?limit=1000", []string{"asset-a", "asset-b", "asset-c"}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/manifest"+test.query, nil))
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response = %d, headers = %v", response.Code, response.Header())
			}
			var page service.AssetManifestPage
			if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			ids := make([]string, 0, len(page.Items))
			for _, item := range page.Items {
				ids = append(ids, item.AssetID)
			}
			if page.Items == nil || page.Total != 3 || !reflect.DeepEqual(ids, test.wantIDs) {
				t.Fatalf("page = %+v, want IDs %v and total 3", page, test.wantIDs)
			}
			if test.nextCursor == "" {
				if page.NextCursor != nil {
					t.Fatalf("next cursor = %q, want nil", *page.NextCursor)
				}
			} else if page.NextCursor == nil || *page.NextCursor != test.nextCursor {
				t.Fatalf("next cursor = %v, want %q", page.NextCursor, test.nextCursor)
			}
		})
	}

	for _, limit := range []string{"", "0", "-1", "1001", "abc", "1.5", "99999999999999999999999999"} {
		response := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/manifest?limit="+limit, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("limit %q: status = %d, want 400", limit, response.Code)
		}
	}
	response := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/manifest", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("POST response = %d, Allow = %q", response.Code, response.Header().Get("Allow"))
	}

	if err := os.WriteFile(manifestPath, []byte(`invalid JSON`), 0o644); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/manifest", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("corrupt manifest status = %d, want 500", response.Code)
	}
	if err := os.Remove(manifestPath); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/manifest", nil))
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"items":[],"total":0,"nextCursor":null}` {
		t.Fatalf("missing manifest response = %d %s", response.Code, response.Body.String())
	}
}

func TestStorageEndpoint(t *testing.T) {
	server := &Server{envVariables: &config.Config{BackupFolderPath: t.TempDir()}}
	response := httptest.NewRecorder()
	server.StorageHandler(response, httptest.NewRequest(http.MethodGet, "/storage", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	var storage storageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &storage); err != nil {
		t.Fatal(err)
	}
	if storage.Path == "" || storage.TotalBytes == 0 || storage.FreeBytes == 0 ||
		storage.UsedBytes != storage.TotalBytes-storage.FreeBytes {
		t.Fatalf("storage response = %+v", storage)
	}
}

func TestStatusEndpointEmptyAndDoesNotExposeToken(t *testing.T) {
	token := "secret-token-value"
	server := &Server{
		envVariables: &config.Config{
			ImmichUrl:      "https://user:password@immich.example:2283/api?token=hidden",
			ImmichApiToken: token,
			NumReplicas:    2,
			NumPeers:       4,
			ListenPort:     "8080",
		},
		status:  status.NewTracker(),
		version: "dev",
	}
	response := httptest.NewRecorder()
	server.StatusHandler(response, httptest.NewRequest(http.MethodGet, "/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, token) || strings.Contains(body, "password") || strings.Contains(body, "hidden") {
		t.Fatalf("status response exposed a secret: %s", body)
	}
	var decoded map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["lastScanStarted"] != nil || decoded["lastScanCompleted"] != nil ||
		decoded["lastSuccessfulDownload"] != nil || decoded["lastError"] != nil {
		t.Fatalf("empty status has non-null event values: %s", body)
	}
	settings, ok := decoded["settings"].(map[string]any)
	if !ok || settings["immichUrl"] != "https://immich.example:2283" {
		t.Fatalf("settings = %#v", decoded["settings"])
	}
}

func TestCollectNodeStatusReportsDeadPeer(t *testing.T) {
	server := &Server{client: &http.Client{}}
	result := server.collectNodeStatus(peers.Peer{
		Id:  "dead-peer",
		Url: "http://127.0.0.1:1",
	})
	if result.Reachable {
		t.Fatal("dead peer reported as reachable")
	}
	if len(result.Errors) == 0 {
		t.Fatal("dead peer returned no errors")
	}
}
