package presentation

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"retrotv/internal/models"
)

func TestResolverResolvesConfiguredChannelBug(t *testing.T) {
	root := t.TempDir()
	bugPath := filepath.Join(root, "channels", "7", "bug.png")
	writePNG(t, bugPath)

	resolver := Resolver{ArtworkDir: root, Logger: log.New(&bytes.Buffer{}, "", 0)}
	program := movieProgram("TV-PG", time.Now().UTC().Add(-time.Minute))
	result := resolver.Resolve(models.Channel{
		ID:       7,
		LogoPath: "/artwork/channels/7/logo.png",
		Presentation: models.ChannelPresentation{
			ChannelBugEnabled:   true,
			ChannelBugAssetPath: "/artwork/channels/7/bug.png",
			ChannelBugPosition:  "top_right",
			ChannelBugWidth:     120,
			ChannelBugOpacity:   0.5,
			ChannelBugMarginX:   12,
			ChannelBugMarginY:   14,
		},
	}, program, time.Now().UTC())

	if result.ChannelBug == nil {
		t.Fatal("ChannelBug = nil")
	}
	if result.ChannelBug.AssetPath != bugPath ||
		result.ChannelBug.Position != PositionTopRight ||
		result.ChannelBug.Width != 120 ||
		result.ChannelBug.Opacity != 0.5 ||
		result.ChannelBug.MarginX != 12 ||
		result.ChannelBug.MarginY != 14 {
		t.Fatalf("ChannelBug = %#v", result.ChannelBug)
	}
}

func TestResolverDisablesChannelBug(t *testing.T) {
	root := t.TempDir()
	writePNG(t, filepath.Join(root, "channels", "7", "logo.png"))

	resolver := Resolver{ArtworkDir: root, Logger: log.New(&bytes.Buffer{}, "", 0)}
	result := resolver.Resolve(models.Channel{
		ID:       7,
		LogoPath: "/artwork/channels/7/logo.png",
	}, movieProgram("TV-PG", time.Now().UTC()), time.Now().UTC())

	if result.ChannelBug != nil {
		t.Fatalf("ChannelBug = %#v, want nil", result.ChannelBug)
	}
}

func TestResolverMapsRatingAssetAndRemainingDuration(t *testing.T) {
	root := t.TempDir()
	writePNG(t, filepath.Join(root, "ratings", "tv_pg.png"))
	start := time.Now().UTC().Add(-5 * time.Second)

	resolver := Resolver{ArtworkDir: root, Logger: log.New(&bytes.Buffer{}, "", 0)}
	result := resolver.Resolve(models.Channel{
		ID: 7,
		Presentation: models.ChannelPresentation{
			RatingOverlayEnabled:         true,
			RatingOverlayPosition:        "bottom_left",
			RatingOverlayDurationSeconds: 15,
		},
	}, movieProgram("TV-PG", start), start.Add(5*time.Second))

	if result.Rating == nil {
		t.Fatal("Rating = nil")
	}
	if result.Rating.AssetPath != filepath.Join(root, "ratings", "tv_pg.png") ||
		result.Rating.Position != PositionBottomLeft ||
		result.Rating.Duration != 10*time.Second {
		t.Fatalf("Rating = %#v", result.Rating)
	}
}

func TestResolverOmitsExpiredOrUnsupportedRating(t *testing.T) {
	root := t.TempDir()
	writePNG(t, filepath.Join(root, "ratings", "tv_ma.png"))
	resolver := Resolver{ArtworkDir: root, Logger: log.New(&bytes.Buffer{}, "", 0)}
	channel := models.Channel{
		ID: 7,
		Presentation: models.ChannelPresentation{
			RatingOverlayEnabled:         true,
			RatingOverlayDurationSeconds: 15,
		},
	}

	start := time.Now().UTC().Add(-20 * time.Second)
	if result := resolver.Resolve(channel, movieProgram("TV-MA", start), time.Now().UTC()); result.Rating != nil {
		t.Fatalf("expired Rating = %#v, want nil", result.Rating)
	}

	start = time.Now().UTC()
	if result := resolver.Resolve(channel, movieProgram("R", start), start); result.Rating != nil {
		t.Fatalf("unsupported Rating = %#v, want nil", result.Rating)
	}
}

