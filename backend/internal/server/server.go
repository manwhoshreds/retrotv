package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"retrotv/internal/config"
	"retrotv/internal/database"
	"retrotv/internal/mediapath"
	"retrotv/internal/metadata"
	"retrotv/internal/models"
	"retrotv/internal/playback"
	"retrotv/internal/presentation"
	"retrotv/internal/scheduler"
)

const (
	streamingModeHLS             = config.StreamingModeHLS
	currentTVShowMetadataVersion = 2
)

type Server struct {
	staticFS         fs.FS
	store            *config.Store
	db               *database.Store
	artworkDir       string
	legacyLogoDir    string
	scheduler        *scheduler.Service
	schedulerRunning bool
	renderer         playback.Renderer
	sessions         *playback.SessionManager
	presentation     presentation.Resolver
	streamBaseURL    string
	streamDir        string
	streamWindowMu   sync.Mutex
	streamWindowID   uint64
	mu               sync.RWMutex
	settings         config.Settings
	movies           []models.MovieRecord
	scanMu           sync.RWMutex
	scanStatus       scanStatus
	scheduleMu       sync.RWMutex
	scheduleStatus   scheduleGenerationStatus
	scheduleRunID    int64
}

type scanStatus struct {
	Active       bool       `json:"active"`
	ContentType  string     `json:"content_type"`
	Phase        string     `json:"phase"`
	Source       string     `json:"source"`
	Current      string     `json:"current"`
	CurrentItem  int        `json:"current_item"`
	TotalItems   int        `json:"total_items"`
	Errors       int        `json:"errors"`
	ErrorDetails []string   `json:"error_details,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

type scheduleGenerationStatus struct {
	Active       bool       `json:"active"`
	Phase        string     `json:"phase"`
	Current      string     `json:"current"`
	CurrentItem  int        `json:"current_item"`
	TotalItems   int        `json:"total_items"`
	Errors       int        `json:"errors"`
	ErrorDetails []string   `json:"error_details,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

type Config struct {
	// StaticFS serves the web UI. When nil, files are read from StaticDir on disk.
	StaticFS          fs.FS
	StaticDir         string
	MediaDir          string
	ConfigDir         string
	DatabasePath      string
	ArtworkDir        string
	DisableScheduler  bool
	StreamStarter     playback.SessionStarter
	StreamBaseURL     string
	StreamDir         string
	StreamIdleTimeout time.Duration
}

func NewServer(serverConfig Config) (*Server, error) {
	staticFS := serverConfig.StaticFS
	if staticFS == nil {
		staticDir := serverConfig.StaticDir
		if staticDir == "" {
			staticDir = "web"
		}
		staticFS = os.DirFS(staticDir)
	}

	configDir := serverConfig.ConfigDir
	if configDir == "" {
		configDir = "data"
	}

	store := config.NewStore(filepath.Join(configDir, "config.json"))
	settings, err := store.Load()
	if err != nil {
		settings = config.DefaultSettings()
	}
	if serverConfig.MediaDir != "" {
		settings.MediaPath = serverConfig.MediaDir
	}

	databasePath := serverConfig.DatabasePath
	if databasePath == "" {
		databasePath = filepath.Join(configDir, "retrotv.sqlite")
	}

	db, err := database.Open(databasePath)
	if err != nil {
		return nil, err
	}

	artworkDir := serverConfig.ArtworkDir
	if artworkDir == "" {
		artworkDir = filepath.Join(configDir, "artwork")
	}
	streamDir := serverConfig.StreamDir
	if streamDir == "" {
		streamDir = filepath.Join(configDir, "streams")
	}
	legacyLogoDir := filepath.Join(configDir, "logos")

	server := &Server{
		staticFS:      staticFS,
		store:         store,
		db:            db,
		artworkDir:    artworkDir,
		legacyLogoDir: legacyLogoDir,
		renderer:      playback.DirectFileRenderer{},
		presentation:  presentation.Resolver{ArtworkDir: artworkDir},
		streamBaseURL: serverConfig.StreamBaseURL,
		streamDir:     streamDir,
		settings:      settings,
	}
	server.scheduler = scheduler.New(db)
	if !serverConfig.DisableScheduler {
		server.scheduler.Start(context.Background())
		server.schedulerRunning = true
	}
	if serverConfig.StreamStarter != nil {
		sessionOptions := []playback.SessionManagerOption{
			playback.WithChannelRequestProvider(server),
		}
		if serverConfig.StreamIdleTimeout > 0 {
			sessionOptions = append(sessionOptions, playback.WithIdleTimeout(serverConfig.StreamIdleTimeout))
		}
		server.sessions = playback.NewSessionManager(
			serverConfig.StreamStarter,
			sessionOptions...,
		)
	}
	return server, nil
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleMetadataFacets(w http.ResponseWriter, r *http.Request) {
	facets, err := s.db.MetadataFacets(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, facets)
}

func currentAndFuturePrograms(programs []models.ScheduledProgram, now time.Time) []models.ScheduledProgram {
	for index, program := range programs {
		if program.EndTime.After(now) {
			return programs[index:]
		}
	}
	return []models.ScheduledProgram{}
}

func (s *Server) startPersistentChannelStreams(ctx context.Context) error {
	if s.sessions == nil {
		return nil
	}
	if s.streamingMode() != streamingModeHLS {
		return nil
	}
	requests, err := s.persistentChannelRequests(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	return s.sessions.StartChannels(ctx, requests)
}

func (s *Server) persistentChannelRequests(ctx context.Context, now time.Time) ([]playback.ChannelRequest, error) {
	if s.streamingMode() != streamingModeHLS {
		return []playback.ChannelRequest{}, nil
	}
	channels, err := s.db.Channels(ctx)
	if err != nil {
		return nil, err
	}
	channels = airingChannels(channels)
	requests := []playback.ChannelRequest{}
	for _, channel := range channels {
		programs, err := s.db.CurrentAndNextScheduledPrograms(ctx, channel.ID, now)
		if err != nil {
			return nil, err
		}
		programs = effectiveBroadcastPrograms(programs)
		broadcast, ok := scheduler.CurrentBroadcast(programs, now)
		if !ok {
			continue
		}
		requests = append(requests, s.channelRequestFromBroadcast(channel, broadcast, now, s.streamBaseURL))
	}
	return requests, nil
}

func (s *Server) CurrentChannelRequest(ctx context.Context, channelID int64, now time.Time) (playback.ChannelRequest, bool, error) {
	if s.streamingMode() != streamingModeHLS {
		return playback.ChannelRequest{}, false, nil
	}
	channel, err := s.db.Channel(ctx, channelID)
	if err != nil {
		return playback.ChannelRequest{}, false, err
	}
	programs, err := s.db.CurrentAndNextScheduledPrograms(ctx, channelID, now)
	if err != nil {
		return playback.ChannelRequest{}, false, err
	}
	programs = effectiveBroadcastPrograms(programs)
	broadcast, ok := scheduler.CurrentBroadcast(programs, now)
	if !ok {
		return playback.ChannelRequest{}, false, nil
	}
	return s.channelRequestFromBroadcast(channel, broadcast, now, s.streamBaseURL), true, nil
}

func (s *Server) channelRequestFromBroadcast(channel models.Channel, broadcast scheduler.Broadcast, now time.Time, baseURL string) playback.ChannelRequest {
	program := broadcast.Program
	if baseURL == "" {
		baseURL = s.streamBaseURL
	}
	return playback.ChannelRequest{
		ChannelID: channel.ID,
		BaseURL:   baseURL,
		ProgramID: program.ID,
		StartTime: program.StartTime,
		EndTime:   program.EndTime,
		Request: playback.Request{
			ChannelID:    channel.ID,
			Program:      playbackProgram(program),
			SeekPosition: broadcast.SeekPosition,
			Presentation: s.presentation.Resolve(channel, program, now),
		},
	}
}

func (s *Server) importMovies(ctx context.Context, files []models.MovieFile) ([]models.MovieRecord, []string) {
	cached, err := s.db.Movies(ctx)
	if err != nil {
		return nil, []string{err.Error()}
	}
	return s.importMoviesWithCache(ctx, files, movieRecordsByPath(cached), false)
}

func (s *Server) importMoviesWithCache(ctx context.Context, files []models.MovieFile, cachedByPath map[string]models.MovieRecord, force bool) ([]models.MovieRecord, []string) {
	downloader := metadata.NewMetadataDownloader(os.Getenv("TMDB_READ_TOKEN"))
	prober := metadata.NewFFprobeProber()

	records := []models.MovieRecord{}
	scanErrors := []string{}
	for index, file := range files {
		var err error
		s.updateScan(scanStatus{
			Phase:       "Importing movie metadata",
			Source:      filepath.Dir(file.Path),
			Current:     scanDisplayName(file.Title, file.Filename),
			CurrentItem: index + 1,
			TotalItems:  len(files),
		})

		cachedRecord, hasCachedRecord := cachedByPath[file.Path]

		if !force && hasCachedRecord && sameFingerprint(file.MediaMetadata, cachedRecord.MediaMetadata) {
			records = append(records, cachedRecord)
			continue
		}

		cachedMediaMetadata := cachedRecord.MediaMetadata
		if force {
			cachedMediaMetadata = models.MediaMetadata{}
		}
		mediaMetadata := probeMediaMetadata(ctx, prober, file.Path, cachedMediaMetadata)

		if !force && hasCachedRecord && movieRecordComplete(cachedRecord) {
			cachedRecord.MediaMetadata = mediaMetadata
			record, err := s.db.UpsertMovie(ctx, cachedRecord)
			if err != nil {
				scanErrors = append(scanErrors, file.Path+": "+err.Error())
				s.updateScanErrors(scanErrors)
				continue
			}
			records = append(records, record)
			continue
		}

		movie, err := movieMetadata(ctx, downloader, file)
		if err != nil {
			scanErrors = append(scanErrors, file.Path+": "+err.Error())
			s.updateScanErrors(scanErrors)
			record := localMovieRecord(file, mediaMetadata)
			if hasCachedRecord {
				record = cachedRecord
				record.LocalTitle = localTitle(file.Title, file.Filename)
				record.LocalYear = file.Year
				record.MediaMetadata = mediaMetadata
			}
			record, err := s.db.UpsertMovie(ctx, record)
			if err != nil {
				scanErrors = append(scanErrors, file.Path+": "+err.Error())
				s.updateScanErrors(scanErrors)
				continue
			}
			records = append(records, record)
			continue
		}

		posterLocalPath, backdropLocalPath, err := downloader.DownloadArtwork(ctx, movie, s.artworkDir)
		if err != nil {
			scanErrors = append(scanErrors, file.Path+": "+err.Error())
			s.updateScanErrors(scanErrors)
		}

		record, err := s.db.UpsertMovie(ctx, models.MovieRecord{
			Movie:             movie,
			FilePath:          file.Path,
			Filename:          file.Filename,
			LocalTitle:        file.Title,
			LocalYear:         file.Year,
			MediaMetadata:     mediaMetadata,
			PosterLocalPath:   posterLocalPath,
			BackdropLocalPath: backdropLocalPath,
		})
		if err != nil {
			scanErrors = append(scanErrors, file.Path+": "+err.Error())
			s.updateScanErrors(scanErrors)
			continue
		}

		records = append(records, record)
	}

	return records, scanErrors
}

func (s *Server) importTVShows(ctx context.Context, files []models.TVShowFile) ([]models.TVShowRecord, []string) {
	cached, err := s.db.TVShows(ctx)
	if err != nil {
		return nil, []string{err.Error()}
	}
	return s.importTVShowsWithCache(ctx, files, tvRecordsByPath(cached), false)
}

func (s *Server) importTVShowsWithCache(ctx context.Context, files []models.TVShowFile, cachedByPath map[string]models.TVShowRecord, force bool) ([]models.TVShowRecord, []string) {
	downloader := metadata.NewMetadataDownloader(os.Getenv("TMDB_READ_TOKEN"))
	prober := metadata.NewFFprobeProber()

	records := []models.TVShowRecord{}
	scanErrors := []string{}
	for index, file := range files {
		var err error
		s.updateScan(scanStatus{
			Phase:       "Importing TV show metadata",
			Source:      filepath.Dir(file.Path),
			Current:     scanDisplayName(file.Title, filepath.Base(file.Path)),
			CurrentItem: index + 1,
			TotalItems:  len(files),
		})
		cachedRecord, hasCachedRecord := cachedByPath[file.Path]
		if !force && hasCachedRecord && tvFingerprintsMatch(cachedRecord, file) {
			records = append(records, cachedRecord)
			continue
		}

		show := cachedRecord.TVShow
		posterLocalPath := cachedRecord.PosterLocalPath
		backdropLocalPath := cachedRecord.BackdropLocalPath
		if force || !hasCachedRecord || !tvShowMetadataComplete(cachedRecord) {
			show, err = tvShowMetadata(ctx, downloader, file)
			if err != nil {
				scanErrors = append(scanErrors, file.Path+": "+err.Error())
				s.updateScanErrors(scanErrors)
				if hasCachedRecord {
					show = cachedRecord.TVShow
				} else {
					show = localTVShow(file)
					posterLocalPath = ""
					backdropLocalPath = ""
				}
			}
			if show.TMDBID > 0 {
				show.MetadataVersion = currentTVShowMetadataVersion
			}
		}

		if show.TMDBID > 0 && (force || !hasCachedRecord || !tvShowArtworkPresent(cachedRecord)) {
			posterLocalPath, backdropLocalPath, err = downloader.DownloadTVArtwork(ctx, show, s.artworkDir)
			if err != nil {
				scanErrors = append(scanErrors, file.Path+": "+err.Error())
				s.updateScanErrors(scanErrors)
			}
		}
		cachedEpisodes := cachedRecord.EpisodeMetadata
		if force {
			cachedEpisodes = nil
		}
		episodeMetadata := localTVEpisodeMetadata(file.EpisodePaths, cachedEpisodes)
		if show.TMDBID > 0 {
			var episodeErrors []string
			episodeMetadata, episodeErrors = downloader.DownloadTVEpisodeThumbnailsWithCache(ctx, show, file.EpisodePaths, filepath.Join(s.artworkDir, "tvshows"), cachedEpisodes)
			if len(episodeErrors) > 0 {
				scanErrors = append(scanErrors, episodeErrors...)
				s.updateScanErrors(scanErrors)
			}
		}
		episodeMetadata = probeEpisodeMediaMetadata(ctx, prober, episodeMetadata, cachedEpisodes)

		localShowTitle := localTVShow(file).Name
		record, err := s.db.UpsertTVShow(ctx, models.TVShowRecord{
			TVShow:            show,
			Path:              file.Path,
			LocalTitle:        localShowTitle,
			LocalYear:         file.Year,
			LocalEpisodeCount: file.EpisodeCount,
			EpisodePaths:      file.EpisodePaths,
			EpisodeMetadata:   episodeMetadata,
			PosterLocalPath:   posterLocalPath,
			BackdropLocalPath: backdropLocalPath,
		})
		if err != nil {
			scanErrors = append(scanErrors, file.Path+": "+err.Error())
			s.updateScanErrors(scanErrors)
			continue
		}
		records = append(records, record)
	}

	return records, scanErrors
}

func movieRecordsByPath(records []models.MovieRecord) map[string]models.MovieRecord {
	result := make(map[string]models.MovieRecord, len(records))
	for _, record := range records {
		result[record.FilePath] = record
	}
	return result
}

func tvRecordsByPath(records []models.TVShowRecord) map[string]models.TVShowRecord {
	result := make(map[string]models.TVShowRecord, len(records))
	for _, record := range records {
		result[record.Path] = record
	}
	return result
}

func sameFingerprint(left, right models.MediaMetadata) bool {
	return left.FileSize > 0 && left.ModTimeUnix > 0 &&
		left.FileSize == right.FileSize && left.ModTimeUnix == right.ModTimeUnix
}

func tvFingerprintsMatch(record models.TVShowRecord, file models.TVShowFile) bool {
	if len(record.EpisodeMetadata) != len(file.EpisodeFingerprints) {
		return false
	}
	for _, episode := range record.EpisodeMetadata {
		fingerprint, ok := file.EpisodeFingerprints[episode.Path]
		if !ok || !sameFingerprint(fingerprint, episode.MediaMetadata) {
			return false
		}
	}
	return len(file.EpisodeFingerprints) > 0
}

func localMovieRecord(file models.MovieFile, mediaMetadata models.MediaMetadata) models.MovieRecord {
	title := localTitle(file.Title, file.Filename)
	return models.MovieRecord{
		Movie: models.Movie{
			Title:         title,
			OriginalTitle: title,
			Year:          file.Year,
		},
		FilePath:      file.Path,
		Filename:      file.Filename,
		LocalTitle:    title,
		LocalYear:     file.Year,
		MediaMetadata: mediaMetadata,
	}
}

func localTitle(title string, filename string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	return title
}

func localTVShow(file models.TVShowFile) models.TVShow {
	title := strings.TrimSpace(file.Title)
	if title == "" {
		title = strings.TrimSpace(filepath.Base(file.Path))
	}
	return models.TVShow{
		Name:         title,
		OriginalName: title,
		Year:         file.Year,
		Episodes:     file.EpisodeCount,
	}
}

func localTVEpisodeMetadata(episodePaths []string, cachedEpisodes []models.TVEpisode) []models.TVEpisode {
	cachedByPath := map[string]models.TVEpisode{}
	for _, episode := range cachedEpisodes {
		cachedByPath[episode.Path] = episode
	}

	episodes := make([]models.TVEpisode, 0, len(episodePaths))
	for index, path := range episodePaths {
		if cached, ok := cachedByPath[path]; ok {
			episodes = append(episodes, cached)
			continue
		}
		episode := models.TVEpisode{Path: path, Episode: index + 1}
		if parsed, ok := metadata.ParseTVEpisodeFilename(path); ok {
			episode.Season = parsed.Season
			episode.Episode = parsed.Episode
		}
		episodes = append(episodes, episode)
	}
	return episodes
}

func (s *Server) cachedMovieRecord(ctx context.Context, file models.MovieFile) (models.MovieRecord, bool, error) {
	record, err := s.db.MovieByFilePath(ctx, file.Path)
	if errors.Is(err, sql.ErrNoRows) {
		return models.MovieRecord{}, false, nil
	}
	if err != nil {
		return models.MovieRecord{}, false, err
	}
	return record, true, nil
}

func movieRecordComplete(record models.MovieRecord) bool {
	if record.TMDBID <= 0 || record.Title == "" || record.Year <= 0 || record.Overview == "" {
		return false
	}
	if record.PosterPath != "" && !existingFile(record.PosterLocalPath) {
		return false
	}
	if record.BackdropPath != "" && !existingFile(record.BackdropLocalPath) {
		return false
	}
	return true
}

func (s *Server) cachedTVShowRecord(ctx context.Context, file models.TVShowFile) (models.TVShowRecord, bool, error) {
	record, err := s.db.TVShowByPath(ctx, file.Path)
	if errors.Is(err, sql.ErrNoRows) {
		return models.TVShowRecord{}, false, nil
	}
	if err != nil {
		return models.TVShowRecord{}, false, err
	}
	return record, true, nil
}

func cachedTVShowRecordComplete(record models.TVShowRecord, file models.TVShowFile) bool {
	return tvShowMetadataComplete(record) &&
		tvShowArtworkPresent(record) &&
		tvEpisodeMetadataComplete(record.EpisodeMetadata, file.EpisodePaths) &&
		tvEpisodeMediaMetadataFresh(record.EpisodeMetadata, file.EpisodePaths)
}

func tvShowMetadataComplete(record models.TVShowRecord) bool {
	return record.TMDBID > 0 &&
		record.MetadataVersion >= currentTVShowMetadataVersion &&
		record.Name != "" &&
		record.Year > 0 &&
		record.Overview != ""
}

func tvShowArtworkPresent(record models.TVShowRecord) bool {
	if record.PosterPath != "" && !existingFile(record.PosterLocalPath) {
		return false
	}
	if record.BackdropPath != "" && !existingFile(record.BackdropLocalPath) {
		return false
	}
	return true
}

func tvEpisodeMetadataComplete(episodes []models.TVEpisode, episodePaths []string) bool {
	byPath := map[string]models.TVEpisode{}
	for _, episode := range episodes {
		byPath[episode.Path] = episode
	}
	seen := map[string]struct{}{}
	for _, episodePath := range episodePaths {
		if episodePath == "" {
			continue
		}
		if _, ok := seen[episodePath]; ok {
			continue
		}
		seen[episodePath] = struct{}{}

		episode, ok := byPath[episodePath]
		if !ok || !tvEpisodeMetadataPresent(episode) {
			return false
		}
	}
	return len(seen) > 0
}

func tvEpisodeMetadataPresent(episode models.TVEpisode) bool {
	if episode.Path == "" || episode.Season < 0 || episode.Episode <= 0 || episode.Name == "" {
		return false
	}
	if episode.StillPath == "" {
		return true
	}
	return existingFile(episode.ThumbnailLocalPath)
}

func tvEpisodeMediaMetadataFresh(episodes []models.TVEpisode, episodePaths []string) bool {
	byPath := map[string]models.TVEpisode{}
	for _, episode := range episodes {
		byPath[episode.Path] = episode
	}
	seen := map[string]struct{}{}
	for _, episodePath := range episodePaths {
		if episodePath == "" {
			continue
		}
		if _, ok := seen[episodePath]; ok {
			continue
		}
		seen[episodePath] = struct{}{}
		episode, ok := byPath[episodePath]
		if !ok || !mediaMetadataFresh(episodePath, episode.MediaMetadata) {
			return false
		}
	}
	return len(seen) > 0
}

func mediaMetadataFresh(path string, mediaMetadata models.MediaMetadata) bool {
	if mediaMetadata.FileSize <= 0 || mediaMetadata.ModTimeUnix <= 0 {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() &&
		info.Size() == mediaMetadata.FileSize &&
		info.ModTime().Unix() == mediaMetadata.ModTimeUnix
}

func probeMediaMetadata(ctx context.Context, prober metadata.FFprobeProber, path string, cached models.MediaMetadata) models.MediaMetadata {
	if mediaMetadataFresh(path, cached) {
		return cached
	}
	mediaMetadata, err := prober.Probe(ctx, path)
	if err != nil {
		if cached.FileSize > 0 || cached.ModTimeUnix > 0 {
			cached.ProbeError = err.Error()
			return cached
		}
		return models.MediaMetadata{ProbeError: err.Error()}
	}
	return mediaMetadata
}

func probeEpisodeMediaMetadata(ctx context.Context, prober metadata.FFprobeProber, episodes []models.TVEpisode, cachedEpisodes []models.TVEpisode) []models.TVEpisode {
	cachedByPath := map[string]models.MediaMetadata{}
	for _, episode := range cachedEpisodes {
		cachedByPath[episode.Path] = episode.MediaMetadata
	}

	for index := range episodes {
		episodes[index].MediaMetadata = probeMediaMetadata(ctx, prober, episodes[index].Path, cachedByPath[episodes[index].Path])
	}
	return episodes
}

func existingFile(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir() && info.Size() > 0
	}
	if filepath.IsAbs(path) {
		return false
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	info, err = os.Stat(absolute)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func movieMetadata(ctx context.Context, downloader *metadata.MetadataDownloader, file models.MovieFile) (models.Movie, error) {
	if file.TMDBID != "" {
		tmdbID, err := strconv.Atoi(file.TMDBID)
		if err != nil {
			return models.Movie{}, err
		}

		return downloader.GetMovieByID(ctx, tmdbID)
	}

	return downloader.GetMovie(ctx, file.Title, file.Year)
}

func tvShowMetadata(ctx context.Context, downloader *metadata.MetadataDownloader, file models.TVShowFile) (models.TVShow, error) {
	if file.TMDBID != "" {
		tmdbID, err := strconv.Atoi(file.TMDBID)
		if err != nil {
			return models.TVShow{}, err
		}
		return downloader.GetTVShowByID(ctx, tmdbID)
	}

	return downloader.GetTVShow(ctx, file.Title, file.Year)
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid channel id")
		return 0, false
	}
	return id, true
}

func decodeChannelRequest(w http.ResponseWriter, r *http.Request) (models.Channel, bool) {
	var channel models.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return models.Channel{}, false
	}
	channel.Name = strings.TrimSpace(channel.Name)
	channel.Description = strings.TrimSpace(channel.Description)
	channel.ContentType = strings.TrimSpace(channel.ContentType)
	channel.LogoPath = strings.TrimSpace(channel.LogoPath)
	channel.Presentation.ChannelBugAssetPath = strings.TrimSpace(channel.Presentation.ChannelBugAssetPath)
	channel.Presentation.ChannelBugPosition = strings.TrimSpace(channel.Presentation.ChannelBugPosition)
	channel.Presentation.RatingOverlayPosition = strings.TrimSpace(channel.Presentation.RatingOverlayPosition)
	channel.Presentation.NowPlayingPosition = strings.TrimSpace(channel.Presentation.NowPlayingPosition)
	if channel.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return models.Channel{}, false
	}
	if channel.ContentType == "" {
		channel.ContentType = "movie"
	}
	switch channel.ContentType {
	case "movie", "tv", "mixed":
	default:
		writeError(w, http.StatusBadRequest, "content_type must be movie, tv, or mixed")
		return models.Channel{}, false
	}
	if !validOverlayPosition(channel.Presentation.ChannelBugPosition) ||
		!validOverlayPosition(channel.Presentation.RatingOverlayPosition) ||
		!validOverlayPosition(channel.Presentation.NowPlayingPosition) {
		writeError(w, http.StatusBadRequest, "overlay position must be top_left, top_right, bottom_left, or bottom_right")
		return models.Channel{}, false
	}
	if channel.Presentation.ChannelBugOpacity < 0 || channel.Presentation.ChannelBugOpacity > 1 {
		writeError(w, http.StatusBadRequest, "channel_bug_opacity must be between 0 and 1")
		return models.Channel{}, false
	}
	for i := range channel.Rules {
		channel.Rules[i].MediaType = strings.TrimSpace(channel.Rules[i].MediaType)
		channel.Rules[i].Field = strings.TrimSpace(channel.Rules[i].Field)
		channel.Rules[i].Operator = strings.TrimSpace(channel.Rules[i].Operator)
		channel.Rules[i].Value = strings.TrimSpace(channel.Rules[i].Value)
		if !validRule(channel.Rules[i]) {
			writeError(w, http.StatusBadRequest, "invalid channel rule")
			return models.Channel{}, false
		}
	}
	for i := range channel.ProgrammingBlocks {
		channel.ProgrammingBlocks[i].Title = strings.TrimSpace(channel.ProgrammingBlocks[i].Title)
		channel.ProgrammingBlocks[i].Name = strings.TrimSpace(channel.ProgrammingBlocks[i].Name)
		channel.ProgrammingBlocks[i].StartTime = strings.TrimSpace(channel.ProgrammingBlocks[i].StartTime)
		channel.ProgrammingBlocks[i].EndTime = strings.TrimSpace(channel.ProgrammingBlocks[i].EndTime)
		channel.ProgrammingBlocks[i].Placement = strings.TrimSpace(channel.ProgrammingBlocks[i].Placement)
		channel.ProgrammingBlocks[i].ContentSource = strings.TrimSpace(channel.ProgrammingBlocks[i].ContentSource)
		channel.ProgrammingBlocks[i].RuleSet = strings.TrimSpace(channel.ProgrammingBlocks[i].RuleSet)
		channel.ProgrammingBlocks[i].ScheduleMode = strings.TrimSpace(channel.ProgrammingBlocks[i].ScheduleMode)
		channel.ProgrammingBlocks[i].ExactDate = strings.TrimSpace(channel.ProgrammingBlocks[i].ExactDate)
		channel.ProgrammingBlocks[i].PlaybackMode = strings.TrimSpace(channel.ProgrammingBlocks[i].PlaybackMode)
		if channel.ProgrammingBlocks[i].Title == "" {
			channel.ProgrammingBlocks[i].Title = channel.ProgrammingBlocks[i].Name
		}
		if channel.ProgrammingBlocks[i].ContentSource == "" {
			channel.ProgrammingBlocks[i].ContentSource = "collection"
		}
		if channel.ProgrammingBlocks[i].Placement == "" {
			channel.ProgrammingBlocks[i].Placement = models.ProgrammingBlockPlacementTimeline
		}
		if channel.ProgrammingBlocks[i].ScheduleMode == "" {
			channel.ProgrammingBlocks[i].ScheduleMode = "weekly"
		}
		if channel.ProgrammingBlocks[i].PlaybackMode == "" {
			channel.ProgrammingBlocks[i].PlaybackMode = "shuffle"
		}
		if !validProgrammingBlock(channel.ProgrammingBlocks[i]) {
			writeError(w, http.StatusBadRequest, "invalid programming block")
			return models.Channel{}, false
		}
	}
	return channel, true
}

func validOverlayPosition(value string) bool {
	switch value {
	case "", "top_left", "top_right", "bottom_left", "bottom_right":
		return true
	default:
		return false
	}
}

func validRule(rule models.ChannelRule) bool {
	if rule.Value == "" {
		return false
	}
	switch rule.MediaType {
	case "", "movie", "tv":
	default:
		return false
	}
	switch rule.Field {
	case "title":
		return rule.Operator == "is" || rule.Operator == "is_not"
	case "genre":
		return rule.Operator == "is" || rule.Operator == "is_not"
	case "year":
		if _, err := strconv.Atoi(rule.Value); err != nil {
			return false
		}
		return rule.Operator == "equals" || rule.Operator == "greater_than" || rule.Operator == "less_than"
	case "runtime":
		if _, err := strconv.Atoi(rule.Value); err != nil {
			return false
		}
		return rule.Operator == "equals" || rule.Operator == "greater_than" || rule.Operator == "less_than"
	case "rating":
		return rule.Operator == "is" || rule.Operator == "is_not"
	case "first_air_date", "release_date", "date_added":
		if _, err := time.Parse("2006-01-02", rule.Value); err != nil {
			return false
		}
		return rule.Operator == "equals" || rule.Operator == "greater_than" || rule.Operator == "less_than"
	case "quality":
		return (rule.Value == "hd" || rule.Value == "sd") && (rule.Operator == "is" || rule.Operator == "is_not")
	case "studio", "network", "actor", "director", "writer", "country", "language", "series_status", "tag":
		return rule.Operator == "is" || rule.Operator == "is_not"
	default:
		return false
	}
}

func validProgrammingBlock(block models.ProgrammingBlock) bool {
	if block.Title == "" {
		return false
	}
	switch block.Placement {
	case models.ProgrammingBlockPlacementAllDay:
	case models.ProgrammingBlockPlacementTimeline:
		if _, err := time.Parse("15:04", block.StartTime); err != nil {
			return false
		}
		if _, err := time.Parse("15:04", block.EndTime); err != nil {
			return false
		}
	default:
		return false
	}
	switch block.ContentSource {
	case "rules", "specific", "media":
	case "collection":
		if block.CollectionID <= 0 {
			return false
		}
	default:
		return false
	}
	if len(block.Days) == 0 {
		return false
	}
	switch block.ScheduleMode {
	case "", "weekly":
	case "once":
		if _, err := time.Parse("2006-01-02", block.ExactDate); err != nil {
			return false
		}
	default:
		return false
	}
	switch block.PlaybackMode {
	case "", "shuffle", "in_order", "chronological":
	default:
		return false
	}
	for _, excluded := range block.ExcludedDates {
		if _, err := time.Parse("2006-01-02", excluded); err != nil {
			return false
		}
	}
	validDays := map[string]struct{}{"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {}}
	for _, day := range block.Days {
		if _, ok := validDays[day]; !ok {
			return false
		}
	}
	for _, item := range block.MediaItems {
		if item.MediaID <= 0 {
			return false
		}
		if item.MediaType != "movie" && item.MediaType != "tv" {
			return false
		}
	}
	return true
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &notDirectoryError{path: path}
	}

	return nil
}

func firstMediaSourcePath(sources []config.MediaSource, contentType string) string {
	for _, source := range sources {
		if source.ContentType == contentType {
			return source.Path
		}
	}
	return ""
}

func sourcePaths(settings config.Settings, contentType string) []string {
	paths := []string{}
	for _, source := range settings.MediaSources {
		if source.ContentType == contentType {
			paths = append(paths, source.Path)
		}
	}
	if len(settings.MediaSources) > 0 {
		return paths
	}
	if len(paths) == 0 {
		if contentType == "movie" && settings.MediaPath != "" {
			paths = append(paths, settings.MediaPath)
		}
		if contentType == "tv" && settings.TVMediaPath != "" {
			paths = append(paths, settings.TVMediaPath)
		}
	}
	return paths
}

func (s *Server) beginScan(contentType string, phase string, source string) {
	now := time.Now()
	s.scanMu.Lock()
	s.scanStatus = scanStatus{
		Active:      true,
		ContentType: contentType,
		Phase:       phase,
		Source:      source,
		StartedAt:   &now,
		UpdatedAt:   &now,
	}
	s.scanMu.Unlock()
}

func (s *Server) currentScanStatus() scanStatus {
	s.scanMu.RLock()
	status := s.scanStatus
	s.scanMu.RUnlock()
	return status
}

func (s *Server) updateScan(update scanStatus) {
	now := time.Now()
	s.scanMu.Lock()
	status := s.scanStatus
	if update.ContentType != "" {
		status.ContentType = update.ContentType
	}
	if update.Phase != "" {
		status.Phase = update.Phase
	}
	if update.Source != "" {
		status.Source = update.Source
	}
	if update.Current != "" {
		status.Current = update.Current
	}
	if update.CurrentItem < 0 {
		status.CurrentItem = 0
	} else if update.CurrentItem > 0 {
		status.CurrentItem = update.CurrentItem
	}
	if update.TotalItems < 0 {
		status.TotalItems = 0
	} else if update.TotalItems > 0 {
		status.TotalItems = update.TotalItems
	}
	if update.Errors > 0 {
		status.Errors = update.Errors
	}
	if len(update.ErrorDetails) > 0 {
		status.ErrorDetails = append([]string(nil), update.ErrorDetails...)
		status.Errors = len(status.ErrorDetails)
	}
	status.UpdatedAt = &now
	s.scanStatus = status
	s.scanMu.Unlock()
}

func (s *Server) updateScanErrors(errors []string) {
	if len(errors) == 0 {
		return
	}
	s.updateScan(scanStatus{
		Errors:       len(errors),
		ErrorDetails: errors,
	})
}

func (s *Server) finishScan(phase string) {
	now := time.Now()
	s.scanMu.Lock()
	status := s.scanStatus
	status.Active = false
	status.Phase = phase
	status.Current = ""
	status.UpdatedAt = &now
	status.FinishedAt = &now
	s.scanStatus = status
	s.scanMu.Unlock()
}

func scanDisplayName(title string, fallback string) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return "Unknown media"
}

func logoExtension(contentType string, originalExt string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	}

	switch strings.ToLower(originalExt) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return strings.ToLower(originalExt), false
	default:
		return "", false
	}
}

