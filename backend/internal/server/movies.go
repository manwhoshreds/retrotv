package server

import (
	"encoding/json"
	"net/http"
	"os"

	"retrotv/internal/metadata"
	"retrotv/internal/models"
)

func (s *Server) handleGetMovies(w http.ResponseWriter, r *http.Request) {
	movies, err := s.db.Movies(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.mu.RLock()
	if len(movies) == 0 {
		movies = s.movies
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"movies": movies,
		"count":  len(movies),
	})
}

func (s *Server) handleDeleteMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.db.DeleteMovie(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.scheduler.GenerateAll(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMovieMetadataMatches(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	movie, err := s.db.MovieByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	downloader := metadata.NewMetadataDownloader(os.Getenv("TMDB_READ_TOKEN"))
	title := movie.LocalTitle
	if title == "" {
		title = movie.Title
	}
	year := movie.LocalYear
	if year == 0 {
		year = movie.Year
	}
	if query := r.URL.Query().Get("query"); query != "" {
		title = query
		year = 0
	}
	matches, err := downloader.SearchMovies(r.Context(), title, year)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"matches": matches,
		"count":   len(matches),
	})
}

func (s *Server) handleRefreshMovieMetadata(w http.ResponseWriter, r *http.Request) {
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

	existing, err := s.db.MovieByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	downloader := metadata.NewMetadataDownloader(os.Getenv("TMDB_READ_TOKEN"))
	movie, err := downloader.GetMovieByID(r.Context(), request.TMDBID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	posterLocalPath, backdropLocalPath, err := downloader.DownloadArtwork(r.Context(), movie, s.artworkDir)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	refreshed, err := s.db.UpsertMovie(r.Context(), models.MovieRecord{
		Movie:             movie,
		FilePath:          existing.FilePath,
		Filename:          existing.Filename,
		LocalTitle:        existing.LocalTitle,
		LocalYear:         existing.LocalYear,
		MediaMetadata:     existing.MediaMetadata,
		PosterLocalPath:   posterLocalPath,
		BackdropLocalPath: backdropLocalPath,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, refreshed)
}
