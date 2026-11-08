package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pando/internal/peers"
	"pando/internal/service"
	"pando/internal/status"
	"pando/internal/utils"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type RegisterPeerRequest = peers.RegisterPeerRequest

type RegisterPeersReceivedRequest = peers.RegisterPeersReceivedRequest

type storageResponse struct {
	Path       string `json:"path"`
	TotalBytes uint64 `json:"totalBytes"`
	FreeBytes  uint64 `json:"freeBytes"`
	UsedBytes  uint64 `json:"usedBytes"`
}

type clusterNodeStatus struct {
	ID          string           `json:"id"`
	URL         string           `json:"url"`
	Me          bool             `json:"me"`
	Reachable   bool             `json:"reachable"`
	Healthy     bool             `json:"healthy"`
	Storage     *storageResponse `json:"storage,omitempty"`
	PhotoCount  int              `json:"photoCount"`
	LastSeen    string           `json:"lastSeen"`
	Errors      []string         `json:"errors,omitempty"`
	Status      json.RawMessage  `json:"status,omitempty"`
	StatusError string           `json:"statusError,omitempty"`
}

type clusterStatusResponse struct {
	Nodes []clusterNodeStatus `json:"nodes"`
}

func writeJSONResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type statusSettings struct {
	Replicas              int    `json:"replicas"`
	ExpectedPeers         int    `json:"expectedPeers"`
	ListenPort            string `json:"listenPort"`
	ImmichURL             string `json:"immichUrl"`
	MinFreeBytes          uint64 `json:"minFreeBytes,omitempty"`
	StorageRecheckSeconds int    `json:"storageRecheckSeconds,omitempty"`
}

type statusResponse struct {
	State                  status.State   `json:"state"`
	StateSince             string         `json:"stateSince"`
	LastScanStarted        *string        `json:"lastScanStarted"`
	LastScanCompleted      *string        `json:"lastScanCompleted"`
	LastSuccessfulDownload *string        `json:"lastSuccessfulDownload"`
	LastError              *statusError   `json:"lastError"`
	StartedAt              string         `json:"startedAt"`
	UptimeSeconds          int64          `json:"uptimeSeconds"`
	Version                string         `json:"version"`
	Settings               statusSettings `json:"settings"`
}

type statusError struct {
	Message string `json:"message"`
	At      string `json:"at"`
	Source  string `json:"source"`
}

func formatStatusTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func (s *Server) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if s.status == nil {
		writeJSONResponse(w, http.StatusInternalServerError, "Status tracker is not configured")
		return
	}
	snapshot := s.status.Snapshot()
	response := statusResponse{
		State:                  snapshot.State,
		StateSince:             snapshot.StateSince.UTC().Format(time.RFC3339),
		LastScanStarted:        formatStatusTime(snapshot.LastScanStarted),
		LastScanCompleted:      formatStatusTime(snapshot.LastScanCompleted),
		LastSuccessfulDownload: formatStatusTime(snapshot.LastSuccessfulDownload),
		StartedAt:              snapshot.StartedAt.UTC().Format(time.RFC3339),
		UptimeSeconds:          snapshot.UptimeSeconds,
		Version:                s.version,
		Settings: statusSettings{
			Replicas:      s.envVariables.NumReplicas,
			ExpectedPeers: s.envVariables.NumPeers,
			ListenPort:    s.envVariables.ListenPort,
			ImmichURL:     safeURLOrigin(s.envVariables.ImmichUrl),
		},
	}
	if snapshot.LastError != nil {
		response.LastError = &statusError{
			Message: snapshot.LastError.Message,
			At:      snapshot.LastError.At.UTC().Format(time.RFC3339),
			Source:  snapshot.LastError.Source,
		}
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func safeURLOrigin(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func (s *Server) StorageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(filepath.Clean(s.envVariables.BackupFolderPath), &stat); err != nil {
		fmt.Printf("[/storage] Failed to inspect storage: %v\n", err)
		writeJSONResponse(w, http.StatusInternalServerError, "Failed to inspect storage")
		return
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	writeJSONResponse(w, http.StatusOK, storageResponse{
		Path: s.envVariables.BackupFolderPath, TotalBytes: total, FreeBytes: free, UsedBytes: used,
	})
}

func (s *Server) ClusterStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	s.statusMu.Lock()
	if time.Since(s.statusCached) < 10*time.Second && s.statusCache != nil {
		data := append([]byte(nil), s.statusCache...)
		s.statusMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
		return
	}
	s.statusMu.Unlock()

	nodes := peers.GetAllPeers()
	if me := peers.GetMe(); me != nil {
		found := false
		for _, node := range nodes {
			if node.Id == me.Id {
				found = true
				break
			}
		}
		if !found {
			nodes = append(nodes, *me)
		}
	}
	results := make([]clusterNodeStatus, len(nodes))
	var wg sync.WaitGroup
	for i, peer := range nodes {
		wg.Add(1)
		go func(i int, peer peers.Peer) {
			defer wg.Done()
			results[i] = s.collectNodeStatus(peer)
		}(i, peer)
	}
	wg.Wait()
	data, err := json.Marshal(clusterStatusResponse{Nodes: results})
	if err != nil {
		fmt.Printf("[/cluster/status] Failed to encode status: %v\n", err)
		writeJSONResponse(w, http.StatusInternalServerError, "Failed to encode cluster status")
		return
	}
	s.statusMu.Lock()
	s.statusCache = append([]byte(nil), data...)
	s.statusCached = time.Now()
	s.statusMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) collectNodeStatus(peer peers.Peer) clusterNodeStatus {
	result := clusterNodeStatus{ID: peer.Id, URL: peer.Url, Me: peer.Me, LastSeen: peer.LastSeen}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request := func(path string, target any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, peer.Url+path, nil)
		if err != nil {
			return err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("%s returned %s", path, resp.Status)
		}
		return json.NewDecoder(resp.Body).Decode(target)
	}
	var health string
	if err := request("/health", &health); err != nil {
		result.Errors = append(result.Errors, "health: "+err.Error())
	} else {
		result.Reachable, result.Healthy = true, true
	}
	var storage storageResponse
	if err := request("/storage", &storage); err != nil {
		result.Errors = append(result.Errors, "storage: "+err.Error())
	} else {
		result.Reachable = true
		result.Storage = &storage
	}
	var manifest struct {
		Total int `json:"total"`
	}
	if err := request("/manifest?limit=1", &manifest); err != nil {
		result.Errors = append(result.Errors, "manifest: "+err.Error())
	} else {
		result.Reachable, result.PhotoCount = true, manifest.Total
	}
	var nodeStatus json.RawMessage
	if err := request("/status", &nodeStatus); err != nil {
		result.StatusError = err.Error()
	} else {
		result.Status = nodeStatus
	}
	return result
}

