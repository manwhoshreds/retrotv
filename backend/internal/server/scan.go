package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"retrotv/internal/metadata"
	"retrotv/internal/models"
)

func (s *Server) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.currentScanStatus())
}

func (s *Server) handleMovieScan(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MediaPath     string `json:"media_path"`
		ForceMetadata bool   `json:"force_metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	mediaDirs := sourcePaths(settings, "movie")
	if request.MediaPath != "" {
		mediaDirs = []string{request.MediaPath}
	}
	s.beginScan("movie", "Preparing movie scan", strings.Join(mediaDirs, ", "))
	defer s.finishScan("Movie scan complete")

	allMovies := []models.MovieFile{}
	successfulRoots := []string{}
	cached, err := s.db.Movies(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cachedByPath := movieRecordsByPath(cached)
	fingerprints := make(map[string]models.MediaMetadata, len(cached))
	for _, record := range cached {
		fingerprints[record.FilePath] = record.MediaMetadata
	}
	scanErrors := []string{}
	for _, mediaDir := range mediaDirs {
		s.updateScan(scanStatus{
			Phase:       "Finding movie files",
			Source:      mediaDir,
			Current:     "Discovering files",
			CurrentItem: -1,
			TotalItems:  -1,
		})
		movies, err := metadata.ScanMoviesWithIndex(mediaDir, fingerprints, func(path string, discovered int) {
			s.updateScan(scanStatus{
				Phase:       "Finding movie files",
				Source:      mediaDir,
				Current:     path,
				CurrentItem: len(allMovies) + discovered,
			})
		})
		if err != nil {
			scanErrors = append(scanErrors, mediaDir+": "+err.Error())
			s.updateScanErrors(scanErrors)
			continue
		}
		successfulRoots = append(successfulRoots, mediaDir)
		allMovies = append(allMovies, movies...)
	}
	cached, consolidationErrors := s.consolidateMovieDuplicates(r.Context(), allMovies, cached)
	scanErrors = append(scanErrors, consolidationErrors...)
	cached, relocationErrors := s.relocateMovedMovies(r.Context(), allMovies, cached)
	scanErrors = append(scanErrors, relocationErrors...)
	cachedByPath = movieRecordsByPath(cached)
	counters := classifyMovies(allMovies, cachedByPath, successfulRoots, request.ForceMetadata)
	allRecords, importErrors := s.importMoviesWithCache(r.Context(), allMovies, cachedByPath, request.ForceMetadata)
	scanErrors = append(scanErrors, importErrors...)
	discoveredMoviePaths := make([]string, 0, len(allMovies))
	for _, movie := range allMovies {
		discoveredMoviePaths = append(discoveredMoviePaths, movie.Path)
	}
	if err := s.db.ReconcileMoviePresence(r.Context(), successfulRoots, discoveredMoviePaths, time.Now()); err != nil {
		scanErrors = append(scanErrors, "reconcile movie library: "+err.Error())
	} else if records, err := s.db.Movies(r.Context()); err != nil {
		scanErrors = append(scanErrors, "reload reconciled movie library: "+err.Error())
	} else {
		allRecords = records
	}
	s.updateScanErrors(scanErrors)
	s.mu.Lock()
	s.movies = allRecords
	s.mu.Unlock()
	if err := s.scheduler.GenerateAll(r.Context()); err != nil {
		scanErrors = append(scanErrors, "regenerate schedules: "+err.Error())
		s.updateScanErrors(scanErrors)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"media_path": strings.Join(mediaDirs, ", "),
		"sources":    mediaDirs,
		"count":      len(allRecords),
		"scanned":    len(allMovies),
		"discovered": counters.Discovered,
		"unchanged":  counters.Unchanged,
		"new":        counters.New,
		"changed":    counters.Changed,
		"missing":    counters.Missing,
		"movies":     allRecords,
		"errors":     scanErrors,
	})
}

func (s *Server) handleTVShowScan(w http.ResponseWriter, r *http.Request) {
	var request struct {
		TVMediaPath   string `json:"tv_media_path"`
		ForceMetadata bool   `json:"force_metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	s.mu.RLock()
	settings := s.settings
	s.mu.RUnlock()

	tvDirs := sourcePaths(settings, "tv")
	if request.TVMediaPath != "" {
		tvDirs = []string{request.TVMediaPath}
	}
	s.beginScan("tv", "Preparing TV scan", strings.Join(tvDirs, ", "))
	defer s.finishScan("TV scan complete")

	allShows := []models.TVShowFile{}
	successfulRoots := []string{}
	cached, err := s.db.TVShows(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cachedByPath := tvRecordsByPath(cached)
	fingerprints := map[string]models.MediaMetadata{}
	for _, show := range cached {
		for _, episode := range show.EpisodeMetadata {
			fingerprints[episode.Path] = episode.MediaMetadata
		}
	}
	scanErrors := []string{}
	for _, tvDir := range tvDirs {
		s.updateScan(scanStatus{
			Phase:       "Finding TV show files",
			Source:      tvDir,
			Current:     "Discovering files",
			CurrentItem: -1,
			TotalItems:  -1,
		})
		shows, err := metadata.ScanTVShowsWithIndex(tvDir, fingerprints, func(path string, discovered int) {
			s.updateScan(scanStatus{
				Phase:       "Finding TV episode files",
				Source:      tvDir,
				Current:     path,
				CurrentItem: discovered,
			})
		})
		if err != nil {
			scanErrors = append(scanErrors, tvDir+": "+err.Error())
			s.updateScanErrors(scanErrors)
			continue
		}
		successfulRoots = append(successfulRoots, tvDir)
		allShows = append(allShows, shows...)
	}
	cached, consolidationErrors := s.consolidateTVShowDuplicates(r.Context(), allShows, cached)
	scanErrors = append(scanErrors, consolidationErrors...)
	cached, relocationErrors := s.relocateMovedTVShows(r.Context(), allShows, cached)
	scanErrors = append(scanErrors, relocationErrors...)
	cachedByPath = tvRecordsByPath(cached)
	counters := classifyTVEpisodes(allShows, cached, successfulRoots, request.ForceMetadata)
	allRecords, importErrors := s.importTVShowsWithCache(r.Context(), allShows, cachedByPath, request.ForceMetadata)
	scanErrors = append(scanErrors, importErrors...)
	discoveredShowPaths := make([]string, 0, len(allShows))
	for _, show := range allShows {
		discoveredShowPaths = append(discoveredShowPaths, show.Path)
	}
	if err := s.db.ReconcileTVShowPresence(r.Context(), successfulRoots, discoveredShowPaths, time.Now()); err != nil {
		scanErrors = append(scanErrors, "reconcile TV library: "+err.Error())
	} else if records, err := s.db.TVShows(r.Context()); err != nil {
		scanErrors = append(scanErrors, "reload reconciled TV library: "+err.Error())
	} else {
		allRecords = records
	}
	s.updateScanErrors(scanErrors)
	if err := s.scheduler.GenerateAll(r.Context()); err != nil {
		scanErrors = append(scanErrors, "regenerate schedules: "+err.Error())
		s.updateScanErrors(scanErrors)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tv_media_path": strings.Join(tvDirs, ", "),
		"sources":       tvDirs,
		"count":         len(allRecords),
		"scanned":       counters.Discovered,
		"discovered":    counters.Discovered,
		"unchanged":     counters.Unchanged,
		"new":           counters.New,
		"changed":       counters.Changed,
		"missing":       counters.Missing,
		"tv_shows":      allRecords,
		"errors":        scanErrors,
	})
}

