package peers

import (
	"fmt"
	"net/http"
)

type PeersTable struct {
	Peers map[string]*Peer
}
type Peer struct {
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
	peersTable.Peers[peer.Id] = &peer
	return peersTable.Peers[peer.Id]
}

func GetAllPeers() []Peer {
	peers := []Peer{}
	for _, peer := range peersTable.Peers {
		peers = append(peers, *peer)
	}
	return peers
}

func GetPeerById(peerId string) *Peer {
	if peer, exists := peersTable.Peers[peerId]; exists {
		return peer
	}
	return nil
}

func UpdatePeerStatus(peerId string, status string) *Peer {
	peer := GetPeerById(peerId)
	if peer != nil {
		peer.Status = status
		return peer
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
