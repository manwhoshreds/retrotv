package playback

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FFmpegHLSStarter struct {
	FFmpegPath      string
	OutputDir       string
	SegmentDuration time.Duration
	InitialBurst    time.Duration
	PlaylistSize    int
	StartupTimeout  time.Duration
	PollInterval    time.Duration
	VideoEncoder    string
	VideoPreset     string
	Logger          *log.Logger
}

func (s FFmpegHLSStarter) StartSession(ctx context.Context, request ChannelRequest) (SessionProcess, error) {
	if request.ChannelID <= 0 {
		return nil, ErrInvalidChannelID
	}
	sourcePath := strings.TrimSpace(request.Request.Program.SourcePath)
	if sourcePath == "" {
		return nil, errors.New("playback source path is empty")
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return nil, fmt.Errorf("playback source %q is not available: %w", sourcePath, err)
	}
	outputDir := s.channelOutputDir(request.ChannelID)
	if err := os.RemoveAll(outputDir); err != nil {
		return nil, fmt.Errorf("clear channel stream directory: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create channel stream directory: %w", err)
	}

	playlistPath := filepath.Join(outputDir, "index.m3u8")
	segmentPattern := filepath.Join(outputDir, "segment-"+strconv.FormatInt(time.Now().UnixNano(), 36)+"-%06d.ts")
	args := s.ffmpegArgs(request, sourcePath, playlistPath, segmentPattern)
	cmd := exec.Command(s.ffmpegPath(), args...)
	cmd.Stdout = logWriter{s.logger(), "ffmpeg stdout: "}
	cmd.Stderr = logWriter{s.logger(), "ffmpeg stderr: "}
	s.logger().Printf("event=ffmpeg_start channel=%d program=%d source=%q seek=%d", request.ChannelID, request.ProgramID, sourcePath, request.Request.SeekPosition)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	process := &ffmpegHLSProcess{
		cmd: cmd,
		playback: Playback{
			URL:          strings.TrimRight(request.BaseURL, "/") + "/streams/channels/" + strconv.FormatInt(request.ChannelID, 10) + "/index.m3u8",
			SeekPosition: request.Request.SeekPosition,
		},
		done: make(chan error, 1),
	}
	go process.wait()
	if err := s.waitForPlayablePlaylist(ctx, process, playlistPath, outputDir); err != nil {
		_ = process.Stop(context.Background())
		return nil, err
	}
	return process, nil
}

func (s FFmpegHLSStarter) ffmpegArgs(request ChannelRequest, sourcePath string, playlistPath string, segmentPattern string) []string {
	args := []string{
		"-hide_banner",
		"-nostdin",
		"-loglevel", "warning",
	}
	if request.Request.SeekPosition > 0 {
		args = append(args, "-ss", strconv.Itoa(request.Request.SeekPosition))
	}
	args = append(args,
		"-re",
		"-readrate_initial_burst", strconv.Itoa(s.initialBurstSeconds()),
		"-i", sourcePath,
	)

	filter := BuildFFmpegPresentationFilter(request.Request.Presentation)
	for _, inputPath := range filter.InputPaths {
		args = append(args, "-i", inputPath)
	}
	if filter.FilterGraph != "" {
		args = append(args, "-filter_complex", filter.FilterGraph, "-map", "["+filter.OutputLabel+"]", "-map", "0:a?")
		args = append(args, "-c:v", s.videoEncoder())
		if preset := s.videoPreset(); preset != "" {
			args = append(args, "-preset", preset)
		}
		if s.videoEncoder() == "libx264" {
			args = append(args, "-tune", "zerolatency", "-pix_fmt", "yuv420p")
		}
		args = append(args,
			"-g", strconv.Itoa(s.segmentDurationSeconds()*30),
			"-force_key_frames", "expr:gte(t,n_forced*"+strconv.Itoa(s.segmentDurationSeconds())+")",
			"-c:a", "copy",
		)
	} else {
		// With no pixel-changing presentation filters, the pre-transcoded media
		// can be remuxed into HLS without an unnecessary decode/encode cycle.
		args = append(args, "-map", "0:v:0", "-map", "0:a?", "-c:v", "copy", "-c:a", "copy")
	}

	args = append(args,
		"-f", "hls",
		"-hls_time", strconv.Itoa(s.segmentDurationSeconds()),
		"-hls_list_size", strconv.Itoa(s.playlistSize()),
		"-hls_flags", "delete_segments+independent_segments+program_date_time+temp_file",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	)
	return args
}

