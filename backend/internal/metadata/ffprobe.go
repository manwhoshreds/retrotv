package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"retrotv/internal/models"
)

var ErrFFprobeUnavailable = errors.New("ffprobe unavailable")

type FFprobeProber struct {
	Path    string
	Timeout time.Duration
}

func NewFFprobeProber() FFprobeProber {
	return FFprobeProber{
		Path:    "ffprobe",
		Timeout: 15 * time.Second,
	}
}

func (p FFprobeProber) Probe(ctx context.Context, path string) (models.MediaMetadata, error) {
	info, err := os.Stat(path)
	if err != nil {
		return models.MediaMetadata{}, err
	}
	if p.Path == "" {
		p.Path = "ffprobe"
	}
	if p.Timeout <= 0 {
		p.Timeout = 15 * time.Second
	}

	probeCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	cmd := exec.CommandContext(
		probeCtx,
		p.Path,
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return models.MediaMetadata{}, ErrFFprobeUnavailable
		}
		if probeCtx.Err() != nil {
			return models.MediaMetadata{}, probeCtx.Err()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(string(exitErr.Stderr))
			if message != "" {
				return models.MediaMetadata{}, fmt.Errorf("ffprobe failed: %s", message)
			}
		}
		return models.MediaMetadata{}, err
	}

	metadata, err := ParseFFprobeMetadata(output)
	if err != nil {
		return models.MediaMetadata{}, err
	}
	metadata.FileSize = info.Size()
	metadata.ModTimeUnix = info.ModTime().Unix()
	return metadata, nil
}

func ParseFFprobeMetadata(data []byte) (models.MediaMetadata, error) {
	var payload ffprobePayload
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&payload); err != nil {
		return models.MediaMetadata{}, fmt.Errorf("decode ffprobe JSON: %w", err)
	}

	metadata := models.MediaMetadata{}
	if duration := parseFloat(payload.Format.Duration); duration > 0 {
		metadata.DurationSeconds = duration
	}

	for _, stream := range payload.Streams {
		switch stream.CodecType {
		case "video":
			if metadata.VideoCodec == "" {
				metadata.VideoCodec = stream.CodecName
				metadata.Width = stream.Width
				metadata.Height = stream.Height
				metadata.FrameRate = parseFrameRate(stream.AvgFrameRate)
				if metadata.FrameRate == 0 {
					metadata.FrameRate = parseFrameRate(stream.RFrameRate)
				}
			}
		case "audio":
			if metadata.AudioCodec == "" {
				metadata.AudioCodec = stream.CodecName
			}
			metadata.AudioTracks = append(metadata.AudioTracks, models.AudioTrack{
				Index:    stream.Index,
				Codec:    stream.CodecName,
				Language: stream.Tags.Language,
				Channels: stream.Channels,
			})
		case "subtitle":
			metadata.SubtitleTracks = append(metadata.SubtitleTracks, models.SubtitleTrack{
				Index:    stream.Index,
				Codec:    stream.CodecName,
				Language: stream.Tags.Language,
			})
		}
	}
	return metadata, nil
}

type ffprobePayload struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

type ffprobeStream struct {
	Index        int         `json:"index"`
	CodecName    string      `json:"codec_name"`
	CodecType    string      `json:"codec_type"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	AvgFrameRate string      `json:"avg_frame_rate"`
	RFrameRate   string      `json:"r_frame_rate"`
	Channels     int         `json:"channels"`
	Tags         ffprobeTags `json:"tags"`
}

type ffprobeTags struct {
	Language string `json:"language"`
}

func parseFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func parseFrameRate(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return parseFloat(value)
	}
	numerator := parseFloat(parts[0])
	denominator := parseFloat(parts[1])
	if numerator == 0 || denominator == 0 {
		return 0
	}
	return math.Round((numerator/denominator)*1000) / 1000
}
