package playback

import (
	"fmt"
	"net/url"
	"strings"

	"retrotv/internal/presentation"
)

const (
	MediaTypeMovie = "movie"
	MediaTypeTV    = "tv"
	MediaTypeAsset = "asset"
)

type Program struct {
	MediaType   string
	MovieID     int64
	TVShowID    int64
	AssetID     int64
	EpisodePath string
	SourcePath  string
}

type Request struct {
	ChannelID    int64
	Program      Program
	SeekPosition int
	Presentation presentation.BroadcastPresentation
}

type Playback struct {
	URL          string
	SeekPosition int
}

type Renderer interface {
	Render(baseURL string, request Request) Playback
}

type DirectFileRenderer struct{}

func (DirectFileRenderer) Render(baseURL string, request Request) Playback {
	baseURL = strings.TrimRight(baseURL, "/")
	program := request.Program
	if program.MediaType == MediaTypeTV {
		playURL := fmt.Sprintf("%s/media/tvshows/%d", baseURL, program.TVShowID)
		if program.EpisodePath != "" {
			playURL += "?episode_path=" + url.QueryEscape(program.EpisodePath)
		}
		return Playback{URL: playURL, SeekPosition: request.SeekPosition}
	}
	if program.MediaType == MediaTypeAsset {
		return Playback{
			URL:          fmt.Sprintf("%s/media/assets/%d", baseURL, program.AssetID),
			SeekPosition: request.SeekPosition,
		}
	}
	return Playback{
		URL:          fmt.Sprintf("%s/media/movies/%d", baseURL, program.MovieID),
		SeekPosition: request.SeekPosition,
	}
}