func sanitizeFilename(value string) string {
	value = strings.ToLower(value)
	builder := strings.Builder{}
	lastDash := false
	for _, r := range value {
		isAlphaNumeric := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNumeric {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func requestBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}

	host := r.Host
	if host == "" {
		host = "localhost"
	}

	return scheme + "://" + host
}

func publicURL(baseURL string, path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return strings.TrimRight(baseURL, "/") + path
	}
	return path
}

func artworkURLPath(path string) string {
	normalized := filepath.ToSlash(path)
	const prefix = "data/artwork/"
	if index := strings.Index(normalized, prefix); index >= 0 {
		return "/artwork/" + normalized[index+len(prefix):]
	}
	return path
}

func firstTVEpisodePath(show models.TVShowRecord) string {
	paths := tvEpisodePaths(show)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func tvProgramEpisodePath(program models.ScheduledProgram) string {
	if program.EpisodePath != "" {
		if resolved, ok := resolveTVEpisodePath(program.TVMedia, program.EpisodePath); ok {
			return resolved
		}
		return program.EpisodePath
	}
	return firstTVEpisodePath(program.TVMedia)
}

func tvEpisodeBelongsToShow(show models.TVShowRecord, episodePath string) bool {
	_, ok := resolveTVEpisodePath(show, episodePath)
	return ok
}

func resolveTVEpisodePath(show models.TVShowRecord, episodePath string) (string, bool) {
	requested := absoluteMediaPath(episodePath)
	paths := tvEpisodePaths(show)
	for _, path := range paths {
		if absoluteMediaPath(path) == requested {
			return path, true
		}
	}

	// Source moves change the root but normally preserve Season/filename. Use
	// that stable suffix only when it identifies exactly one current episode.
	requestedSuffix := episodePathSuffix(episodePath)
	match := ""
	for _, path := range paths {
		if requestedSuffix == "" || episodePathSuffix(path) != requestedSuffix {
			continue
		}
		if match != "" {
			return "", false
		}
		match = path
	}
	return match, match != ""
}

func episodePathSuffix(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return path
	}
	return strings.ToLower(strings.Join(parts[len(parts)-2:], "/"))
}

