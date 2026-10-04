package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"pando/internal/peers"
	"pando/internal/utils"
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