func (s FFmpegHLSStarter) waitForPlayablePlaylist(ctx context.Context, process *ffmpegHLSProcess, playlistPath string, outputDir string) error {
	timeout := s.StartupTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	pollInterval := s.PollInterval
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if playableHLSPlaylist(playlistPath, outputDir) {
			return nil
		}
		select {
		case err := <-process.Done():
			if playableHLSPlaylist(playlistPath, outputDir) {
				return nil
			}
			if err == nil {
				return errors.New("ffmpeg exited before playlist was ready")
			}
			return fmt.Errorf("ffmpeg exited before playlist was ready: %w", err)
		case <-waitCtx.Done():
			return fmt.Errorf("hls playlist was not ready before timeout: %w", waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func playableHLSPlaylist(playlistPath string, outputDir string) bool {
	data, err := os.ReadFile(playlistPath)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "://") {
			continue
		}
		segmentPath := filepath.Join(outputDir, filepath.Base(line))
		info, err := os.Stat(segmentPath)
		if err == nil && info.Size() > 0 {
			return true
		}
	}
	return false
}

func (s FFmpegHLSStarter) segmentDurationSeconds() int {
	if s.SegmentDuration > 0 {
		seconds := int(s.SegmentDuration.Seconds())
		if seconds > 0 {
			return seconds
		}
	}
	return 2
}

func (s FFmpegHLSStarter) initialBurstSeconds() int {
	if s.InitialBurst > 0 {
		if seconds := int(s.InitialBurst.Seconds()); seconds > 0 {
			return seconds
		}
	}
	return 4
}

func (s FFmpegHLSStarter) videoEncoder() string {
	if encoder := strings.TrimSpace(s.VideoEncoder); encoder != "" {
		return encoder
	}
	return "libx264"
}

func (s FFmpegHLSStarter) videoPreset() string {
	if preset := strings.TrimSpace(s.VideoPreset); preset != "" {
		return preset
	}
	if strings.TrimSpace(s.VideoEncoder) == "" || s.videoEncoder() == "libx264" {
		return "ultrafast"
	}
	return ""
}

func (s FFmpegHLSStarter) playlistSize() int {
	if s.PlaylistSize > 0 {
		return s.PlaylistSize
	}
	return 6
}

func (s FFmpegHLSStarter) channelOutputDir(channelID int64) string {
	return filepath.Join(s.outputDir(), "channels", strconv.FormatInt(channelID, 10))
}

func (s FFmpegHLSStarter) outputDir() string {
	if s.OutputDir != "" {
		return s.OutputDir
	}
	return filepath.Join("data", "streams")
}

func (s FFmpegHLSStarter) ffmpegPath() string {
	if s.FFmpegPath != "" {
		return s.FFmpegPath
	}
	return "ffmpeg"
}

func (s FFmpegHLSStarter) logger() *log.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return log.Default()
}

type ffmpegHLSProcess struct {
	cmd      *exec.Cmd
	playback Playback
	done     chan error
	stopOnce sync.Once
}

func (p *ffmpegHLSProcess) Playback() Playback {
	return p.playback
}

func (p *ffmpegHLSProcess) Stop(ctx context.Context) error {
	var err error
	p.stopOnce.Do(func() {
		if p.cmd.Process == nil {
			return
		}
		err = p.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			err = p.cmd.Process.Kill()
			return
		}
		select {
		case <-p.done:
		case <-ctx.Done():
			err = p.cmd.Process.Kill()
		}
	})
	return err
}

func (p *ffmpegHLSProcess) Done() <-chan error {
	return p.done
}

func (p *ffmpegHLSProcess) wait() {
	err := p.cmd.Wait()
	p.done <- err
	close(p.done)
}

type logWriter struct {
	logger *log.Logger
	prefix string
}

func (w logWriter) Write(data []byte) (int, error) {
	message := strings.TrimSpace(string(data))
	if message != "" {
		w.logger.Print(w.prefix + message)
	}
	return len(data), nil
}
