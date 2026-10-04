package peers

import (
	"fmt"
	"net/http"
	"pando/internal/utils"
	"sync"
)

type PeersTable struct {
	mu    sync.RWMutex
	Peers map[string]*Peer
}
type PeerStatus string

const (
	StatusHealthy   PeerStatus = "healthy"
	StatusUnhealthy PeerStatus = "unhealthy"
)

type Peer struct {
	Me        bool
	Url       string
	Id        string
	FirstSeen string
	LastSeen  string
	Status    PeerStatus
}

type RegisterPeerRequest struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

type RegisterPeersReceivedRequest struct {
	Peers []RegisterPeerRequest `json:"peers"`
}

var peersTable *PeersTable = initPeersTable()

type Store struct {
	client *http.Client
}

func NewStore(client *http.Client) *Store {
	return &Store{
		client: client,
	}
}

func initPeersTable() *PeersTable {
	return &PeersTable{
		Peers: make(map[string]*Peer),
	}

}

func RegisterPeer(peer Peer) *Peer {
	peersTable.mu.Lock()
	defer peersTable.mu.Unlock()
	peersTable.Peers[peer.Id] = &peer
	registeredPeer := *peersTable.Peers[peer.Id]
	return &registeredPeer
}

func GetAllPeers() []Peer {
	peers := []Peer{}
	peersTable.mu.RLock()
	defer peersTable.mu.RUnlock()
	for _, peer := range peersTable.Peers {
		peers = append(peers, *peer)
	}
	return peers
}

func GetPeerById(peerId string) *Peer {
	peersTable.mu.RLock()
	defer peersTable.mu.RUnlock()
	if peer, exists := peersTable.Peers[peerId]; exists {
		peerSnapshot := *peer
		return &peerSnapshot
	}
	return nil
}

func UpdatePeerStatus(peerId string, status PeerStatus) *Peer {
	peersTable.mu.Lock()
	defer peersTable.mu.Unlock()
	peer := peersTable.Peers[peerId]
	if peer != nil {
		peer.Status = status
		peerSnapshot := *peer
		return &peerSnapshot
	}
	return nil
}

func GetMe() *Peer {
	peersTable.mu.RLock()
	defer peersTable.mu.RUnlock()
	for _, peer := range peersTable.Peers {
		if peer.Me {
			peerSnapshot := *peer
			return &peerSnapshot
		}
	}
	return nil
}

func (s *Store) CheckPeerHealth(peerId string) bool {
	peer := GetPeerById(peerId)
	if peer == nil {
		return false
	}

	healthUrl := peer.Url + "/health"
	resp, err := s.client.Get(healthUrl)
	if err != nil {
		fmt.Println("Failed to check health for peer:", peerId, "error:", err)
		return false
	}
	defer resp.Body.Close()

	fmt.Println("Checked health for peer:", peerId, "status code:", resp.StatusCode)

	return resp.StatusCode == http.StatusOK
}

func (s *Store) RelayNewPeer() {
	endpoint := "/registerPeersReceived"
	peers := GetAllPeers()
	payload := RegisterPeersReceivedRequest{Peers: make([]RegisterPeerRequest, 0, len(peers))}
	for _, peer := range peers {
		payload.Peers = append(payload.Peers, RegisterPeerRequest{Id: peer.Id, Url: peer.Url})
	}
	for _, peer := range peers {

		url := peer.Url + endpoint
		resp, err := utils.Fetcher(s.client, url, "POST", map[string]string{}, payload)

		if err != nil {
			fmt.Printf("Failed to create request for %s: %v\n", peer.Url, err)
			continue
		}
		resp.Body.Close()

		fmt.Printf("Successfully relayed new peer to %s\n", peer.Url)
	}
}

func IsAllPeersActiveAndHealthy(numPeers int) bool {
	peers := GetAllPeers()

	if len(peers) != numPeers {
		return false
	}

	for _, peer := range peers {
		if peer.Status != StatusHealthy {
			return false
		}
	}
	return true
}
