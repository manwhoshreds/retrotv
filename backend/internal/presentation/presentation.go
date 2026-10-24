package presentation

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"retrotv/internal/models"
)

type OverlayPosition string

const (
	PositionTopLeft     OverlayPosition = "top_left"
	PositionTopRight    OverlayPosition = "top_right"
	PositionBottomLeft  OverlayPosition = "bottom_left"
	PositionBottomRight OverlayPosition = "bottom_right"
)

type BroadcastPresentation struct {
	ChannelBug *ImageOverlay
	Rating     *ImageOverlay
	NowPlaying *TextOverlay
}

type ImageOverlay struct {
	AssetPath string
	Position  OverlayPosition
	Width     int
	Opacity   float64
	MarginX   int
	MarginY   int
	Duration  time.Duration
}

type TextOverlay struct {
	Text     string
	Position OverlayPosition
	Duration time.Duration
	MarginX  int
	MarginY  int
}

type Resolver struct {
	ArtworkDir     string
	RatingAssetDir string
	Logger         *log.Logger
}

func (r Resolver) Resolve(channel models.Channel, program models.ScheduledProgram, now time.Time) BroadcastPresentation {
	settings := Defaults(channel.Presentation)
	elapsed := now.Sub(program.StartTime)
	if elapsed < 0 {
		elapsed = 0
	}

	result := BroadcastPresentation{}
	if settings.ChannelBugEnabled {
		asset := strings.TrimSpace(settings.ChannelBugAssetPath)
		if asset == "" {
			asset = channel.LogoPath
		}
		if path, ok := r.validOptionalPNG(channel.ID, program.ID, "channel_bug", asset); ok {
			result.ChannelBug = &ImageOverlay{
				AssetPath: path,
				Position:  position(settings.ChannelBugPosition, PositionBottomRight),
				Width:     settings.ChannelBugWidth,
				Opacity:   settings.ChannelBugOpacity,
				MarginX:   settings.ChannelBugMarginX,
				MarginY:   settings.ChannelBugMarginY,
			}
		}
	}

	if settings.RatingOverlayEnabled {
		if rating := supportedRating(programRating(program)); rating != "" {
			remaining := remainingDuration(time.Duration(settings.RatingOverlayDurationSeconds)*time.Second, elapsed)
			if remaining > 0 {
				asset := filepath.Join(r.ratingAssetDir(), strings.ToLower(strings.ReplaceAll(rating, "-", "_"))+".png")
				if path, ok := r.validOptionalPNG(channel.ID, program.ID, "rating", asset); ok {
					result.Rating = &ImageOverlay{
						AssetPath: path,
						Position:  position(settings.RatingOverlayPosition, PositionTopLeft),
						Width:     96,
						Opacity:   1,
						MarginX:   32,
						MarginY:   32,
						Duration:  remaining,
					}
				}
			}
		}
	}

	if settings.NowPlayingEnabled {
		remaining := remainingDuration(time.Duration(settings.NowPlayingDurationSeconds)*time.Second, elapsed)
		if remaining > 0 {
			if text := nowPlayingText(program); text != "" {
				result.NowPlaying = &TextOverlay{
					Text:     text,
					Position: position(settings.NowPlayingPosition, PositionBottomLeft),
					Duration: remaining,
					MarginX:  48,
					MarginY:  48,
				}
			}
		}
	}
	return result
}

func Defaults(settings models.ChannelPresentation) models.ChannelPresentation {
	if settings.ChannelBugPosition == "" {
		settings.ChannelBugPosition = string(PositionBottomRight)
	}
	if settings.ChannelBugWidth <= 0 {
		settings.ChannelBugWidth = 160
	}
	if settings.ChannelBugOpacity <= 0 || settings.ChannelBugOpacity > 1 {
		settings.ChannelBugOpacity = 0.85
	}
	if settings.ChannelBugMarginX <= 0 {
		settings.ChannelBugMarginX = 32
	}
	if settings.ChannelBugMarginY <= 0 {
		settings.ChannelBugMarginY = 32
	}
	if settings.RatingOverlayPosition == "" {
		settings.RatingOverlayPosition = string(PositionTopLeft)
	}
	if settings.RatingOverlayDurationSeconds <= 0 {
		settings.RatingOverlayDurationSeconds = 15
	}
	if settings.NowPlayingPosition == "" {
		settings.NowPlayingPosition = string(PositionBottomLeft)
	}
	if settings.NowPlayingDurationSeconds <= 0 {
		settings.NowPlayingDurationSeconds = 8
	}
	return settings
}

