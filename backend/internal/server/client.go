package server

import (
	"net/http"
	"time"
)

const serverVersion = "0.1.0"

type clientStartupResponse struct {
	Status        string    `json:"status"`
	ServerVersion string    `json:"server_version"`
	ServerTime    time.Time `json:"server_time"`
	StreamingMode string    `json:"streaming_mode"`
	ChannelCount  int       `json:"channel_count"`
	ScanActive    bool      `json:"scan_active"`
	ScanPhase     string    `json:"scan_phase,omitempty"`
}

func (s *Server) handleClientStartup(w http.ResponseWriter, r *http.Request) {
	channels, err := s.db.Channels(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}

	scan := s.currentScanStatus()
	channelCount := len(airingChannels(channels))
	writeJSON(w, http.StatusOK, clientStartupResponse{
		Status:        clientStartupStatus(channelCount, scan),
		ServerVersion: serverVersion,
		ServerTime:    time.Now().UTC(),
		StreamingMode: s.streamingMode(),
		ChannelCount:  channelCount,
		ScanActive:    scan.Active,
		ScanPhase:     scan.Phase,
	})
}

func clientStartupStatus(channelCount int, scan scanStatus) string {
	if scan.Active {
		return "schedule_generating"
	}
	if channelCount == 0 {
		return "no_channels"
	}
	return "ready"
}
