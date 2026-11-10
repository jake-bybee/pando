package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"pando/internal/config"
	"pando/internal/immich"
	"pando/internal/peers"
	"pando/internal/service"
	"pando/internal/status"
	"pando/internal/utils"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
	firstPeer := peers.GetPeerById(id)
	if firstPeer == nil {
		t.Fatal("registered peer not found")
	}
	time.Sleep(time.Second)

	response = httptest.NewRecorder()
	body = `{"id":"` + id + `","url":"HTTP://UPDATED.EXAMPLE/"}`
	server.RegisterPeerHandler(response, httptest.NewRequest(http.MethodPost, "/registerPeer", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("duplicate registration status = %d, want 200", response.Code)
	}
	updatedPeer := peers.GetPeerById(id)
	if updatedPeer == nil || updatedPeer.Url != "http://updated.example" || updatedPeer.FirstSeen != firstPeer.FirstSeen || updatedPeer.LastSeen == firstPeer.LastSeen {
		t.Fatalf("updated peer = %+v, first peer = %+v", updatedPeer, firstPeer)
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
	body = `{"peers":[{"id":"` + batchID + `","url":"HTTP://UPDATED-BATCH.EXAMPLE/"}]}`
	response = httptest.NewRecorder()
	server.RegisterPeersReceivedHandler(response, httptest.NewRequest(http.MethodPost, "/registerPeersReceived", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("updated batch status = %d, want 200", response.Code)
	}
	batchPeer = peers.GetPeerById(batchID)
	if batchPeer == nil || batchPeer.Url != "http://updated-batch.example" {
		t.Fatalf("updated batch peer = %+v", batchPeer)
	}
}

func TestRegisterPeerHandlerDoesNotChangeLocalPeer(t *testing.T) {
	client := &http.Client{Transport: registrationTransport(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}
	server := &Server{peersStore: peers.NewStore(client), utilsStore: utils.NewStore("UTC")}
	id := uuid.NewString()
	peers.RegisterPeer(peers.Peer{
		Id:        id,
		Url:       "http://local.example",
		Me:        true,
		FirstSeen: "first",
		LastSeen:  "last",
		Status:    peers.StatusHealthy,
	})
	before := peers.GetPeerById(id)
	if before == nil {
		t.Fatal("local peer not found")
	}

	response := httptest.NewRecorder()
	body := `{"id":"` + id + `","url":"http://remote.example"}`
	server.RegisterPeerHandler(response, httptest.NewRequest(http.MethodPost, "/registerPeer", strings.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	after := peers.GetPeerById(id)
	if after == nil || *after != *before {
		t.Fatalf("local peer changed from %+v to %+v", before, after)
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

func TestWhichNodeHasThisImageValidatesImmichAssets(t *testing.T) {
	const validID = "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name       string
		query      string
		status     int
		response   string
		wantStatus int
	}{
		{"unknown file name", "fileName=missing.jpg", http.StatusOK, `{"assets":{"items":[]}}`, http.StatusNotFound},
		{"malformed asset ID", "assetId=not-an-id", http.StatusOK, `{"assets":{"items":[]}}`, http.StatusNotFound},
		{"unknown asset ID", "assetId=" + validID, http.StatusOK, `{"assets":{"items":[]}}`, http.StatusNotFound},
		{"Immich error", "assetId=" + validID, http.StatusBadGateway, `{"error":"unavailable"}`, http.StatusBadGateway},
		{"known asset without node copy", "assetId=" + validID, http.StatusOK, `{"assets":{"items":[{"id":"` + validID + `","originalFileName":"photo.jpg"}]}}`, http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			immichServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.response))
			}))
			defer immichServer.Close()
			cfg := &config.Config{BackupFolderPath: t.TempDir(), ImmichUrl: immichServer.URL, NumReplicas: 100}
			store := immich.NewStore(cfg, immichServer.Client(), immichServer.Client())
			peerClient := &http.Client{Transport: registrationTransport(func(request *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
			})}
			server := &Server{client: peerClient, envVariables: cfg, immichStore: store}
			response := httptest.NewRecorder()
			server.WhichNodeHasThisImage(response, httptest.NewRequest(http.MethodGet, "/whichNodeHasThisImage?"+test.query, nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.name == "malformed asset ID" && requests != 0 {
				t.Fatalf("Immich requests = %d, want 0", requests)
			}
			if test.name == "known asset without node copy" {
				var result struct {
					AssetID      string `json:"assetId"`
					StoringPeers []struct {
						HasImage bool
					} `json:"storingPeers"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.AssetID != validID {
					t.Fatalf("asset ID = %q, want %q", result.AssetID, validID)
				}
				for _, peer := range result.StoringPeers {
					if peer.HasImage {
						t.Fatal("known asset incorrectly reported as present on a node")
					}
				}
			}
		})
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