func tvEpisodePaths(show models.TVShowRecord) []string {
	paths := []string{}
	seen := map[string]struct{}{}
	for _, path := range show.EpisodePaths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	_ = filepath.WalkDir(show.Path, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".mp4", ".mkv", ".avi", ".mov":
			if _, ok := seen[path]; ok {
				return nil
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
		return nil
	})
	return paths
}

func absoluteMediaPath(path string) string {
	path = mediapath.Resolve(path)
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}

type xmltvTV struct {
	XMLName    xml.Name         `xml:"tv"`
	Channels   []xmltvChannel   `xml:"channel"`
	Programmes []xmltvProgramme `xml:"programme"`
}

type xmltvChannel struct {
	ID          string         `xml:"id,attr"`
	DisplayName xmltvTextValue `xml:"display-name"`
	Icon        *xmltvIcon     `xml:"icon,omitempty"`
}

type xmltvIcon struct {
	Src string `xml:"src,attr"`
}

type xmltvProgramme struct {
	Start   string         `xml:"start,attr"`
	Stop    string         `xml:"stop,attr"`
	Channel string         `xml:"channel,attr"`
	Title   xmltvTextValue `xml:"title"`
	Desc    xmltvTextValue `xml:"desc"`
	Icon    *xmltvIcon     `xml:"icon,omitempty"`
}

