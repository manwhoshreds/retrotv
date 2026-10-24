package server

import (
	"net/http"
	"os"

	"golang.org/x/sys/unix"
)

func (s *Server) handleDashboardStorage(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	path := "."
	for _, source := range settings.MediaSources {
		if _, err := os.Stat(source.Path); err == nil {
			path = source.Path
			break
		}
	}
	var stats unix.Statfs_t
	if err := unix.Statfs(path, &stats); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	total := uint64(stats.Blocks) * uint64(stats.Bsize)
	available := uint64(stats.Bavail) * uint64(stats.Bsize)
	writeJSON(w, http.StatusOK, map[string]any{
		"path":        path,
		"total_bytes": total,
		"used_bytes":  total - available,
	})
}
