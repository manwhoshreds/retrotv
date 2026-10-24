package server

import (
	"net/http"
	"strings"

	assetscanner "retrotv/internal/assets"
	"retrotv/internal/models"
)

func (s *Server) handleGetAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := s.db.ChannelAssets(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"assets": assets,
		"count":  len(assets),
	})
}

func (s *Server) handleAssetScan(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	sourcePaths := make([]string, 0, len(settings.AssetSources))
	for _, source := range settings.AssetSources {
		sourcePaths = append(sourcePaths, source.Path)
	}
	s.beginScan("assets", "Preparing asset scan", strings.Join(sourcePaths, ", "))
	defer s.finishScan("Asset scan complete")

	s.updateScan(scanStatus{
		Phase:       "Finding channel assets",
		Source:      strings.Join(sourcePaths, ", "),
		Current:     "Discovering files",
		CurrentItem: -1,
		TotalItems:  -1,
	})

	channels, err := s.db.Channels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scanner := assetscanner.Scanner{Channels: channels}
	discovered, scanErrors := scanner.Scan(r.Context(), settings.AssetSources)
	saved := []models.ChannelAsset{}
	for index, asset := range discovered {
		s.updateScan(scanStatus{
			Phase:       "Importing channel assets",
			Source:      asset.FilePath,
			Current:     asset.Filename,
			CurrentItem: index + 1,
			TotalItems:  len(discovered),
		})
		record, err := s.db.UpsertChannelAsset(r.Context(), asset)
		if err != nil {
			scanErrors = append(scanErrors, asset.FilePath+": "+err.Error())
			s.updateScanErrors(scanErrors)
			continue
		}
		saved = append(saved, record)
	}
	s.updateScanErrors(scanErrors)

	s.updateScan(scanStatus{Phase: "Generating channel schedules", Current: "Refreshing lineups"})
	if err := s.scheduler.RegenerateAll(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sources":       sourcePaths,
		"asset_sources": settings.AssetSources,
		"count":         len(saved),
		"scanned":       len(discovered),
		"assets":        saved,
		"errors":        scanErrors,
	})
}