func (s *Server) ManifestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query()
	limit := service.DefaultManifestLimit
	if query.Has("limit") {
		parsedLimit, err := strconv.Atoi(query.Get("limit"))
		if err != nil || parsedLimit < 1 || parsedLimit > service.MaxManifestLimit {
			writeJSONResponse(w, http.StatusBadRequest, fmt.Sprintf("limit must be an integer between 1 and %d", service.MaxManifestLimit))
			return
		}
		limit = parsedLimit
	}

	page, err := service.GetAssetManifestPage(s.envVariables.BackupFolderPath, limit, query.Get("after"))
	if err != nil {
		fmt.Printf("[/manifest] Failed to read asset manifest: %v\n", err)
		writeJSONResponse(w, http.StatusInternalServerError, "Failed to read asset manifest")
		return
	}
	writeJSONResponse(w, http.StatusOK, page)
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

func (s *Server) DoesThisNodeHaveImage(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("assetId")
	fileName := r.URL.Query().Get("fileName")
	var exists bool
	if assetID != "" {
		exists = service.FileIsSavedFromAssetID(assetID, s.envVariables.BackupFolderPath)
	} else {
		exists, assetID = service.FileIsSavedFromOriginalFileName(fileName, s.envVariables.BackupFolderPath)
	}
	fmt.Printf("[/doesThisNodeHaveImage] Checking image, exists: %v, assetId: %s\n", exists, assetID)
	if exists {
		writeJSONResponse(w, http.StatusOK, map[string]string{"assetId": assetID})
	} else {
		writeJSONResponse(w, http.StatusNotFound, "Image not found")
	}

}

func (s *Server) WhichNodeHasThisImage(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("fileName")
	assetId := r.URL.Query().Get("assetId")

	resolvedAssetId := assetId
	if fileName != "" {
		var err error
		resolvedAssetId, err = s.immichStore.ResolveFileNameToAssetId(fileName)
		if err != nil {
			fmt.Printf("[/whichNodeHasThisImage] Error resolving file name to asset ID: %v\n", err)
			writeJSONResponse(w, http.StatusInternalServerError, "Error resolving file name to asset ID")
			return
		}
	}

	storingPeerIds := service.WhichPeersShouldStore(resolvedAssetId, peers.GetPeerIds(), s.envVariables.NumReplicas)

	type storingPeer struct {
		peers.Peer
		HasImage bool
	}

	storingPeers := make([]storingPeer, 0, len(storingPeerIds))
	for _, peerId := range storingPeerIds {
		peer := peers.GetPeerById(peerId)
		if peer != nil {
			peerHasImage := false
			if peer.Me {
				peerHasImage = service.FileIsSavedFromAssetID(resolvedAssetId, s.envVariables.BackupFolderPath)
			} else {
				peerHasImage = s.peerHasImage(peer.Url, resolvedAssetId)
			}
			storingPeers = append(storingPeers, storingPeer{
				Peer:     *peer,
				HasImage: peerHasImage,
			})
		}
	}

	response := map[string]interface{}{
		"assetId":      resolvedAssetId,
		"storingPeers": storingPeers,
	}
	writeJSONResponse(w, http.StatusOK, response)

}

func (s *Server) peerHasImage(peerURL, assetID string) bool {
	requestURL := peerURL + "/doesThisNodeHaveImage?assetId=" + url.QueryEscape(assetID)
	resp, err := s.client.Get(requestURL)
	if err != nil {
		fmt.Printf("Failed to query peer %s for asset %s: %v\n", peerURL, assetID, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
