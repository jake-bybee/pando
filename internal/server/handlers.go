package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pando/internal/peers"
	"pando/internal/utils"
)

type RegisterPeerRequest struct {
	Url string `json:"url"`
}

type RegisterPeersReceivedRequest struct {
	Peers []RegisterPeerRequest `json:"peers"`
}

func writeJSONResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[/health] Received request, running health check")
	defer r.Body.Close()
	success := s.healthStore.RunHealthCheck()

	if success {
		writeJSONResponse(w, http.StatusOK, "Server is healthy")
	} else {
		writeJSONResponse(w, http.StatusInternalServerError, "Server is unhealthy")
	}
}

func (s *Server) RegisterPeerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[/registerPeer] Received request to register peer")
	defer r.Body.Close()
	payload := io.Reader(r.Body)

	var registerRequest RegisterPeerRequest
	decoder := json.NewDecoder(payload)
	err := decoder.Decode(&registerRequest)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, "Invalid request payload")
		fmt.Println("[/registerPeer] Invalid request payload")
		return
	}
	normalizedUrl, err := utils.NormalizeURL(registerRequest.Url)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, "Invalid URL")
		fmt.Println("[/registerPeer] Invalid URL:", registerRequest.Url)
		return
	}
	urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(normalizedUrl)))
	if peers.GetPeerById(urlHash) != nil {
		writeJSONResponse(w, http.StatusConflict, "Peer already registered")
		fmt.Println("[/registerPeer] Peer already registered with URL:", normalizedUrl)
		return
	}

	registeredTime := utils.TimeNow()
	peers.RegisterPeer(peers.Peer{
		Url:       normalizedUrl,
		Id:        urlHash,
		FirstSeen: registeredTime,
		LastSeen:  registeredTime,
		Status:    "",
	})
	fmt.Println("[/registerPeer] Registered peer:", normalizedUrl)

	var status string
	isHealthy := s.peersStore.CheckPeerHealth(urlHash)
	if !isHealthy {
		status = "unhealthy"
	} else {
		status = "healthy"
	}

	registeredPeer := peers.UpdatePeerStatus(urlHash, status)

	writeJSONResponse(w, http.StatusOK, registeredPeer)
	fmt.Println("[/registerPeer] Successfully registered peer with URL:", normalizedUrl)
}

func (s *Server) RelayPeersHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[/relayPeers] Received request to relay peers")
	defer r.Body.Close()

	peersList := peers.GetAllPeers()

	writeJSONResponse(w, http.StatusOK, peersList)
	fmt.Println("[/relayPeers] Successfully relayed peers list")
}

func (s *Server) RegisterPeersReceivedHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[/registerPeersReceived] Received request to register peers")
	defer r.Body.Close()

	payload := io.Reader(r.Body)

	var registerRequest RegisterPeersReceivedRequest
	decoder := json.NewDecoder(payload)
	err := decoder.Decode(&registerRequest)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, "Invalid request payload")
		fmt.Println("[/registerPeersReceived] Invalid request payload")
		return
	}

	for _, peerInfo := range registerRequest.Peers {
		normalizedUrl, err := utils.NormalizeURL(peerInfo.Url)
		if err != nil {
			fmt.Println("[/registerPeersReceived] Invalid URL:", peerInfo.Url)
			continue
		}
		urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(normalizedUrl)))
		if peers.GetPeerById(urlHash) == nil {
			registeredTime := utils.TimeNow()
			peers.RegisterPeer(peers.Peer{
				Url:       normalizedUrl,
				Id:        urlHash,
				FirstSeen: registeredTime,
				LastSeen:  registeredTime,
				Status:    "",
			})
			isHealthy := s.peersStore.CheckPeerHealth(urlHash)
			if isHealthy {
				peers.UpdatePeerStatus(urlHash, "healthy")
			} else {
				peers.UpdatePeerStatus(urlHash, "unhealthy")
			}
			fmt.Println("[/registerPeersReceived] Registered peer:", normalizedUrl)
		}
	}

	writeJSONResponse(w, http.StatusOK, "Peers registration processed")
	fmt.Println("[/registerPeersReceived] Successfully processed peers registration")
}
