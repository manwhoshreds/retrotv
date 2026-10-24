package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"retrotv/internal/config"
)

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var request config.Settings
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if len(request.MediaSources) == 0 {
		if request.MediaPath != "" {
			request.MediaSources = append(request.MediaSources, config.MediaSource{Path: request.MediaPath, ContentType: "movie"})
		}
		if request.TVMediaPath != "" {
			request.MediaSources = append(request.MediaSources, config.MediaSource{Path: request.TVMediaPath, ContentType: "tv"})
		}
	}
	if len(request.MediaSources) == 0 {
		writeError(w, http.StatusBadRequest, "at least one media source is required")
		return
	}
	for _, source := range request.MediaSources {
		if source.Path == "" {
			writeError(w, http.StatusBadRequest, "source path is required")
			return
		}
		if source.ContentType != "movie" && source.ContentType != "tv" {
			writeError(w, http.StatusBadRequest, "source content_type must be movie or tv")
			return
		}
		if err := requireDirectory(source.Path); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	for _, source := range request.AssetSources {
		if source.Path == "" {
			writeError(w, http.StatusBadRequest, "asset source path is required")
			return
		}
		if source.Scope != "" && source.Scope != config.AssetScopeGlobal {
			writeError(w, http.StatusBadRequest, "asset source scope must be global")
			return
		}
	}
	if request.MediaPath == "" {
		request.MediaPath = firstMediaSourcePath(request.MediaSources, "movie")
	}
	if request.TVMediaPath == "" {
		request.TVMediaPath = firstMediaSourcePath(request.MediaSources, "tv")
	}
	if request.MediaPath == "" {
		request.MediaPath = config.DefaultSettings().MediaPath
	}
	if request.TVMediaPath == "" {
		request.TVMediaPath = config.DefaultSettings().TVMediaPath
	}
	if request.StreamingMode != "" && request.StreamingMode != config.StreamingModeDirect && request.StreamingMode != config.StreamingModeHLS {
		writeError(w, http.StatusBadRequest, "streaming_mode must be direct or hls")
		return
	}
	request.StreamingMode = config.NormalizeStreamingMode(request.StreamingMode)
	if err := s.store.Save(request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	settings, err := s.store.Load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.db.MarkMoviesOutsideRootsMissing(r.Context(), sourcePaths(settings, "movie"), time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "reconcile removed movie sources: "+err.Error())
		return
	}
	if err := s.db.MarkTVShowsOutsideRootsMissing(r.Context(), sourcePaths(settings, "tv"), time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "reconcile removed TV sources: "+err.Error())
		return
	}

	s.mu.Lock()
	s.settings = settings
	s.mu.Unlock()
	if settings.StreamingMode == config.StreamingModeDirect && s.sessions != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.sessions.StopActiveSessions(stopCtx)
	}

	log.Printf("event=settings_saved streaming_mode=%q media_sources=%d asset_sources=%d", settings.StreamingMode, len(settings.MediaSources), len(settings.AssetSources))
	writeJSON(w, http.StatusOK, settings)
}