type scanCounters struct{ Discovered, Unchanged, New, Changed, Missing int }

func (s *Server) consolidateMovieDuplicates(ctx context.Context, files []models.MovieFile, cached []models.MovieRecord) ([]models.MovieRecord, []string) {
	byPath := movieRecordsByPath(cached)
	removed := map[int64]struct{}{}
	errors := []string{}
	for _, file := range files {
		active, ok := byPath[file.Path]
		if !ok || active.MissingSince != nil {
			continue
		}
		matches := []models.MovieRecord{}
		for _, candidate := range cached {
			if _, alreadyRemoved := removed[candidate.ID]; alreadyRemoved {
				continue
			}
			if candidate.MissingSince != nil && sameFingerprint(file.MediaMetadata, candidate.MediaMetadata) {
				matches = append(matches, candidate)
			}
		}
		if len(matches) != 1 {
			continue
		}
		stale := matches[0]
		if err := s.db.MergeMovieRecords(ctx, stale.ID, active.ID); err != nil {
			errors = append(errors, file.Path+": consolidate prior movie record: "+err.Error())
			continue
		}
		removed[stale.ID] = struct{}{}
	}
	return recordsWithoutMovieIDs(cached, removed), errors
}

func (s *Server) consolidateTVShowDuplicates(ctx context.Context, files []models.TVShowFile, cached []models.TVShowRecord) ([]models.TVShowRecord, []string) {
	byPath := tvRecordsByPath(cached)
	removed := map[int64]struct{}{}
	errors := []string{}
	for _, file := range files {
		active, ok := byPath[file.Path]
		if !ok || active.MissingSince != nil {
			continue
		}
		matches := []models.TVShowRecord{}
		for _, candidate := range cached {
			if _, alreadyRemoved := removed[candidate.ID]; alreadyRemoved {
				continue
			}
			if candidate.MissingSince != nil && tvFingerprintSetsMatch(candidate, file) {
				matches = append(matches, candidate)
			}
		}
		if len(matches) != 1 {
			continue
		}
		stale := matches[0]
		pathMap := relocatedTVEpisodePathMap(stale.EpisodeMetadata, file)
		if err := s.db.MergeTVShowRecords(ctx, stale.ID, active.ID, pathMap); err != nil {
			errors = append(errors, file.Path+": consolidate prior TV record: "+err.Error())
			continue
		}
		removed[stale.ID] = struct{}{}
	}
	result := make([]models.TVShowRecord, 0, len(cached)-len(removed))
	for _, record := range cached {
		if _, drop := removed[record.ID]; !drop {
			result = append(result, record)
		}
	}
	return result, errors
}