type xmltvTextValue struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

func xmltvResponse(channel models.Channel, programs []models.ScheduledProgram, baseURL string) xmltvTV {
	channelID := fmt.Sprintf("retrotv-%d", channel.ID)
	tvChannel := xmltvChannel{
		ID:          channelID,
		DisplayName: xmltvTextValue{Value: channel.Name},
	}
	if channel.LogoPath != "" {
		tvChannel.Icon = &xmltvIcon{Src: publicURL(baseURL, channel.LogoPath)}
	}

	response := xmltvTV{
		Channels:   []xmltvChannel{tvChannel},
		Programmes: []xmltvProgramme{},
	}
	for _, program := range guideVisiblePrograms(programs) {
		response.Programmes = append(response.Programmes, xmltvProgramme{
			Start:   xmltvTime(program.StartTime),
			Stop:    xmltvTime(program.EndTime),
			Channel: channelID,
			Title:   xmltvTextValue{Value: programTitle(program)},
			Desc:    xmltvTextValue{Value: programOverview(program)},
			Icon:    xmltvProgramIcon(program, baseURL),
		})
	}
	return response
}

func guideVisiblePrograms(programs []models.ScheduledProgram) []models.ScheduledProgram {
	visible := make([]models.ScheduledProgram, 0, len(programs))
	for _, program := range programs {
		if program.MediaType == "asset" {
			if len(visible) > 0 && program.EndTime.After(visible[len(visible)-1].EndTime) {
				visible[len(visible)-1].EndTime = program.EndTime
			}
			continue
		}
		visible = append(visible, program)
	}
	return visible
}

