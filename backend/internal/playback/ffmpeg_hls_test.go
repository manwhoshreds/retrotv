package playback

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"retrotv/internal/presentation"
)

func TestFFmpegHLSStarterBuildsSeekedHLSCommand(t *testing.T) {
	starter := FFmpegHLSStarter{OutputDir: "/tmp/retrotv-streams"}
	args := starter.ffmpegArgs(ChannelRequest{
		ChannelID: 7,
		Request: Request{
			Program:      Program{SourcePath: "/media/movie.mkv"},
			SeekPosition: 600,
			Presentation: presentation.BroadcastPresentation{
				NowPlaying: &presentation.TextOverlay{
					Text:     "Movie: Pilot",
					Position: presentation.PositionBottomLeft,
					Duration: 5 * time.Second,
				},
			},
		},
	}, "/media/movie.mkv", "/tmp/retrotv-streams/channels/7/index.m3u8", "/tmp/retrotv-streams/channels/7/segment-abc-%06d.ts")

	command := strings.Join(args, " ")
	for _, expected := range []string{
		"-ss 600",
		"-re -readrate_initial_burst 4 -i /media/movie.mkv",
		"-filter_complex",
		"drawtext=text='Movie\\: Pilot'",
		"-map [vout] -map 0:a?",
		"-c:v libx264 -preset ultrafast",
		"-g 60",
		"-force_key_frames expr:gte(t,n_forced*2)",
		"-c:a copy",
		"-f hls",
		"-hls_time 2",
		"-hls_list_size 6",
		"-hls_segment_filename /tmp/retrotv-streams/channels/7/segment-abc-%06d.ts",
		"/tmp/retrotv-streams/channels/7/index.m3u8",
	} {
		if !strings.Contains(command, expected) {
			t.Fatalf("command missing %q:\n%s", expected, command)
		}
	}
}

func TestFFmpegHLSStarterUsesConfiguredHLSSettings(t *testing.T) {
	starter := FFmpegHLSStarter{
		OutputDir:       "/tmp/retrotv-streams",
		SegmentDuration: 2 * time.Second,
		InitialBurst:    6 * time.Second,
		PlaylistSize:    3,
	}
	args := starter.ffmpegArgs(ChannelRequest{
		ChannelID: 7,
		Request: Request{
			Program: Program{SourcePath: "/media/movie.mkv"},
		},
	}, "/media/movie.mkv", "/tmp/retrotv-streams/channels/7/index.m3u8", "/tmp/retrotv-streams/channels/7/segment-abc-%06d.ts")

	command := strings.Join(args, " ")
	if !strings.Contains(command, "-readrate_initial_burst 6") || !strings.Contains(command, "-hls_time 2") || !strings.Contains(command, "-hls_list_size 3") {
		t.Fatalf("command = %s", command)
	}
	if !strings.Contains(command, "-c:v copy -c:a copy") {
		t.Fatalf("unfiltered command should stream-copy media: %s", command)
	}
}

func TestFFmpegHLSStarterUsesConfiguredEncoderForPresentation(t *testing.T) {
	starter := FFmpegHLSStarter{
		OutputDir:    "/tmp/retrotv-streams",
		VideoEncoder: "h264_nvenc",
		VideoPreset:  "p1",
	}
	args := starter.ffmpegArgs(ChannelRequest{
		ChannelID: 7,
		Request: Request{
			Program: Program{SourcePath: "/media/movie.mkv"},
			Presentation: presentation.BroadcastPresentation{
				NowPlaying: &presentation.TextOverlay{Text: "Movie"},
			},
		},
	}, "/media/movie.mkv", "/tmp/retrotv-streams/channels/7/index.m3u8", "/tmp/retrotv-streams/channels/7/segment-abc-%06d.ts")

	command := strings.Join(args, " ")
	if !strings.Contains(command, "-c:v h264_nvenc -preset p1") {
		t.Fatalf("command = %s", command)
	}
	if strings.Contains(command, "-tune zerolatency") {
		t.Fatalf("libx264-only options leaked into hardware command: %s", command)
	}
}

func TestPlayableHLSPlaylistRequiresSegmentFile(t *testing.T) {
	root := t.TempDir()
	playlist := filepath.Join(root, "index.m3u8")
	if err := os.WriteFile(playlist, []byte("#EXTM3U\n#EXTINF:4,\nsegment-1.ts\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if playableHLSPlaylist(playlist, root) {
		t.Fatal("playlist without segment is playable")
	}
	if err := os.WriteFile(filepath.Join(root, "segment-1.ts"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !playableHLSPlaylist(playlist, root) {
		t.Fatal("playlist with segment is not playable")
	}
}

func TestFFmpegHLSStarterPlaybackURLIsStableChannelPlaylist(t *testing.T) {
	starter := FFmpegHLSStarter{OutputDir: "/tmp/retrotv-streams"}
	process := &ffmpegHLSProcess{
		playback: Playback{
			URL: "http://retrotv.test/streams/channels/7/index.m3u8",
		},
	}
	if process.Playback().URL != "http://retrotv.test/streams/channels/7/index.m3u8" {
		t.Fatalf("playback = %#v", process.Playback())
	}
	_ = starter
}
