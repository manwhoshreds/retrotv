package playback

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"retrotv/internal/presentation"
)

func TestBuildFFmpegPresentationFilterBuildsOverlayGraph(t *testing.T) {
	filter := BuildFFmpegPresentationFilter(presentation.BroadcastPresentation{
		ChannelBug: &presentation.ImageOverlay{
			AssetPath: "/tmp/bug.png",
			Position:  presentation.PositionBottomRight,
			Width:     120,
			Opacity:   0.5,
			MarginX:   14,
			MarginY:   16,
		},
		Rating: &presentation.ImageOverlay{
			AssetPath: "/tmp/tv_pg.png",
			Position:  presentation.PositionTopLeft,
			Width:     96,
			Opacity:   1,
			MarginX:   20,
			MarginY:   22,
			Duration:  10 * time.Second,
		},
		NowPlaying: &presentation.TextOverlay{
			Text:     "Martin - S01E02 - The Gift Rapper",
			Position: presentation.PositionBottomLeft,
			Duration: 5 * time.Second,
			MarginX:  48,
			MarginY:  50,
		},
	})

	if !reflect.DeepEqual(filter.InputPaths, []string{"/tmp/bug.png", "/tmp/tv_pg.png"}) {
		t.Fatalf("InputPaths = %#v", filter.InputPaths)
	}
	if filter.OutputLabel != "vout" {
		t.Fatalf("OutputLabel = %q", filter.OutputLabel)
	}
	expectContains := []string{
		"[1:v]format=rgba[bug1]",
		"[bug1]scale=120:-1,colorchannelmixer=aa=0.5[bug_scaled1]",
		"overlay=x=main_w-overlay_w-14:y=main_h-overlay_h-16",
		"[2:v]format=rgba[rating2]",
		":enable='between(t,0,10)'",
		"drawtext=text='Martin - S01E02 - The Gift Rapper'",
		"x=48:y=main_h-text_h-50",
		":enable='between(t,0,5)'",
		"null[vout]",
	}
	for _, expected := range expectContains {
		if !strings.Contains(filter.FilterGraph, expected) {
			t.Fatalf("FilterGraph missing %q:\n%s", expected, filter.FilterGraph)
		}
	}
}

func TestBuildFFmpegPresentationFilterNoPresentationUsesInputVideo(t *testing.T) {
	filter := BuildFFmpegPresentationFilter(presentation.BroadcastPresentation{})
	if len(filter.InputPaths) != 0 || filter.FilterGraph != "" || filter.OutputLabel != "0:v" {
		t.Fatalf("filter = %#v", filter)
	}
}

func TestEscapeDrawtextEscapesSpecialCharacters(t *testing.T) {
	text := `Bob's Show: 100% [Pilot]\Back, Again`
	escaped := EscapeDrawtext(text)
	expected := `Bob\'s Show\: 100\% \[Pilot\]\\Back\, Again`
	if escaped != expected {
		t.Fatalf("escaped = %q, want %q", escaped, expected)
	}
}
