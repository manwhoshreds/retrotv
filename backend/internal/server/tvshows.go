package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"retrotv/internal/metadata"
	"retrotv/internal/models"
)

func (s *Server) handleGetTVShows(w http.ResponseWriter, r *http.Request) {
	shows, err := s.db.TVShows(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tv_shows": shows,
		"count":    len(shows),
	})
}

func (s *Server) handleDeleteTVShow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.db.DeleteTVShow(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTVShowMetadataMatches(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	show, err := s.db.TVShowByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	downloader := metadata.NewMetadataDownloader(os.Getenv("TMDB_READ_TOKEN"))
	title := show.LocalTitle
	if title == "" {
		title = show.Name
	}
	year := show.LocalYear
	if year == 0 {
		year = show.Year
	}
	if query := r.URL.Query().Get("query"); query != "" {
		title = query
		year = 0
	}
	matches, err := downloader.SearchTVShows(r.Context(), title, year)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"matches": matches,
		"count":   len(matches),
	})
}

func (s *Server) handleRefreshTVShowMetadata(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var request struct {
		TMDBID int `json:"tmdb_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if request.TMDBID <= 0 {
		writeError(w, http.StatusBadRequest, "tmdb_id is required")
		return
	}

	existing, err := s.db.TVShowByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	downloader := metadata.NewMetadataDownloader(os.Getenv("TMDB_READ_TOKEN"))
	show, err := downloader.GetTVShowByID(r.Context(), request.TMDBID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	show.MetadataVersion = currentTVShowMetadataVersion
	posterLocalPath, backdropLocalPath, err := downloader.DownloadTVArtwork(r.Context(), show, s.artworkDir)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	episodes, episodeErrors := downloader.DownloadTVEpisodeThumbnailsWithCache(r.Context(), show, existing.EpisodePaths, filepath.Join(s.artworkDir, "tvshows"), existing.EpisodeMetadata)
	if len(episodeErrors) > 0 {
		writeError(w, http.StatusBadGateway, strings.Join(episodeErrors, "; "))
		return
	}
	episodes = probeEpisodeMediaMetadata(r.Context(), metadata.NewFFprobeProber(), episodes, existing.EpisodeMetadata)

	refreshed, err := s.db.UpsertTVShow(r.Context(), models.TVShowRecord{
		TVShow:            show,
		Path:              existing.Path,
		LocalTitle:        existing.LocalTitle,
		LocalYear:         existing.LocalYear,
		LocalEpisodeCount: existing.LocalEpisodeCount,
		EpisodePaths:      existing.EpisodePaths,
		EpisodeMetadata:   episodes,
		PosterLocalPath:   posterLocalPath,
		BackdropLocalPath: backdropLocalPath,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, refreshed)
}
