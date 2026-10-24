package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"retrotv/internal/models"
	"retrotv/internal/presentation"
	"retrotv/internal/scheduler"
)

func (s *Server) handleGetChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.db.Channels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if r.URL.Query().Get("airing") == "true" {
		channels = airingChannels(channels)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"channels": channels,
		"count":    len(channels),
	})
}

func (s *Server) handleSetChannelAiring(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var request struct {
		Airing bool `json:"airing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.db.SetChannelAiring(r.Context(), id, request.Airing); err != nil {
		writeStoreError(w, err)
		return
	}
	if request.Airing {
		channel, err := s.db.Channel(r.Context(), id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		s.regenerateChannelScheduleAsync(channel)
	} else {
		if err := s.db.ReplaceScheduledPrograms(r.Context(), id, nil); err != nil {
			writeStoreError(w, err)
			return
		}
		if s.sessions != nil {
			s.sessions.StopChannel(context.Background(), id)
		}
	}
	channel, err := s.db.Channel(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, channel)
}

func airingChannels(channels []models.Channel) []models.Channel {
	result := make([]models.Channel, 0, len(channels))
	for _, channel := range channels {
		if channel.Airing {
			result = append(result, channel)
		}
	}
	return result
}

func (s *Server) handleGetChannel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	channel, err := s.db.Channel(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, channel)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.db.DeleteChannel(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	channel, ok := decodeChannelRequest(w, r)
	if !ok {
		return
	}
	saved, err := s.db.SaveChannel(r.Context(), channel)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	saved, err = s.promoteChannelArtwork(r.Context(), saved)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.regenerateChannelScheduleAsync(saved)
	writeJSON(w, http.StatusCreated, saved)
}

func (s *Server) handleReorderChannels(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ChannelIDs []int64 `json:"channel_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	channels, err := s.db.Channels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(request.ChannelIDs) != len(channels) {
		writeError(w, http.StatusBadRequest, "channel_ids must include every channel exactly once")
		return
	}
	seen := map[int64]struct{}{}
	existing := map[int64]struct{}{}
	for _, channel := range channels {
		existing[channel.ID] = struct{}{}
	}
	for _, id := range request.ChannelIDs {
		if _, ok := existing[id]; !ok {
			writeError(w, http.StatusBadRequest, "channel_ids contains an unknown channel")
			return
		}
		if _, ok := seen[id]; ok {
			writeError(w, http.StatusBadRequest, "channel_ids contains a duplicate channel")
			return
		}
		seen[id] = struct{}{}
	}

	if err := s.db.ReorderChannels(r.Context(), request.ChannelIDs); err != nil {
		writeStoreError(w, err)
		return
	}
	channels, err = s.db.Channels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channels": channels,
		"count":    len(channels),
	})
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	channel, ok := decodeChannelRequest(w, r)
	if !ok {
		return
	}
	channel.ID = id
	existing, err := s.db.Channel(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	// Channel updates change station identity only. Programming has its own endpoint.
	channel.ProgrammingBlocks = existing.ProgrammingBlocks
	channel.Rules = existing.Rules
	if len(channel.ProgrammingBlocks) == 1 && channel.ProgrammingBlocks[0].ID < 0 {
		channel.ProgrammingBlocks = nil
	}
	saved, err := s.db.SaveChannel(r.Context(), channel)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	saved, err = s.promoteChannelArtwork(r.Context(), saved)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.regenerateChannelScheduleAsync(saved)
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) handleUpdateChannelProgramming(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var request struct {
		ProgrammingBlocks []models.ProgrammingBlock `json:"programming_blocks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	for index := range request.ProgrammingBlocks {
		block := &request.ProgrammingBlocks[index]
		block.Title = strings.TrimSpace(block.Title)
		block.Name = strings.TrimSpace(block.Name)
		if block.Title == "" {
			block.Title = block.Name
		}
		if block.Placement == "" {
			block.Placement = models.ProgrammingBlockPlacementTimeline
		}
		if block.ContentSource == "" {
			block.ContentSource = "collection"
		}
		if block.ScheduleMode == "" {
			block.ScheduleMode = "weekly"
		}
		if block.PlaybackMode == "" {
			block.PlaybackMode = "shuffle"
		}
		if !validProgrammingBlock(*block) {
			writeError(w, http.StatusBadRequest, "invalid programming block")
			return
		}
		if block.CollectionID > 0 {
			if _, err := s.db.Collection(r.Context(), block.CollectionID); err != nil {
				writeError(w, http.StatusBadRequest, "programming block references an unknown collection")
				return
			}
		}
	}
	channel, err := s.db.Channel(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	previousBlocks := channel.ProgrammingBlocks
	channel.Rules = nil
	channel.ProgrammingBlocks = request.ProgrammingBlocks
	saved, err := s.db.SaveChannel(r.Context(), channel)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.scheduler.PatchChannelSchedule(r.Context(), saved.ID, previousBlocks, request.ProgrammingBlocks); err != nil {
		writeError(w, http.StatusInternalServerError, "programming was saved, but the schedule could not be regenerated: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) regenerateChannelScheduleAsync(channel models.Channel) {
	if s.schedulerRunning {
		s.scheduler.QueueRegenerateChannel(channel.ID)
		return
	}

	go func() {
		if err := s.scheduler.RegenerateChannel(context.Background(), channel.ID); err != nil {
			log.Printf("scheduler channel %d %q: %v", channel.ID, channel.Name, err)
		}
	}()
}

func (s *Server) handleUploadChannelLogo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	file, header, err := r.FormFile("logo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "logo image is required")
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	contentType := http.DetectContentType(buffer[:n])
	ext, ok := logoExtension(contentType, filepath.Ext(header.Filename))
	if !ok {
		writeError(w, http.StatusBadRequest, "logo must be a PNG, JPEG, WEBP, or GIF image")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	channelID := sanitizeFilename(strings.TrimSpace(r.FormValue("channel_id")))
	channelLogoDir := filepath.Join(s.artworkDir, "channels", channelID)
	if channelID == "" {
		channelLogoDir = filepath.Join(s.artworkDir, "channels", "uploads")
	}
	if err := os.MkdirAll(channelLogoDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	name := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	name = sanitizeFilename(name)
	if name == "" {
		name = "channel-logo"
	}
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), name, ext)
	if channelID != "" {
		filename = "logo" + ext
	}
	destination := filepath.Join(channelLogoDir, filename)
	flags := os.O_CREATE | os.O_EXCL | os.O_WRONLY
	if channelID != "" {
		flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	}
	output, err := os.OpenFile(destination, flags, 0o644)
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

	logoPath := "/artwork/channels/uploads/" + filename
	if channelID != "" {
		logoPath = "/artwork/channels/" + channelID + "/" + filename
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"logo_path": logoPath,
	})
}

func (s *Server) promoteChannelArtwork(ctx context.Context, channel models.Channel) (models.Channel, error) {
	const uploadPrefix = "/artwork/channels/uploads/"
	if channel.ID <= 0 || !strings.HasPrefix(channel.LogoPath, uploadPrefix) {
		return channel, nil
	}

	filename := filepath.Base(channel.LogoPath)
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".png"
	}
	source := filepath.Join(s.artworkDir, "channels", "uploads", filename)
	destinationDir := filepath.Join(s.artworkDir, "channels", strconv.FormatInt(channel.ID, 10))
	if err := os.MkdirAll(destinationDir, 0o755); err != nil {
		return channel, err
	}
	destination := filepath.Join(destinationDir, "logo"+ext)
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return channel, err
	}
	if err := os.Rename(source, destination); err != nil {
		return channel, err
	}

	channel.LogoPath = "/artwork/channels/" + strconv.FormatInt(channel.ID, 10) + "/logo" + ext
	return s.db.SaveChannel(ctx, channel)
}

func (s *Server) handleChannelSchedule(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if _, err := s.db.Channel(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	var programs []models.ScheduledProgram
	var err error
	startValue, endValue := r.URL.Query().Get("start"), r.URL.Query().Get("end")
	if startValue != "" || endValue != "" {
		start, startErr := time.Parse(time.RFC3339, startValue)
		end, endErr := time.Parse(time.RFC3339, endValue)
		if startErr != nil || endErr != nil || !end.After(start) {
			writeError(w, http.StatusBadRequest, "start and end must be valid RFC3339 times with end after start")
			return
		}
		programs, err = s.db.ScheduledProgramsWindow(r.Context(), id, start, end)
	} else {
		programs, err = s.db.ScheduledPrograms(r.Context(), id)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if r.URL.Query().Get("current") == "1" {
		now := time.Now().UTC()
		current := make([]models.ScheduledProgram, 0, 1)
		for _, program := range programs {
			if !program.StartTime.After(now) && program.EndTime.After(now) {
				current = append(current, program)
			}
		}
		programs = current
	}
	if r.URL.Query().Get("summary") == "1" {
		writeJSON(w, http.StatusOK, map[string]int{"count": len(programs)})
		return
	}

	type scheduledMedia struct {
		ID              int64    `json:"id"`
		Type            string   `json:"type"`
		Title           string   `json:"title"`
		Overview        string   `json:"overview"`
		ReleaseDate     string   `json:"release_date,omitempty"`
		Year            int      `json:"year,omitempty"`
		FilePath        string   `json:"file_path"`
		PosterLocalPath string   `json:"poster_local_path"`
		StreamURL       string   `json:"stream_url"`
		Genres          []string `json:"genres,omitempty"`
		Rating          string   `json:"rating,omitempty"`
	}
	type scheduledProgram struct {
		ID           int64                `json:"id"`
		ChannelID    int64                `json:"channel_id"`
		MediaID      int64                `json:"media_id"`
		StartTime    time.Time            `json:"start_time"`
		EndTime      time.Time            `json:"end_time"`
		Media        scheduledMedia       `json:"media"`
		ChannelAsset *models.ChannelAsset `json:"channel_asset,omitempty"`
	}
	baseURL := requestBaseURL(r)
	response := struct {
		Programs      []scheduledProgram    `json:"programs"`
		ChannelAssets []models.ChannelAsset `json:"channel_assets"`
		Count         int                   `json:"count"`
	}{
		Programs:      []scheduledProgram{},
		ChannelAssets: []models.ChannelAsset{},
	}
	includedAssets := map[int64]struct{}{}
	mediaResponse := func(program models.ScheduledProgram) scheduledMedia {
		if program.MediaType == "asset" {
			return scheduledMedia{
				ID:        program.Asset.ID,
				Type:      "asset",
				Title:     assetProgramTitle(program.Asset),
				Overview:  program.Asset.AssetType,
				FilePath:  absoluteMediaPath(program.Asset.FilePath),
				StreamURL: s.renderPlayback(baseURL, program, 0).URL,
			}
		}
		if program.MediaType == "tv" {
			return scheduledMedia{
				ID:              program.TVMedia.ID,
				Type:            "tv",
				Title:           tvShowDisplayTitle(program.TVMedia),
				Overview:        program.TVMedia.Overview,
				ReleaseDate:     program.TVMedia.FirstAirDate,
				Year:            program.TVMedia.Year,
				FilePath:        absoluteMediaPath(tvProgramEpisodePath(program)),
				PosterLocalPath: absoluteMediaPath(program.TVMedia.PosterLocalPath),
				StreamURL:       s.renderPlayback(baseURL, program, 0).URL,
				Genres:          program.TVMedia.Genres,
				Rating:          program.TVMedia.Rating,
			}
		}
		return scheduledMedia{
			ID:              program.Media.ID,
			Type:            "movie",
			Title:           program.Media.Title,
			Overview:        program.Media.Overview,
			ReleaseDate:     program.Media.ReleaseDate,
			Year:            program.Media.Year,
			FilePath:        absoluteMediaPath(program.Media.FilePath),
			PosterLocalPath: absoluteMediaPath(program.Media.PosterLocalPath),
			StreamURL:       s.renderPlayback(baseURL, program, 0).URL,
			Genres:          program.Media.Genres,
			Rating:          program.Media.Rating,
		}
	}
	for _, program := range programs {
		programResponse := scheduledProgram{
			ID:        program.ID,
			ChannelID: program.ChannelID,
			MediaID:   program.MediaID,
			StartTime: program.StartTime,
			EndTime:   program.EndTime,
			Media:     mediaResponse(program),
		}
		if program.MediaType == "asset" {
			asset := program.Asset
			programResponse.ChannelAsset = &asset
			if _, ok := includedAssets[asset.ID]; !ok {
				includedAssets[asset.ID] = struct{}{}
				response.ChannelAssets = append(response.ChannelAssets, asset)
			}
		}
		response.Programs = append(response.Programs, programResponse)
	}
	response.Count = len(response.Programs)
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleTuneChannel(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	mode := s.streamingMode()
	var streamWindowID uint64
	if mode == streamingModeHLS && s.sessions != nil {
		s.streamWindowMu.Lock()
		s.streamWindowID++
		streamWindowID = s.streamWindowID
		s.streamWindowMu.Unlock()
	}
	channel, err := s.db.Channel(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if !channel.Airing {
		writeError(w, http.StatusNotFound, "channel is not airing")
		return
	}

	now := time.Now().UTC()
	programs, err := s.db.CurrentAndNextScheduledPrograms(r.Context(), id, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	programs = effectiveBroadcastPrograms(programs)
	viewerSchedule, err := s.db.ViewerSchedule(r.Context(), id, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	viewerPrograms := presentation.ResolveViewerPrograms(viewerSchedule, now)
	baseURL := requestBaseURL(r)
	response := tuneResponse{
		StreamingMode: mode,
		Channel: tuneChannel{
			ID:      channel.ID,
			Name:    channel.Name,
			LogoURL: publicURL(baseURL, channel.LogoPath),
		},
	}

	if broadcast, ok := scheduler.CurrentBroadcast(programs, now); ok {
		if mode == streamingModeHLS && s.sessions != nil {
			if err := s.sessions.EnsureChannel(r.Context(), s.channelRequestFromBroadcast(channel, broadcast, now, baseURL)); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			go s.prewarmAdjacentChannelStreams(streamWindowID, id)
		}
		if viewerPrograms.Current != nil {
			current := s.tuneProgramResponse(baseURL, *viewerPrograms.Current, broadcast)
			response.Program = &current
		}
		if viewerPrograms.Next != nil {
			next := tuneNextProgramResponse(*viewerPrograms.Next)
			response.NextProgram = &next
		}
	}

	writeJSON(w, http.StatusOK, response)
	hasProgram := response.Program != nil
	log.Printf("event=tune_latency channel=%d streaming_mode=%q has_program=%t latency_ms=%d", id, mode, hasProgram, time.Since(startedAt).Milliseconds())
}

func (s *Server) prewarmAdjacentChannelStreams(windowID uint64, currentChannelID int64) {
	channels, err := s.db.Channels(context.Background())
	if err != nil {
		log.Printf("event=stream_prewarm_channels_failed channel=%d err=%v", currentChannelID, err)
		return
	}
	channels = airingChannels(channels)
	currentIndex := -1
	for index, channel := range channels {
		if channel.ID == currentChannelID {
			currentIndex = index
			break
		}
	}
	if currentIndex < 0 || len(channels) == 0 {
		return
	}

	channelIDs := []int64{currentChannelID}
	seen := map[int64]struct{}{currentChannelID: {}}
	const radius = 2
	for distance := 1; distance <= radius; distance++ {
		for _, direction := range []int{-1, 1} {
			index := (currentIndex + direction*distance + len(channels)) % len(channels)
			channelID := channels[index].ID
			if _, exists := seen[channelID]; exists {
				continue
			}
			seen[channelID] = struct{}{}
			channelIDs = append(channelIDs, channelID)
		}
	}

	s.streamWindowMu.Lock()
	if s.streamWindowID != windowID {
		s.streamWindowMu.Unlock()
		return
	}
	s.sessions.RetainChannels(context.Background(), channelIDs)
	s.streamWindowMu.Unlock()

	for _, channelID := range channelIDs[1:] {
		s.streamWindowMu.Lock()
		current := s.streamWindowID == windowID
		s.streamWindowMu.Unlock()
		if !current {
			return
		}
		request, ok, err := s.CurrentChannelRequest(context.Background(), channelID, time.Now().UTC())
		if err != nil || !ok {
			if err != nil {
				log.Printf("event=stream_prewarm_lookup_failed channel=%d err=%v", channelID, err)
			}
			continue
		}
		if err := s.sessions.EnsureChannel(context.Background(), request); err != nil {
			log.Printf("event=stream_prewarm_failed channel=%d err=%v", channelID, err)
		}
	}
}

type tuneResponse struct {
	StreamingMode string           `json:"streaming_mode"`
	Channel       tuneChannel      `json:"channel"`
	Program       *tuneProgram     `json:"program"`
	NextProgram   *tuneNextProgram `json:"next_program"`
}

type tuneChannel struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

type tuneProgram struct {
	ID                int64     `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Year              int       `json:"year,omitempty"`
	Rating            string    `json:"rating,omitempty"`
	Genres            []string  `json:"genres,omitempty"`
	SeasonNumber      int       `json:"season_number,omitempty"`
	EpisodeNumber     int       `json:"episode_number,omitempty"`
	HasClosedCaptions bool      `json:"has_closed_captions"`
	IsHD              bool      `json:"is_hd"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	PlayURL           string    `json:"play_url"`
	PosterURL         string    `json:"poster_url"`
	SeekPosition      int       `json:"seek_position"`
}

type tuneNextProgram struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (s *Server) tuneProgramResponse(baseURL string, viewerProgram models.ScheduledProgram, broadcast scheduler.Broadcast) tuneProgram {
	actualProgram := broadcast.Program
	playback := s.renderPlayback(baseURL, actualProgram, broadcast.SeekPosition)
	log.Printf("event=tune_playback channel=%d program=%d media_type=%q viewer_program=%d play_url=%q seek=%d", actualProgram.ChannelID, actualProgram.ID, actualProgram.MediaType, viewerProgram.ID, playback.URL, playback.SeekPosition)
	response := tuneProgram{
		ID:                viewerProgram.ID,
		Title:             programTitle(viewerProgram),
		Description:       programOverview(viewerProgram),
		HasClosedCaptions: viewerProgram.HasClosedCaptions(),
		IsHD:              viewerProgram.IsHD(),
		StartTime:         viewerProgram.StartTime,
		EndTime:           viewerProgram.EndTime,
		PlayURL:           playback.URL,
		PosterURL:         publicURL(baseURL, artworkURLPath(programPosterPath(viewerProgram))),
		SeekPosition:      playback.SeekPosition,
	}
	if viewerProgram.MediaType == "tv" {
		response.Year = viewerProgram.TVMedia.Year
		if response.Year == 0 {
			response.Year = viewerProgram.TVMedia.LocalYear
		}
		response.Rating = viewerProgram.TVMedia.Rating
		response.Genres = viewerProgram.TVMedia.Genres
		if episode, ok := scheduledEpisode(viewerProgram); ok {
			response.SeasonNumber = episode.Season
			response.EpisodeNumber = episode.Episode
		}
	} else {
		response.Year = viewerProgram.Media.Year
		if response.Year == 0 {
			response.Year = viewerProgram.Media.LocalYear
		}
		response.Rating = viewerProgram.Media.Rating
		response.Genres = viewerProgram.Media.Genres
	}
	return response
}

func tuneNextProgramResponse(program models.ScheduledProgram) tuneNextProgram {
	return tuneNextProgram{
		ID:        program.ID,
		Title:     programTitle(program),
		StartTime: program.StartTime,
		EndTime:   program.EndTime,
	}
}
