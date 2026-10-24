package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	MediaPath     string        `json:"media_path"`
	TVMediaPath   string        `json:"tv_media_path"`
	MediaSources  []MediaSource `json:"media_sources"`
	AssetSources  []AssetSource `json:"asset_sources"`
	StreamingMode string        `json:"streaming_mode"`
}

type MediaSource struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
}

type AssetSource struct {
	Path  string `json:"path"`
	Scope string `json:"scope"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func DefaultSettings() Settings {
	settings := Settings{
		MediaPath:     filepath.Join("data", "movies"),
		TVMediaPath:   filepath.Join("data", "tvshows"),
		AssetSources:  []AssetSource{{Path: filepath.Join("data", "assets"), Scope: AssetScopeGlobal}},
		StreamingMode: StreamingModeDirect,
	}
	settings.MediaSources = []MediaSource{
		{Path: settings.MediaPath, ContentType: "movie"},
		{Path: settings.TVMediaPath, ContentType: "tv"},
	}
	return settings
}

const (
	StreamingModeDirect = "direct"
	StreamingModeHLS    = "hls"
	AssetScopeGlobal    = "global"
)

func (s *Store) Load() (Settings, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	defer file.Close()

	settings := DefaultSettings()
	if err := json.NewDecoder(file).Decode(&settings); err != nil {
		return Settings{}, err
	}
	if settings.MediaPath == "" {
		settings.MediaPath = DefaultSettings().MediaPath
	}
	if settings.TVMediaPath == "" {
		settings.TVMediaPath = DefaultSettings().TVMediaPath
	}
	settings.MediaSources = normalizeMediaSources(settings)
	settings.AssetSources = normalizeAssetSources(settings.AssetSources)
	settings.StreamingMode = NormalizeStreamingMode(settings.StreamingMode)

	return settings, nil
}

func (s *Store) Save(settings Settings) error {
	settings.MediaSources = normalizeMediaSources(settings)
	settings.AssetSources = normalizeAssetSources(settings.AssetSources)
	settings.StreamingMode = NormalizeStreamingMode(settings.StreamingMode)
	if len(settings.MediaSources) == 0 {
		return fmt.Errorf("at least one media source is required")
	}
	settings.MediaPath = firstSourcePath(settings.MediaSources, "movie", settings.MediaPath)
	settings.TVMediaPath = firstSourcePath(settings.MediaSources, "tv", settings.TVMediaPath)

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	tempPath := s.path + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(settings)
	closeErr := file.Close()
	if encodeErr != nil {
		_ = os.Remove(tempPath)
		return encodeErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return closeErr
	}

	return os.Rename(tempPath, s.path)
}

func NormalizeStreamingMode(mode string) string {
	switch mode {
	case StreamingModeHLS:
		return StreamingModeHLS
	case StreamingModeDirect:
		return StreamingModeDirect
	default:
		return StreamingModeDirect
	}
}

func normalizeMediaSources(settings Settings) []MediaSource {
	sources := []MediaSource{}
	seen := map[MediaSource]struct{}{}
	for _, source := range settings.MediaSources {
		source.Path = filepath.Clean(source.Path)
		if source.Path == "." || source.Path == "" {
			continue
		}
		switch source.ContentType {
		case "movie", "tv":
		default:
			continue
		}
		if _, ok := seen[source]; ok {
			continue
		}
		seen[source] = struct{}{}
		sources = append(sources, source)
	}
	if len(sources) == 0 {
		if settings.MediaPath != "" {
			sources = append(sources, MediaSource{Path: filepath.Clean(settings.MediaPath), ContentType: "movie"})
		}
		if settings.TVMediaPath != "" {
			sources = append(sources, MediaSource{Path: filepath.Clean(settings.TVMediaPath), ContentType: "tv"})
		}
	}
	return sources
}

func normalizeAssetSources(sources []AssetSource) []AssetSource {
	normalized := []AssetSource{}
	seen := map[AssetSource]struct{}{}
	for _, source := range sources {
		source.Path = filepath.Clean(source.Path)
		if source.Path == "." || source.Path == "" {
			continue
		}
		if source.Scope == "" {
			source.Scope = AssetScopeGlobal
		}
		switch source.Scope {
		case AssetScopeGlobal:
		default:
			continue
		}
		if _, ok := seen[source]; ok {
			continue
		}
		seen[source] = struct{}{}
		normalized = append(normalized, source)
	}
	if len(normalized) == 0 {
		return []AssetSource{{Path: filepath.Join("data", "assets"), Scope: AssetScopeGlobal}}
	}
	return normalized
}

func firstSourcePath(sources []MediaSource, contentType string, fallback string) string {
	for _, source := range sources {
		if source.ContentType == contentType {
			return source.Path
		}
	}
	return fallback
}
