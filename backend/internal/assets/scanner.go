package assets

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"

	"retrotv/internal/config"
	"retrotv/internal/metadata"
	"retrotv/internal/models"
)

type Prober interface {
	Probe(ctx context.Context, path string) (models.MediaMetadata, error)
}

type Scanner struct {
	Prober   Prober
	Channels []models.Channel
}

func (s Scanner) Scan(ctx context.Context, sources []config.AssetSource) ([]models.ChannelAsset, []string) {
	prober := s.Prober
	if prober == nil {
		prober = metadata.NewFFprobeProber()
	}

	assets := []models.ChannelAsset{}
	scanErrors := []string{}
	for _, source := range sources {
		source.Path = filepath.Clean(source.Path)
		if source.Path == "." || source.Path == "" {
			continue
		}
		err := filepath.WalkDir(source.Path, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				scanErrors = append(scanErrors, path+": "+walkErr.Error())
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			asset, ok := s.classify(source, path)
			if !ok {
				return nil
			}
			mediaMetadata, err := prober.Probe(ctx, path)
			if err != nil {
				scanErrors = append(scanErrors, path+": "+err.Error())
			}
			asset.MediaMetadata = mediaMetadata
			asset.DurationSeconds = int(math.Round(mediaMetadata.DurationSeconds))
			assets = append(assets, asset)
			return nil
		})
		if err != nil {
			scanErrors = append(scanErrors, source.Path+": "+err.Error())
		}
	}
	return assets, scanErrors
}

func (s Scanner) classify(source config.AssetSource, path string) (models.ChannelAsset, bool) {
	if !videoFile(path) {
		return models.ChannelAsset{}, false
	}
	relative, err := filepath.Rel(source.Path, path)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
		return models.ChannelAsset{}, false
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 2 {
		return models.ChannelAsset{}, false
	}
	if strings.EqualFold(parts[0], "channels") {
		return s.classifyChannelAsset(source, path, parts)
	}
	assetType, ok := normalizeAssetType(parts[0])
	if !ok {
		return models.ChannelAsset{}, false
	}
	tag := ""
	if len(parts) > 2 {
		tag = strings.TrimSpace(parts[1])
	}
	scope := source.Scope
	if scope == "" {
		scope = config.AssetScopeGlobal
	}
	return models.ChannelAsset{
		FilePath:  path,
		Filename:  filepath.Base(path),
		AssetType: assetType,
		Tag:       tag,
		Scope:     scope,
		Enabled:   true,
	}, true
}

func (s Scanner) classifyChannelAsset(source config.AssetSource, path string, parts []string) (models.ChannelAsset, bool) {
	if len(parts) < 4 {
		return models.ChannelAsset{}, false
	}
	channelID := s.channelIDForSlug(parts[1])
	if channelID == 0 {
		return models.ChannelAsset{}, false
	}
	assetType, ok := normalizeAssetType(parts[2])
	if !ok || assetType == "commercial" {
		return models.ChannelAsset{}, false
	}
	tag := ""
	if len(parts) > 4 {
		tag = strings.TrimSpace(parts[3])
	}
	return models.ChannelAsset{
		ChannelID: channelID,
		FilePath:  path,
		Filename:  filepath.Base(path),
		AssetType: assetType,
		Tag:       tag,
		Scope:     "channel",
		Enabled:   true,
	}, true
}

func (s Scanner) channelIDForSlug(slug string) int64 {
	for _, channel := range s.Channels {
		if channelSlug(channel.Name) == slug {
			return channel.ID
		}
	}
	return 0
}

func channelSlug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, slug)
	slug = strings.Trim(slug, "_")
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}
	return slug
}

func normalizeAssetType(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bumper", "bumpers":
		return "bumper", true
	case "commercial", "commercials":
		return "commercial", true
	case "promo", "promos":
		return "promo", true
	case "station_id", "station_ids", "station-id", "station-ids", "stationid", "stationids":
		return "station_id", true
	default:
		return "", false
	}
}

func videoFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mkv", ".avi", ".mov":
		return true
	default:
		return false
	}
}

var ErrNoAssetSources = errors.New("no asset sources configured")