func remainingDuration(window time.Duration, elapsed time.Duration) time.Duration {
	if window <= 0 || elapsed >= window {
		return 0
	}
	return window - elapsed
}

func position(value string, fallback OverlayPosition) OverlayPosition {
	trimmed := strings.TrimSpace(value)
	switch OverlayPosition(trimmed) {
	case PositionTopLeft, PositionTopRight, PositionBottomLeft, PositionBottomRight:
		return OverlayPosition(trimmed)
	default:
		return fallback
	}
}

func supportedRating(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "TV-Y", "TV-Y7", "TV-G", "TV-PG", "TV-14", "TV-MA":
		return normalized
	default:
		return ""
	}
}

func programRating(program models.ScheduledProgram) string {
	if program.MediaType == "asset" {
		return ""
	}
	if program.MediaType == "tv" {
		return program.TVMedia.Rating
	}
	return program.Media.Rating
}

func nowPlayingText(program models.ScheduledProgram) string {
	if program.MediaType == "asset" {
		return ""
	}
	if program.MediaType == "tv" {
		episode := programEpisode(program)
		if episode.Name != "" {
			if episode.Season > 0 && episode.Episode > 0 {
				return fmt.Sprintf("%s - S%02dE%02d - %s", program.TVMedia.Name, episode.Season, episode.Episode, episode.Name)
			}
			return program.TVMedia.Name + " - " + episode.Name
		}
		return program.TVMedia.Name
	}
	return program.Media.Title
}

func programEpisode(program models.ScheduledProgram) models.TVEpisode {
	episodePath := strings.TrimSpace(program.EpisodePath)
	if episodePath == "" && len(program.TVMedia.EpisodePaths) > 0 {
		episodePath = program.TVMedia.EpisodePaths[0]
	}
	for _, episode := range program.TVMedia.EpisodeMetadata {
		if episode.Path == episodePath {
			return episode
		}
	}
	return models.TVEpisode{}
}

func (r Resolver) ratingAssetDir() string {
	if r.RatingAssetDir != "" {
		return r.RatingAssetDir
	}
	return filepath.Join(r.ArtworkDir, "ratings")
}

func (r Resolver) validOptionalPNG(channelID int64, programID int64, overlayType string, assetPath string) (string, bool) {
	path := r.resolveAssetPath(assetPath)
	if path == "" {
		r.logf("presentation overlay omitted: channel=%d program=%d overlay=%s asset=%q error=missing asset path", channelID, programID, overlayType, assetPath)
		return "", false
	}
	file, err := os.Open(path)
	if err != nil {
		r.logf("presentation overlay omitted: channel=%d program=%d overlay=%s asset=%q error=%v", channelID, programID, overlayType, path, err)
		return "", false
	}
	defer file.Close()
	_, format, err := image.DecodeConfig(file)
	if err != nil {
		r.logf("presentation overlay omitted: channel=%d program=%d overlay=%s asset=%q error=%v", channelID, programID, overlayType, path, err)
		return "", false
	}
	if format != "png" {
		r.logf("presentation overlay omitted: channel=%d program=%d overlay=%s asset=%q error=unsupported image format %q", channelID, programID, overlayType, path, format)
		return "", false
	}
	return path, true
}

func (r Resolver) resolveAssetPath(assetPath string) string {
	assetPath = strings.TrimSpace(assetPath)
	if assetPath == "" {
		return ""
	}
	if filepath.IsAbs(assetPath) && !strings.HasPrefix(assetPath, "/artwork/") {
		return filepath.Clean(assetPath)
	}
	if strings.HasPrefix(assetPath, "/artwork/") {
		return filepath.Join(r.ArtworkDir, strings.TrimPrefix(assetPath, "/artwork/"))
	}
	return filepath.Clean(assetPath)
}

func (r Resolver) logf(format string, args ...any) {
	if r.Logger != nil {
		r.Logger.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}
