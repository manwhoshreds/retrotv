package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *Server) handleMovieStream(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	movie, err := s.db.MovieByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	path := absoluteMediaPath(movie.FilePath)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, path+" is not available")
		return
	}
	log.Printf("event=media_stream media_type=%q id=%d path=%q range=%q", "movie", id, path, r.Header.Get("Range"))
	http.ServeFile(w, r, path)
}

func (s *Server) handleTVShowStream(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	show, err := s.db.TVShowByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	path := r.URL.Query().Get("episode_path")
	if path != "" {
		resolved, ok := resolveTVEpisodePath(show, path)
		if !ok {
			writeError(w, http.StatusNotFound, "TV episode is not available for this show")
			return
		}
		path = resolved
	} else {
		path = firstTVEpisodePath(show)
	}
	if path == "" {
		writeError(w, http.StatusNotFound, "TV show has no scanned episode files")
		return
	}
	path = absoluteMediaPath(path)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, path+" is not available")
		return
	}
	log.Printf("event=media_stream media_type=%q id=%d path=%q range=%q", "tv", id, path, r.Header.Get("Range"))
	http.ServeFile(w, r, path)
}

func (s *Server) handleAssetStream(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	asset, err := s.db.ChannelAsset(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	path := absoluteMediaPath(asset.FilePath)
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, path+" is not available")
		return
	}
	log.Printf("event=media_stream media_type=%q id=%d path=%q range=%q", "asset", id, path, r.Header.Get("Range"))
	http.ServeFile(w, r, path)
}

func (s *Server) handleHLSPlaylist(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if s.streamingMode() != streamingModeHLS {
		writeError(w, http.StatusNotFound, "HLS streaming is disabled")
		return
	}
	if s.sessions != nil {
		s.sessions.Touch(id)
	}
	path := filepath.Join(s.streamDir, "channels", strconv.FormatInt(id, 10), "index.m3u8")
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	http.ServeFile(w, r, path)
}

func (s *Server) handleHLSSegment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if s.streamingMode() != streamingModeHLS {
		writeError(w, http.StatusNotFound, "HLS streaming is disabled")
		return
	}
	if s.sessions != nil {
		s.sessions.Touch(id)
	}
	segment := filepath.Base(r.PathValue("segment"))
	if segment == "." || segment == "" || strings.Contains(segment, "/") || !strings.HasSuffix(segment, ".ts") {
		writeError(w, http.StatusNotFound, "segment is not available")
		return
	}
	path := filepath.Join(s.streamDir, "channels", strconv.FormatInt(id, 10), segment)
	w.Header().Set("Content-Type", "video/mp2t")
	http.ServeFile(w, r, path)
}
