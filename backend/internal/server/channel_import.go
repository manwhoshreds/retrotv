package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"retrotv/internal/models"
)

const maxChannelImportSize = 32 << 20

type importedChannel struct {
	Number      int
	Name        string
	Description string
	ContentType string
	LogoHref    string
	Rules       []models.ChannelRule
}

type rtvChannelBundle struct {
	Channels []rtvChannel `json:"channels"`
}

type rtvChannel struct {
	Number      int                  `json:"number"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	ContentType string               `json:"content_type"`
	Logo        string               `json:"logo"`
	LogoPath    string               `json:"logo_path"`
	Rules       []models.ChannelRule `json:"rules"`
}

type channelImportResult struct {
	Imported int              `json:"imported"`
	Created  int              `json:"created"`
	Updated  int              `json:"updated"`
	Channels []models.Channel `json:"channels"`
}

type importedScheduleChannel struct {
	ID   int64
	Name string
}

func (s *Server) handleChannelImportStatus(w http.ResponseWriter, r *http.Request) {
	s.scheduleMu.RLock()
	status := s.scheduleStatus
	s.scheduleMu.RUnlock()
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleExportChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.db.Channels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(channels) == 0 {
		writeError(w, http.StatusBadRequest, "there are no channels to export")
		return
	}

	var buffer bytes.Buffer
	if err := s.writeChannelExportBundle(&buffer, channels); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="retrotv-channels.rtv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buffer.Bytes())
}

func (s *Server) handleImportChannels(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxChannelImportSize)
	file, header, err := r.FormFile("channels")
	if err != nil {
		writeError(w, http.StatusBadRequest, "RetroTV .rtv channel bundle is required")
		return
	}
	defer file.Close()
	if header != nil && !strings.EqualFold(filepath.Ext(header.Filename), ".rtv") {
		writeError(w, http.StatusBadRequest, "channel bundle must use the .rtv extension")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	imported, zipReader, err := parseRTVChannels(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.importChannels(r.Context(), imported, zipReader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) importChannels(ctx context.Context, imported []importedChannel, zipReader *zip.Reader) (channelImportResult, error) {
	existing, err := s.db.Channels(ctx)
	if err != nil {
		return channelImportResult{}, err
	}
	existingByName := map[string]models.Channel{}
	for _, channel := range existing {
		existingByName[strings.ToLower(channel.Name)] = channel
	}

	result := channelImportResult{}
	importedIDs := []int64{}
	scheduleChannels := []importedScheduleChannel{}
	for _, source := range imported {
		channel := models.Channel{
			Name:        source.Name,
			Description: source.Description,
			ContentType: source.ContentType,
			Rules:       source.Rules,
		}
		if existing, ok := existingByName[strings.ToLower(source.Name)]; ok {
			channel.ID = existing.ID
			channel.LogoPath = existing.LogoPath
			result.Updated++
		} else {
			result.Created++
		}

		saved, err := s.db.SaveChannel(ctx, channel)
		if err != nil {
			return channelImportResult{}, err
		}
		if source.LogoHref != "" {
			saved, err = s.saveImportedChannelLogo(ctx, saved, source.LogoHref, zipReader)
			if err != nil {
				return channelImportResult{}, err
			}
		}
		importedIDs = append(importedIDs, saved.ID)
		scheduleChannels = append(scheduleChannels, importedScheduleChannel{ID: saved.ID, Name: saved.Name})
		result.Imported++
	}

	allChannels, err := s.db.Channels(ctx)
	if err != nil {
		return channelImportResult{}, err
	}
	reorderedIDs := reorderImportedChannelIDs(allChannels, importedIDs)
	if len(reorderedIDs) > 0 {
		if err := s.db.ReorderChannels(ctx, reorderedIDs); err != nil {
			return channelImportResult{}, err
		}
		allChannels, err = s.db.Channels(ctx)
		if err != nil {
			return channelImportResult{}, err
		}
	}
	result.Channels = allChannels
	if s.schedulerRunning && len(scheduleChannels) > 0 {
		s.startImportedScheduleGeneration(scheduleChannels)
	}
	return result, nil
}

func (s *Server) writeChannelExportBundle(writer io.Writer, channels []models.Channel) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	bundle := rtvChannelBundle{Channels: make([]rtvChannel, 0, len(channels))}
	for index, channel := range channels {
		exported := rtvChannel{
			Number:      index + 1,
			Name:        channel.Name,
			Description: channel.Description,
			ContentType: channel.ContentType,
			Rules:       channel.Rules,
		}
		if channel.LogoPath != "" {
			logoPath := s.channelLogoFilesystemPath(channel.LogoPath)
			if logoPath != "" {
				logoName := exportLogoName(index+1, logoPath)
				if err := addFileToZip(zipWriter, logoName, logoPath); err == nil {
					exported.Logo = logoName
				}
			}
		}
		bundle.Channels = append(bundle.Channels, exported)
	}

	content, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	file, err := zipWriter.Create("channels.json")
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		return err
	}
	return nil
}

func (s *Server) channelLogoFilesystemPath(logoPath string) string {
	logoPath = strings.TrimSpace(logoPath)
	if logoPath == "" {
		return ""
	}
	if filepath.IsAbs(logoPath) && !strings.HasPrefix(logoPath, "/artwork/") {
		return logoPath
	}
	if strings.HasPrefix(logoPath, "/artwork/") {
		return filepath.Join(s.artworkDir, strings.TrimPrefix(logoPath, "/artwork/"))
	}
	return logoPath
}

func exportLogoName(number int, sourcePath string) string {
	ext := strings.ToLower(filepath.Ext(sourcePath))
	if !validImportedLogoExtension(ext) {
		ext = ".png"
	}
	return fmt.Sprintf("logos/%03d%s", number, ext)
}

func addFileToZip(zipWriter *zip.Writer, name string, sourcePath string) error {
	input, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()

	file, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, input)
	return err
}

func (s *Server) startImportedScheduleGeneration(channels []importedScheduleChannel) {
	runID := s.beginScheduleGeneration(len(channels))
	go func() {
		errors := []string{}
		for index, channel := range channels {
			s.updateScheduleGeneration(runID, index+1, channel.Name, errors)
			if err := s.scheduler.RegenerateChannel(context.Background(), channel.ID); err != nil {
				errors = append(errors, channel.Name+": "+err.Error())
				s.updateScheduleGeneration(runID, index+1, channel.Name, errors)
			}
		}
		s.finishScheduleGeneration(runID, errors)
	}()
}

func (s *Server) beginScheduleGeneration(total int) int64 {
	now := time.Now().UTC()
	s.scheduleMu.Lock()
	defer s.scheduleMu.Unlock()
	s.scheduleRunID++
	s.scheduleStatus = scheduleGenerationStatus{
		Active:     true,
		Phase:      "Generating channel schedules",
		Current:    "Queued imported channels",
		TotalItems: total,
		StartedAt:  &now,
		UpdatedAt:  &now,
	}
	return s.scheduleRunID
}

func (s *Server) updateScheduleGeneration(runID int64, currentItem int, current string, errors []string) {
	now := time.Now().UTC()
	s.scheduleMu.Lock()
	defer s.scheduleMu.Unlock()
	if runID != s.scheduleRunID {
		return
	}
	s.scheduleStatus.Active = true
	s.scheduleStatus.Phase = "Generating channel schedules"
	s.scheduleStatus.CurrentItem = currentItem
	s.scheduleStatus.Current = current
	s.scheduleStatus.Errors = len(errors)
	s.scheduleStatus.ErrorDetails = append([]string(nil), errors...)
	s.scheduleStatus.UpdatedAt = &now
}

func (s *Server) finishScheduleGeneration(runID int64, errors []string) {
	now := time.Now().UTC()
	s.scheduleMu.Lock()
	defer s.scheduleMu.Unlock()
	if runID != s.scheduleRunID {
		return
	}
	s.scheduleStatus.Active = false
	s.scheduleStatus.CurrentItem = s.scheduleStatus.TotalItems
	s.scheduleStatus.Current = "Channel schedules ready"
	s.scheduleStatus.Errors = len(errors)
	s.scheduleStatus.ErrorDetails = append([]string(nil), errors...)
	s.scheduleStatus.UpdatedAt = &now
	s.scheduleStatus.FinishedAt = &now
}

func reorderImportedChannelIDs(channels []models.Channel, importedIDs []int64) []int64 {
	importedSet := map[int64]struct{}{}
	for _, id := range importedIDs {
		importedSet[id] = struct{}{}
	}
	ids := make([]int64, 0, len(channels))
	for _, channel := range channels {
		if _, ok := importedSet[channel.ID]; ok {
			continue
		}
		ids = append(ids, channel.ID)
	}
	return append(ids, importedIDs...)
}

func (s *Server) saveImportedChannelLogo(ctx context.Context, channel models.Channel, href string, zipReader *zip.Reader) (models.Channel, error) {
	href = cleanBundlePath(href)
	if href == "" {
		return channel, nil
	}
	var source *zip.File
	for _, file := range zipReader.File {
		if cleanBundlePath(file.Name) == href {
			source = file
			break
		}
	}
	if source == nil {
		return channel, nil
	}
	ext := strings.ToLower(filepath.Ext(source.Name))
	if !validImportedLogoExtension(ext) {
		return channel, nil
	}
	destinationDir := filepath.Join(s.artworkDir, "channels", strconv.FormatInt(channel.ID, 10))
	if err := os.MkdirAll(destinationDir, 0o755); err != nil {
		return channel, err
	}
	destination := filepath.Join(destinationDir, "logo"+ext)
	input, err := source.Open()
	if err != nil {
		return channel, err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return channel, err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return channel, err
	}
	if err := output.Close(); err != nil {
		return channel, err
	}
	channel.LogoPath = "/artwork/channels/" + strconv.FormatInt(channel.ID, 10) + "/logo" + ext
	return s.db.SaveChannel(ctx, channel)
}

func parseRTVChannels(data []byte) ([]importedChannel, *zip.Reader, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, errors.New("invalid .rtv file")
	}
	content, err := readZipFile(zipReader, "channels.json")
	if err != nil {
		return nil, nil, errors.New(".rtv bundle must contain channels.json")
	}
	channels, err := parseRTVChannelsJSON(content, zipReader)
	if err != nil {
		return nil, nil, err
	}
	if len(channels) == 0 {
		return nil, nil, errors.New("channels.json did not contain any importable channels")
	}
	return channels, zipReader, nil
}

func parseRTVChannelsJSON(content []byte, zipReader *zip.Reader) ([]importedChannel, error) {
	var bundle rtvChannelBundle
	if err := json.Unmarshal(content, &bundle); err != nil {
		var channels []rtvChannel
		if arrayErr := json.Unmarshal(content, &channels); arrayErr != nil {
			return nil, fmt.Errorf("invalid channels.json: %w", err)
		}
		bundle.Channels = channels
	}

	imported := make([]importedChannel, 0, len(bundle.Channels))
	seenNames := map[string]struct{}{}
	for index, channel := range bundle.Channels {
		parsed, err := importedChannelFromRTV(channel, index+1, zipReader)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(parsed.Name)
		if _, ok := seenNames[key]; ok {
			return nil, fmt.Errorf("channels.json contains duplicate channel name %q", parsed.Name)
		}
		seenNames[key] = struct{}{}
		imported = append(imported, parsed)
	}
	slices.SortFunc(imported, func(a, b importedChannel) int {
		return a.Number - b.Number
	})
	return imported, nil
}

func importedChannelFromRTV(channel rtvChannel, fallbackNumber int, zipReader *zip.Reader) (importedChannel, error) {
	name := strings.TrimSpace(channel.Name)
	if name == "" {
		return importedChannel{}, fmt.Errorf("channel %d is missing name", fallbackNumber)
	}
	description := strings.TrimSpace(channel.Description)
	contentType := strings.ToLower(strings.TrimSpace(channel.ContentType))
	if contentType == "" {
		contentType = "mixed"
	}
	switch contentType {
	case "movie", "tv", "mixed":
	default:
		return importedChannel{}, fmt.Errorf("channel %q has invalid content_type %q", name, contentType)
	}
	number := channel.Number
	if number <= 0 {
		number = fallbackNumber
	}
	for i := range channel.Rules {
		channel.Rules[i].MediaType = strings.TrimSpace(channel.Rules[i].MediaType)
		channel.Rules[i].Field = strings.TrimSpace(channel.Rules[i].Field)
		channel.Rules[i].Operator = strings.TrimSpace(channel.Rules[i].Operator)
		channel.Rules[i].Value = strings.TrimSpace(channel.Rules[i].Value)
		if !validRule(channel.Rules[i]) {
			return importedChannel{}, fmt.Errorf("channel %q has an invalid rule", name)
		}
	}

	logo := strings.TrimSpace(channel.Logo)
	if logo == "" {
		logo = strings.TrimSpace(channel.LogoPath)
	}
	if logo == "" {
		logo = defaultRTVLogoPath(number, zipReader)
	}
	return importedChannel{
		Number:      number,
		Name:        name,
		Description: description,
		ContentType: contentType,
		LogoHref:    logo,
		Rules:       channel.Rules,
	}, nil
}

func defaultRTVLogoPath(number int, zipReader *zip.Reader) string {
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif"} {
		name := fmt.Sprintf("logos/%03d%s", number, ext)
		if zipHasPath(zipReader, name) {
			return name
		}
	}
	return ""
}

func zipHasPath(zipReader *zip.Reader, name string) bool {
	name = cleanBundlePath(name)
	for _, file := range zipReader.File {
		if cleanBundlePath(file.Name) == name {
			return true
		}
	}
	return false
}

func readZipFile(zipReader *zip.Reader, name string) ([]byte, error) {
	name = cleanBundlePath(name)
	for _, file := range zipReader.File {
		if cleanBundlePath(file.Name) != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	}
	return nil, os.ErrNotExist
}

func cleanBundlePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "/")
	if path == "." || strings.HasPrefix(path, "../") {
		return ""
	}
	return path
}

func validImportedLogoExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	default:
		return false
	}
}
