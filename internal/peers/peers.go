package peers

import (
	"net/http"
	"pando/internal/utils"
)

type PeersTable struct {
	Peers map[string]Peer
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
	utils  *utils.Store
	client *http.Client
}

func initPeersTable() *PeersTable {
	return &PeersTable{
		Peers: make(map[string]Peer),
	}
}

func RegisterPeer(peer Peer) Peer {
	peersTable.Peers[peer.Id] = peer
	return peersTable.Peers[peer.Id]
}

func GetAllPeers() []Peer {
	peers := []Peer{}
	for _, peer := range peersTable.Peers {
		peers = append(peers, peer)
	}
	return peers
}

func GetPeerById(peerId string) *Peer {
	if peer, exists := peersTable.Peers[peerId]; exists {
		return &peer
	}
	return nil
}

func UpdatePeerStatus(peerId string, status string) {
	GetPeerById(peerId).Status = status
}

func (s *Store) CheckPeerHealth(peerId string) bool {
	peer := GetPeerById(peerId)
	if peer == nil {
		return false
	}

	healthUrl := peer.Url + "/health"
	resp, err := s.client.Get(healthUrl)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
