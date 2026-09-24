package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pando/internal/peers"
)

type RegisterPeerRequest struct {
	Url string `json:"url"`
}

type RegisterPeersReceivedRequest struct {
	Peers []RegisterPeerRequest `json:"peers"`
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[/health] Received request, running health check")
	defer r.Body.Close()
	success := s.healthStore.RunHealthCheck()

	if success {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is healthy"))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server is unhealthy"))
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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request payload"))
		fmt.Println("[/registerPeer] Invalid request payload")
		return
	}

	urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(registerRequest.Url)))
	if peers.GetPeerById(urlHash) != nil {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("Peer already registered"))
		fmt.Println("[/registerPeer] Peer already registered with URL:", registerRequest.Url)
		return
	}

	registeredTime := s.utils.TimeNow()
	peers.RegisterPeer(peers.Peer{
		Url:       registerRequest.Url,
		Id:        urlHash,
		FirstSeen: registeredTime,
		LastSeen:  registeredTime,
		Status:    "",
	})
	fmt.Println("[/registerPeer] Registered peer:", registerRequest.Url)

	var status string
	isHealthy := s.peersStore.CheckPeerHealth(urlHash)
	if !isHealthy {
		status = "unhealthy"
	} else {
		status = "healthy"
	}

	registeredPeer := peers.UpdatePeerStatus(urlHash, status)

	jsonString, err := json.Marshal(registeredPeer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to serialize registered peer"))
		fmt.Println("[/registerPeer] Failed to serialize registered peer:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(jsonString))
	fmt.Println("[/registerPeer] Successfully registered peer with URL:", registerRequest.Url)
}

func (s *Server) RelayPeersHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[/relayPeers] Received request to relay peers")
	defer r.Body.Close()

	peersList := peers.GetAllPeers()

	jsonString, err := json.Marshal(peersList)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to serialize peers list"))
		fmt.Println("[/relayPeers] Failed to serialize peers list:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(jsonString))
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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request payload"))
		fmt.Println("[/registerPeersReceived] Invalid request payload")
		return
	}

	for _, peerInfo := range registerRequest.Peers {
		urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(peerInfo.Url)))
		if peers.GetPeerById(urlHash) == nil {
			registeredTime := s.utils.TimeNow()
			peers.RegisterPeer(peers.Peer{
				Url:       peerInfo.Url,
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
			fmt.Println("[/registerPeersReceived] Registered peer:", peerInfo.Url)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Peers registration processed"))
	fmt.Println("[/registerPeersReceived] Successfully processed peers registration")
}