func xmltvWindow(r *http.Request, now time.Time) (time.Time, time.Time) {
	hours := 24
	if value := r.URL.Query().Get("hours"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 {
			hours = parsed
		}
	}
	if hours > 168 {
		hours = 168
	}
	start := now.UTC().Add(-30 * time.Minute)
	end := now.UTC().Add(time.Duration(hours) * time.Hour)
	if value := r.URL.Query().Get("start"); value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			start = parsed.UTC()
			end = start.Add(time.Duration(hours) * time.Hour)
		}
	}
	return start, end
}

func xmltvProgramIcon(program models.ScheduledProgram, baseURL string) *xmltvIcon {
	path := programPosterPath(program)
	if path == "" {
		return nil
	}
	return &xmltvIcon{Src: publicURL(baseURL, artworkURLPath(path))}
}

func programTitle(program models.ScheduledProgram) string {
	if program.MediaType == "asset" {
		return assetProgramTitle(program.Asset)
	}
	if program.MediaType == "tv" {
		if episode, ok := scheduledEpisode(program); ok {
			code := episodeCode(episode)
			title := tvShowDisplayTitle(program.TVMedia)
			if code != "" {
				title += " " + code
			}
			if episode.Name != "" {
				title += " - " + episode.Name
			}
			return title
		}
		return tvShowDisplayTitle(program.TVMedia)
	}
	return program.Media.Title
}