func recordsWithoutMovieIDs(records []models.MovieRecord, removed map[int64]struct{}) []models.MovieRecord {
	result := make([]models.MovieRecord, 0, len(records)-len(removed))
	for _, record := range records {
		if _, drop := removed[record.ID]; !drop {
			result = append(result, record)
		}
	}
	return result
}

// relocateMovedMovies recognizes only unique, exact file fingerprints. An
// ambiguous match is deliberately left alone for review in the UI.
func (s *Server) relocateMovedMovies(ctx context.Context, files []models.MovieFile, cached []models.MovieRecord) ([]models.MovieRecord, []string) {
	used := map[int64]struct{}{}
	errors := []string{}
	for _, file := range files {
		if _, exists := movieRecordsByPath(cached)[file.Path]; exists {
			continue
		}
		matches := []models.MovieRecord{}
		for _, record := range cached {
			if record.MissingSince == nil {
				continue
			}
			if _, alreadyUsed := used[record.ID]; alreadyUsed {
				continue
			}
			if sameFingerprint(file.MediaMetadata, record.MediaMetadata) {
				matches = append(matches, record)
			}
		}
		if len(matches) != 1 {
			continue
		}
		match := matches[0]
		relocated, err := s.db.RelocateMovie(ctx, match.ID, file.Path, file.Filename, localTitle(file.Title, file.Filename), file.Year, file.MediaMetadata)
		if err != nil {
			errors = append(errors, file.Path+": relocate prior movie record: "+err.Error())
			continue
		}
		used[match.ID] = struct{}{}
		for index := range cached {
			if cached[index].ID == match.ID {
				cached[index] = relocated
				break
			}
		}
	}
	return cached, errors
}

