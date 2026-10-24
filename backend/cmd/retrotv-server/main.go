package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"

	"retrotv/internal/database"
	"retrotv/internal/metadata"
	"retrotv/internal/models"
	"retrotv/internal/playback"
	"retrotv/internal/scheduler"
	"retrotv/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	_ = godotenv.Load()

	if len(args) == 0 {
		return serve()
	}

	switch args[0] {
	case "serve":
		return serve()
	case "regenerate_schedules":
		return regenerateSchedules(args[1:])
	case "get_movie":
		return getMovie(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "test_package":
		testPackage()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText())
	}
}

func testPackage() {
	fmt.Println("testing")
}

func regenerateSchedules(args []string) error {
	db, err := database.Open("")
	if err != nil {
		return err
	}
	defer db.Close()

	scheduler := scheduler.New(db)
	channels, err := db.Channels(context.Background())
	if err != nil {
		return err
	}
	if len(args) > 0 {
		channelID, parseErr := strconv.ParseInt(args[0], 10, 64)
		if parseErr != nil {
			return fmt.Errorf("invalid channel id %q", args[0])
		}
		channel, channelErr := db.Channel(context.Background(), channelID)
		if channelErr != nil {
			return channelErr
		}
		channels = []models.Channel{channel}
	}
	for _, channel := range channels {
		if err := scheduler.RegenerateChannel(context.Background(), channel.ID); err != nil {
			return fmt.Errorf("regenerate channel %q: %w", channel.Name, err)
		}
		fmt.Fprintf(os.Stdout, "regenerated %q\n", channel.Name)
	}
	fmt.Fprintf(os.Stdout, "regenerated %d channel schedules\n", len(channels))
	return nil
}

func serve() error {
	addr := os.Getenv("RETROTV_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8080"
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse RETROTV_ADDR: %w", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return fmt.Errorf("parse RETROTV_ADDR port: %w", err)
	}
	streamBaseURL := os.Getenv("RETROTV_STREAM_BASE_URL")
	if streamBaseURL == "" {
		streamHost := host
		if streamHost == "" || streamHost == "0.0.0.0" || streamHost == "::" {
			streamHost, _ = os.Hostname()
			streamHost += ".local"
		}
		streamBaseURL = "http://" + net.JoinHostPort(streamHost, portText)
	}
	streamDir := os.Getenv("RETROTV_STREAM_DIR")
	if streamDir == "" {
		streamDir = "data/streams"
	}
	ffmpegPath := os.Getenv("RETROTV_FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	streamIdleTimeout := durationEnv("RETROTV_STREAM_IDLE_TIMEOUT", 5*time.Minute)
	hlsSegmentDuration := durationEnv("RETROTV_HLS_SEGMENT_DURATION", 2*time.Second)
	hlsInitialBurst := durationEnv("RETROTV_HLS_INITIAL_BURST", 4*time.Second)
	hlsPlaylistSize := intEnv("RETROTV_HLS_PLAYLIST_SIZE", 6)
	hlsStartupTimeout := durationEnv("RETROTV_HLS_STARTUP_TIMEOUT", 30*time.Second)
	hlsVideoEncoder := stringEnv("RETROTV_HLS_VIDEO_ENCODER", "libx264")
	hlsVideoPreset := stringEnv("RETROTV_HLS_VIDEO_PRESET", "")

	server, err := server.NewServer(server.Config{
		StaticDir:         "web",
		StreamBaseURL:     streamBaseURL,
		StreamDir:         streamDir,
		StreamIdleTimeout: streamIdleTimeout,
		StreamStarter: playback.FFmpegHLSStarter{
			FFmpegPath:      ffmpegPath,
			OutputDir:       streamDir,
			SegmentDuration: hlsSegmentDuration,
			InitialBurst:    hlsInitialBurst,
			PlaylistSize:    hlsPlaylistSize,
			StartupTimeout:  hlsStartupTimeout,
			VideoEncoder:    hlsVideoEncoder,
			VideoPreset:     hlsVideoPreset,
		},
	})
	if err != nil {
		return err
	}

	var mdnsServer *zeroconf.Server
	if os.Getenv("RETROTV_MDNS_DISABLED") != "true" {
		instance := stringEnv("RETROTV_MDNS_NAME", "RetroTV")
		mdnsServer, err = zeroconf.Register(instance, "_retrotv._tcp", "local.", port,
			[]string{"path=/", "version=1"}, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "RetroTV mDNS advertisement unavailable: %v\n", err)
		} else {
			defer mdnsServer.Shutdown()
			fmt.Fprintf(os.Stdout, "RetroTV advertised as %q via mDNS\n", instance)
		}
	}

	fmt.Fprintf(os.Stdout, "RetroTV backend listening at http://%s\n", addr)
	return http.ListenAndServe(addr, server.Handler())
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func stringEnv(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func intEnv(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return fallback
	}
	return number
}

func getMovie(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: retrotv get_movie <movie name> <year>")
	}

	year, err := strconv.Atoi(args[len(args)-1])
	if err != nil {
		return fmt.Errorf("year must be a number: %q", args[len(args)-1])
	}

	title := strings.TrimSpace(strings.Join(args[:len(args)-1], " "))
	downloader := metadata.NewMetadataDownloader(os.Getenv("TMDB_READ_TOKEN"))
	movie, err := downloader.GetMovie(context.Background(), title, year)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(movie)
}

func usageError() error {
	return fmt.Errorf("missing command\n\n%s", usageText())
}

func printUsage() {
	fmt.Fprintln(os.Stdout, usageText())
}

func usageText() string {
	return `Usage:
  retrotv
  retrotv serve
  retrotv regenerate_schedules
  retrotv get_movie <movie name> <year>

Examples:
  retrotv serve
  retrotv regenerate_schedules
  retrotv get_movie Die Hard 1988
  retrotv get_movie "Blade Runner 2049" 2017`
}