func tvShowDisplayTitle(show models.TVShowRecord) string {
	if title := strings.TrimSpace(show.Name); title != "" {
		return title
	}
	return strings.TrimSpace(show.LocalTitle)
}

func programOverview(program models.ScheduledProgram) string {
	if program.MediaType == "asset" {
		return program.Asset.AssetType
	}
	if program.MediaType == "tv" {
		if episode, ok := scheduledEpisode(program); ok && episode.Overview != "" {
			return episode.Overview
		}
		return program.TVMedia.Overview
	}
	return program.Media.Overview
}

func programPosterPath(program models.ScheduledProgram) string {
	if program.MediaType == "asset" {
		return ""
	}
	if program.MediaType == "tv" {
		return program.TVMedia.PosterLocalPath
	}
	return program.Media.PosterLocalPath
}

func scheduledEpisode(program models.ScheduledProgram) (models.TVEpisode, bool) {
	if program.MediaType != "tv" || program.EpisodePath == "" {
		return models.TVEpisode{}, false
	}
	requested := absoluteMediaPath(program.EpisodePath)
	for _, episode := range program.TVMedia.EpisodeMetadata {
		if absoluteMediaPath(episode.Path) == requested {
			return episode, true
		}
	}
	return models.TVEpisode{}, false
}

