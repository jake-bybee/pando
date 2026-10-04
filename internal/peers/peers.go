package peers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type PeersTable struct {
	mu    sync.RWMutex
	Peers map[string]*Peer
}
type Peer struct {
	Me        bool
	Url       string
	Id        string
	FirstSeen string
	LastSeen  string
	Status    string
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

func UpdatePeerStatus(peerId string, status string) *Peer {
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
	for _, peer := range GetAllPeers() {
		url := peer.Url + endpoint
		data, err := json.Marshal(url)

		if err != nil {
			fmt.Printf("Failed to create request for %s: %v\n", peer.Url, err)
			continue
		}
		req, err := http.NewRequest("POST", peer.Url, bytes.NewReader(data))
		if err != nil {
			fmt.Printf("Failed to create request for %s: %v\n", peer.Url, err)
			continue
		}
		_, err = s.client.Do(req)
		if err != nil {
			fmt.Printf("Failed to relay to %s: %v\n", peer.Url, err)
			continue
		}

		fmt.Printf("Successfully relayed new peer to %s\n", peer.Url)
	}
}
