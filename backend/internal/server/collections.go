package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"retrotv/internal/models"
)

func (s *Server) handleUploadCollectionArtwork(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	file, header, err := r.FormFile("artwork")
	if err != nil {
		writeError(w, http.StatusBadRequest, "artwork image is required")
		return
	}
	defer file.Close()
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ext, ok := logoExtension(http.DetectContentType(buffer[:n]), filepath.Ext(header.Filename))
	if !ok {
		writeError(w, http.StatusBadRequest, "artwork must be a PNG, JPEG, WEBP, or GIF image")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	directory := filepath.Join(s.artworkDir, "collections", "uploads")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := sanitizeFilename(strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename)))
	if name == "" {
		name = "collection-artwork"
	}
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), name, ext)
	destination := filepath.Join(directory, filename)
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := io.Copy(output, file); err != nil {
		_ = output.Close()
		_ = os.Remove(destination)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := output.Close(); err != nil {
		_ = os.Remove(destination)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"artwork_path": "/artwork/collections/uploads/" + filename})
}

func (s *Server) handleGetCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := s.db.Collections(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"collections": collections, "count": len(collections)})
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	collection, ok := decodeCollection(w, r)
	if !ok {
		return
	}
	saved, err := s.db.SaveCollection(r.Context(), collection)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if saved.ArtworkMode == "auto" && len(saved.ArtworkMedia) == 2 {
		path, generateErr := s.generateCollectionArtwork(r.Context(), saved)
		if generateErr != nil {
			_ = s.db.DeleteCollection(r.Context(), saved.ID)
			writeError(w, http.StatusBadRequest, generateErr.Error())
			return
		}
		saved.ArtworkPath = path
		saved, err = s.db.SaveCollection(r.Context(), saved)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) handleUpdateCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	collection, ok := decodeCollection(w, r)
	if !ok {
		return
	}
	collection.ID = id
	if collection.ArtworkMode == "auto" && len(collection.ArtworkMedia) == 2 {
		path, err := s.generateCollectionArtwork(r.Context(), collection)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		collection.ArtworkPath = path
	}
	saved, err := s.db.SaveCollection(r.Context(), collection)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.db.DeleteCollection(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePreviewCollection(w http.ResponseWriter, r *http.Request) {
	collection, ok := decodeCollection(w, r)
	if !ok {
		return
	}
	if len(collection.Rules) == 0 {
		pinned := make(map[models.CollectionMedia]struct{}, len(collection.PinnedMedia))
		for _, media := range collection.PinnedMedia {
			if media.MediaID > 0 && (media.MediaType == "movie" || media.MediaType == "tv") {
				pinned[media] = struct{}{}
			}
		}
		for _, media := range collection.ExcludedMedia {
			delete(pinned, media)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": []collectionPreviewItem{}, "count": len(pinned)})
		return
	}
	movies, err := s.db.MatchingMovies(r.Context(), collection.Rules)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	shows, err := s.db.MatchingTVShows(r.Context(), collection.Rules)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type mediaIdentity struct {
		mediaType string
		mediaID   int64
	}
	identities := make(map[mediaIdentity]struct{}, len(movies)+len(shows)+len(collection.PinnedMedia))
	for _, movie := range movies {
		identities[mediaIdentity{mediaType: "movie", mediaID: movie.ID}] = struct{}{}
	}
	for _, show := range shows {
		identities[mediaIdentity{mediaType: "tv", mediaID: show.ID}] = struct{}{}
	}
	for _, pinned := range collection.PinnedMedia {
		if pinned.MediaID > 0 && (pinned.MediaType == "movie" || pinned.MediaType == "tv") {
			identities[mediaIdentity{mediaType: pinned.MediaType, mediaID: pinned.MediaID}] = struct{}{}
		}
	}
	for _, excluded := range collection.ExcludedMedia {
		delete(identities, mediaIdentity{mediaType: excluded.MediaType, mediaID: excluded.MediaID})
	}
	items := make([]collectionPreviewItem, 0, len(movies)+len(shows))
	baseURL := requestBaseURL(r)
	for _, movie := range movies {
		if _, included := identities[mediaIdentity{mediaType: "movie", mediaID: movie.ID}]; !included {
			continue
		}
		items = append(items, collectionPreviewItem{MediaType: "movie", MediaID: movie.ID, Title: programTitle(models.ScheduledProgram{MediaType: "movie", Media: movie}), PosterURL: publicURL(baseURL, artworkURLPath(movie.PosterLocalPath))})
	}
	for _, show := range shows {
		if _, included := identities[mediaIdentity{mediaType: "tv", mediaID: show.ID}]; !included {
			continue
		}
		items = append(items, collectionPreviewItem{MediaType: "tv", MediaID: show.ID, Title: tvShowDisplayTitle(show), PosterURL: publicURL(baseURL, artworkURLPath(show.PosterLocalPath))})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(identities)})
}

type collectionPreviewItem struct {
	MediaType string `json:"media_type"`
	MediaID   int64  `json:"media_id"`
	Title     string `json:"title"`
	PosterURL string `json:"poster_url"`
}

func decodeCollection(w http.ResponseWriter, r *http.Request) (models.Collection, bool) {
	var collection models.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return models.Collection{}, false
	}
	collection.Name = strings.TrimSpace(collection.Name)
	collection.ArtworkPath = strings.TrimSpace(collection.ArtworkPath)
	collection.ArtworkMode = strings.TrimSpace(collection.ArtworkMode)
	for index := range collection.Rules {
		collection.Rules[index].MediaType = strings.TrimSpace(collection.Rules[index].MediaType)
		if collection.Rules[index].MediaType != "movie" && collection.Rules[index].MediaType != "tv" {
			writeError(w, http.StatusBadRequest, "collection rules must explicitly target movies or TV shows")
			return models.Collection{}, false
		}
	}
	if collection.Name == "" && r.URL.Path != "/api/collections/preview" {
		writeError(w, http.StatusBadRequest, "collection name is required")
		return models.Collection{}, false
	}
	if collection.ArtworkPath != "" && !strings.HasPrefix(collection.ArtworkPath, "/artwork/collections/") {
		writeError(w, http.StatusBadRequest, "invalid collection artwork path")
		return models.Collection{}, false
	}
	if collection.ArtworkMode == "" {
		if collection.ArtworkPath != "" {
			collection.ArtworkMode = "custom"
		} else {
			collection.ArtworkMode = "auto"
		}
	}
	if collection.ArtworkMode != "auto" && collection.ArtworkMode != "custom" {
		writeError(w, http.StatusBadRequest, "invalid collection artwork mode")
		return models.Collection{}, false
	}
	if collection.ArtworkMode == "auto" && len(collection.ArtworkMedia) != 2 {
		collection.ArtworkPath = ""
	}
	return collection, true
}