func TestResolverNowPlayingRemainingDurationAndTVEpisodeText(t *testing.T) {
	start := time.Now().UTC().Add(-3 * time.Second)
	resolver := Resolver{Logger: log.New(&bytes.Buffer{}, "", 0)}
	program := models.ScheduledProgram{
		ID:          1,
		MediaType:   "tv",
		EpisodePath: "/tv/show/S01E02.mkv",
		StartTime:   start,
		TVMedia: models.TVShowRecord{
			TVShow: models.TVShow{Name: "Martin"},
			EpisodeMetadata: []models.TVEpisode{{
				Path:    "/tv/show/S01E02.mkv",
				Season:  1,
				Episode: 2,
				Name:    "The Gift Rapper",
			}},
		},
	}
	result := resolver.Resolve(models.Channel{
		ID: 7,
		Presentation: models.ChannelPresentation{
			NowPlayingEnabled:         true,
			NowPlayingDurationSeconds: 8,
		},
	}, program, start.Add(3*time.Second))

	if result.NowPlaying == nil {
		t.Fatal("NowPlaying = nil")
	}
	if result.NowPlaying.Text != "Martin - S01E02 - The Gift Rapper" ||
		result.NowPlaying.Duration != 5*time.Second {
		t.Fatalf("NowPlaying = %#v", result.NowPlaying)
	}

	expired := resolver.Resolve(models.Channel{
		ID: 7,
		Presentation: models.ChannelPresentation{
			NowPlayingEnabled:         true,
			NowPlayingDurationSeconds: 8,
		},
	}, program, start.Add(9*time.Second))
	if expired.NowPlaying != nil {
		t.Fatalf("expired NowPlaying = %#v, want nil", expired.NowPlaying)
	}
}

func TestResolverOmitsMissingOptionalAssetButKeepsOtherPresentation(t *testing.T) {
	start := time.Now().UTC()
	var logs bytes.Buffer
	resolver := Resolver{ArtworkDir: t.TempDir(), Logger: log.New(&logs, "", 0)}
	result := resolver.Resolve(models.Channel{
		ID:       7,
		LogoPath: "/artwork/channels/7/missing.png",
		Presentation: models.ChannelPresentation{
			ChannelBugEnabled: true,
			NowPlayingEnabled: true,
		},
	}, movieProgram("TV-PG", start), start)

	if result.ChannelBug != nil {
		t.Fatalf("ChannelBug = %#v, want nil", result.ChannelBug)
	}
	if result.NowPlaying == nil {
		t.Fatal("NowPlaying = nil")
	}
	if !bytes.Contains(logs.Bytes(), []byte("channel_bug")) {
		t.Fatalf("logs = %s", logs.String())
	}
}

func TestResolverDoesNotLeakPresentationBetweenChannels(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "channels", "1", "logo.png")
	second := filepath.Join(root, "channels", "2", "logo.png")
	writePNG(t, first)
	writePNG(t, second)
	resolver := Resolver{ArtworkDir: root, Logger: log.New(&bytes.Buffer{}, "", 0)}
	now := time.Now().UTC()
	program := movieProgram("TV-PG", now)

	firstResult := resolver.Resolve(models.Channel{ID: 1, LogoPath: "/artwork/channels/1/logo.png", Presentation: models.ChannelPresentation{ChannelBugEnabled: true}}, program, now)
	secondResult := resolver.Resolve(models.Channel{ID: 2, LogoPath: "/artwork/channels/2/logo.png", Presentation: models.ChannelPresentation{ChannelBugEnabled: true}}, program, now)

	if firstResult.ChannelBug.AssetPath != first || secondResult.ChannelBug.AssetPath != second {
		t.Fatalf("channel assets leaked: first=%#v second=%#v", firstResult.ChannelBug, secondResult.ChannelBug)
	}
}

func movieProgram(rating string, start time.Time) models.ScheduledProgram {
	return models.ScheduledProgram{
		ID:        1,
		MediaType: "movie",
		StartTime: start,
		EndTime:   start.Add(time.Hour),
		Media: models.MovieRecord{
			Movie: models.Movie{
				Title:  "Heat",
				Rating: rating,
			},
		},
	}
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}