func episodeCode(episode models.TVEpisode) string {
	if episode.Season <= 0 || episode.Episode <= 0 {
		return ""
	}
	return fmt.Sprintf("S%02dE%02d", episode.Season, episode.Episode)
}

func playbackProgram(program models.ScheduledProgram) playback.Program {
	if program.MediaType == "asset" {
		return playback.Program{
			MediaType:  playback.MediaTypeAsset,
			AssetID:    program.Asset.ID,
			SourcePath: absoluteMediaPath(program.Asset.FilePath),
		}
	}
	if program.MediaType == "tv" {
		return playback.Program{
			MediaType:   playback.MediaTypeTV,
			TVShowID:    program.TVMedia.ID,
			EpisodePath: tvProgramEpisodePath(program),
			SourcePath:  absoluteMediaPath(tvProgramEpisodePath(program)),
		}
	}
	return playback.Program{
		MediaType:  playback.MediaTypeMovie,
		MovieID:    program.Media.ID,
		SourcePath: absoluteMediaPath(program.Media.FilePath),
	}
}

func assetProgramTitle(asset models.ChannelAsset) string {
	switch asset.AssetType {
	case "bumper":
		return "Bumper: " + asset.Filename
	case "commercial":
		return "Commercial: " + asset.Filename
	case "promo":
		return "Promo: " + asset.Filename
	case "station_id":
		return "Station ID: " + asset.Filename
	default:
		return asset.Filename
	}
}