func (s *Server) relocateMovedTVShows(ctx context.Context, files []models.TVShowFile, cached []models.TVShowRecord) ([]models.TVShowRecord, []string) {
	used := map[int64]struct{}{}
	errors := []string{}
	for _, file := range files {
		if _, exists := tvRecordsByPath(cached)[file.Path]; exists {
			continue
		}
		matches := []models.TVShowRecord{}
		for _, record := range cached {
			if record.MissingSince == nil {
				continue
			}
			if _, alreadyUsed := used[record.ID]; alreadyUsed {
				continue
			}
			if tvFingerprintSetsMatch(record, file) {
				matches = append(matches, record)
			}
		}
		if len(matches) != 1 {
			continue
		}
		match := matches[0]
		episodes := relocatedTVEpisodes(match.EpisodeMetadata, file)
		relocated, err := s.db.RelocateTVShow(ctx, match.ID, file.Path, localTVShow(file).Name, file.Year, file.EpisodePaths, episodes)
		if err != nil {
			errors = append(errors, file.Path+": relocate prior TV record: "+err.Error())
			continue
		}
		used[match.ID] = struct{}{}
		for index := range cached {
			if cached[index].ID == match.ID {
				cached[index] = relocated
				break
			}
		}
	}
	return cached, errors
}

func tvFingerprintSetsMatch(record models.TVShowRecord, file models.TVShowFile) bool {
	if len(record.EpisodeMetadata) == 0 || len(record.EpisodeMetadata) != len(file.EpisodeFingerprints) {
		return false
	}
	remaining := append([]models.TVEpisode(nil), record.EpisodeMetadata...)
	for _, fingerprint := range file.EpisodeFingerprints {
		matched := -1
		for index, episode := range remaining {
			if sameFingerprint(fingerprint, episode.MediaMetadata) {
				matched = index
				break
			}
		}
		if matched < 0 {
			return false
		}
		remaining = append(remaining[:matched], remaining[matched+1:]...)
	}
	return len(remaining) == 0
}

func relocatedTVEpisodes(old []models.TVEpisode, file models.TVShowFile) []models.TVEpisode {
	remaining := append([]models.TVEpisode(nil), old...)
	result := make([]models.TVEpisode, 0, len(file.EpisodePaths))
	for _, path := range file.EpisodePaths {
		fingerprint := file.EpisodeFingerprints[path]
		for index, episode := range remaining {
			if !sameFingerprint(fingerprint, episode.MediaMetadata) {
				continue
			}
			episode.Path = path
			episode.MediaMetadata = fingerprint
			result = append(result, episode)
			remaining = append(remaining[:index], remaining[index+1:]...)
			break
		}
	}
	return result
}

func relocatedTVEpisodePathMap(old []models.TVEpisode, file models.TVShowFile) map[string]string {
	remaining := append([]models.TVEpisode(nil), old...)
	result := map[string]string{}
	for _, newPath := range file.EpisodePaths {
		fingerprint := file.EpisodeFingerprints[newPath]
		for index, episode := range remaining {
			if !sameFingerprint(fingerprint, episode.MediaMetadata) {
				continue
			}
			result[episode.Path] = newPath
			remaining = append(remaining[:index], remaining[index+1:]...)
			break
		}
	}
	return result
}

func classifyMovies(files []models.MovieFile, cached map[string]models.MovieRecord, roots []string, force bool) scanCounters {
	c := scanCounters{Discovered: len(files)}
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		seen[file.Path] = struct{}{}
		record, ok := cached[file.Path]
		switch {
		case !ok:
			c.New++
		case !force && sameFingerprint(file.MediaMetadata, record.MediaMetadata):
			c.Unchanged++
		default:
			c.Changed++
		}
	}
	for path := range cached {
		if pathInRoots(path, roots) {
			if _, ok := seen[path]; !ok {
				c.Missing++
			}
		}
	}
	return c
}

func classifyTVEpisodes(files []models.TVShowFile, cached []models.TVShowRecord, roots []string, force bool) scanCounters {
	indexed := map[string]models.MediaMetadata{}
	for _, show := range cached {
		for _, episode := range show.EpisodeMetadata {
			indexed[episode.Path] = episode.MediaMetadata
		}
	}
	c := scanCounters{}
	seen := map[string]struct{}{}
	for _, show := range files {
		for path, fingerprint := range show.EpisodeFingerprints {
			c.Discovered++
			seen[path] = struct{}{}
			old, ok := indexed[path]
			switch {
			case !ok:
				c.New++
			case !force && sameFingerprint(fingerprint, old):
				c.Unchanged++
			default:
				c.Changed++
			}
		}
	}
	for path := range indexed {
		if pathInRoots(path, roots) {
			if _, ok := seen[path]; !ok {
				c.Missing++
			}
		}
	}
	return c
}

func pathInRoots(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
