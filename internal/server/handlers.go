package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"pando/internal/peers"
	"pando/internal/utils"
	"strings"
)

type RegisterPeerRequest = peers.RegisterPeerRequest

type RegisterPeersReceivedRequest = peers.RegisterPeersReceivedRequest

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
	if strings.TrimSpace(registerRequest.Id) == "" {
		writeJSONResponse(w, http.StatusBadRequest, "Peer ID is required")
		return
	}
	normalizedUrl, err := utils.NormalizeURL(registerRequest.Url)
	if err != nil {
		writeJSONResponse(w, http.StatusBadRequest, "Invalid URL")
		fmt.Println("[/registerPeer] Invalid URL:", registerRequest.Url)
		return
	}
	if peers.GetPeerById(registerRequest.Id) != nil {
		writeJSONResponse(w, http.StatusConflict, "Peer already registered")
		fmt.Println("[/registerPeer] Peer already registered with URL:", normalizedUrl)
		return
	}

	registeredTime := s.utilsStore.TimeNow()
	peers.RegisterPeer(peers.Peer{
		Url:       normalizedUrl,
		Id:        registerRequest.Id,
		FirstSeen: registeredTime,
		LastSeen:  registeredTime,
		Status:    "",
	})
	fmt.Println("[/registerPeer] Registered peer:", normalizedUrl)

	var status peers.PeerStatus
	isHealthy := s.peersStore.CheckPeerHealth(registerRequest.Id)
	if !isHealthy {
		status = peers.StatusUnhealthy
	} else {
		status = peers.StatusHealthy
	}

	registeredPeer := peers.UpdatePeerStatus(registerRequest.Id, status)

	writeJSONResponse(w, http.StatusOK, registeredPeer)
	fmt.Println("[/registerPeer] Successfully registered peer with URL:", normalizedUrl)
	s.peersStore.RelayNewPeer()
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

	for index, peerInfo := range registerRequest.Peers {
		if strings.TrimSpace(peerInfo.Id) == "" {
			writeJSONResponse(w, http.StatusBadRequest, "Peer ID is required")
			return
		}
		normalizedUrl, err := utils.NormalizeURL(peerInfo.Url)
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, "Invalid URL")
			fmt.Println("[/registerPeersReceived] Invalid URL:", peerInfo.Url)
			return
		}
		registerRequest.Peers[index].Url = normalizedUrl
	}

	for _, peerInfo := range registerRequest.Peers {
		if peers.GetPeerById(peerInfo.Id) == nil {
			registeredTime := s.utilsStore.TimeNow()
			peers.RegisterPeer(peers.Peer{
				Url:       peerInfo.Url,
				Id:        peerInfo.Id,
				FirstSeen: registeredTime,
				LastSeen:  registeredTime,
				Status:    "",
			})
			isHealthy := s.peersStore.CheckPeerHealth(peerInfo.Id)
			if isHealthy {
				peers.UpdatePeerStatus(peerInfo.Id, peers.StatusHealthy)
			} else {
				peers.UpdatePeerStatus(peerInfo.Id, peers.StatusUnhealthy)
			}
			fmt.Println("[/registerPeersReceived] Registered peer:", peerInfo.Url)
		}
	}

	writeJSONResponse(w, http.StatusOK, "Peers registration processed")
	fmt.Println("[/registerPeersReceived] Successfully processed peers registration")
}