func (s *Server) renderPlayback(baseURL string, program models.ScheduledProgram, seekPosition int) playback.Playback {
	seekPosition = playableSeekPosition(program, seekPosition)
	if s.streamingMode() == streamingModeHLS && s.sessions != nil {
		if playback, ok := s.sessions.Playback(program.ChannelID); ok {
			return playback
		}
		return playback.Playback{
			URL:          channelHLSPlaylistURL(baseURL, program.ChannelID),
			SeekPosition: seekPosition,
		}
	}
	return s.renderer.Render(baseURL, playback.Request{
		ChannelID:    program.ChannelID,
		Program:      playbackProgram(program),
		SeekPosition: seekPosition,
	})
}

func playableSeekPosition(program models.ScheduledProgram, seekPosition int) int {
	if seekPosition <= 0 {
		return 0
	}
	duration := playableDurationSeconds(program)
	if duration <= 0 || seekPosition < duration-2 {
		return seekPosition
	}
	log.Printf("event=playback_seek_clamped channel=%d program=%d media_type=%q requested_seek=%d duration=%d", program.ChannelID, program.ID, program.MediaType, seekPosition, duration)
	return 0
}

func effectiveBroadcastPrograms(programs []models.ScheduledProgram) []models.ScheduledProgram {
	effective := make([]models.ScheduledProgram, 0, len(programs))
	for _, program := range programs {
		duration := playableDurationSeconds(program)
		if duration > 0 {
			mediaEnd := program.StartTime.Add(time.Duration(duration) * time.Second)
			if mediaEnd.Before(program.EndTime) {
				program.EndTime = mediaEnd
			}
		}
		effective = append(effective, program)
	}
	return effective
}

func playableDurationSeconds(program models.ScheduledProgram) int {
	if program.MediaType == "asset" {
		return program.Asset.DurationSeconds
	}
	if program.MediaType == "tv" {
		for _, episode := range program.TVMedia.EpisodeMetadata {
			if episode.Path != program.EpisodePath {
				continue
			}
			if episode.MediaMetadata.DurationSeconds > 0 {
				return int(episode.MediaMetadata.DurationSeconds)
			}
			if episode.RuntimeMinutes > 0 {
				return episode.RuntimeMinutes * 60
			}
		}
		return 0
	}
	if program.Media.MediaMetadata.DurationSeconds > 0 {
		return int(program.Media.MediaMetadata.DurationSeconds)
	}
	if program.Media.RuntimeMinutes > 0 {
		return program.Media.RuntimeMinutes * 60
	}
	return 0
}

func (s *Server) streamingMode() string {
	s.mu.RLock()
	mode := s.settings.StreamingMode
	s.mu.RUnlock()
	return config.NormalizeStreamingMode(mode)
}

func channelHLSPlaylistURL(baseURL string, channelID int64) string {
	return strings.TrimRight(baseURL, "/") + "/streams/channels/" + strconv.FormatInt(channelID, 10) + "/index.m3u8"
}

func xmltvTime(value time.Time) string {
	return value.UTC().Format("20060102150405 -0700")
}

func escapeM3U(value string) string {
	return strings.NewReplacer(`"`, `'`, "\n", " ", "\r", " ").Replace(value)
}

type notDirectoryError struct {
	path string
}

func (e *notDirectoryError) Error() string {
	return e.path + " is not a directory"
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
