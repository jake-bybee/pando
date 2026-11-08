package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) RegisterRoutes() {
	http.HandleFunc("/manifest", s.ManifestHandler)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		s.HealthHandler(w, r)
	})

	http.HandleFunc("/registerPeer", func(w http.ResponseWriter, r *http.Request) {
		s.RegisterPeerHandler(w, r)
	})

	http.HandleFunc("/relayPeers", func(w http.ResponseWriter, r *http.Request) {
		s.RelayPeersHandler(w, r)
	})

	http.HandleFunc("/registerPeersReceived", func(w http.ResponseWriter, r *http.Request) {
		s.RegisterPeersReceivedHandler(w, r)
	})

	http.HandleFunc("/doesThisNodeHaveImage", func(w http.ResponseWriter, r *http.Request) {
		s.DoesThisNodeHaveImage(w, r)
	})

	http.HandleFunc("/whichNodeHasThisImage", func(w http.ResponseWriter, r *http.Request) {
		s.WhichNodeHasThisImage(w, r)
	})

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		data, err := s.immichStore.BulkDownload([]string{

			"00012798-03fa-478a-a79e-e54641afdfd6",
			"000212d9-2eb7-41e9-907b-dc819d139e9b",
			"0002ef04-6445-4440-9c65-61fdc18c352d",
			"000590ed-3f9c-40ca-a255-94dc56b9ec7f",
			"00063a38-0e4a-45ba-bf17-a5cfec3261eb",
			"00082c7a-345d-44aa-b443-52b29feea35e",
			"0008477c-2c48-40c2-9946-ee9057e31b78",
			"0008cf23-2a90-4302-847f-621f8ea8fc61",
			"001071c0-1f0b-4b05-b872-3b88851d6cb1",
			"00112712-8872-4a69-b0ae-31e409f2437c",
			"0011411f-f8d8-4404-9b6e-ce3905bff16a",
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get server statistics: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode server statistics: %v", err), http.StatusInternalServerError)
			return
		}

	})
}
