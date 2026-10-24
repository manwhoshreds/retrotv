package models

import "testing"

func TestScheduledProgramHasClosedCaptions(t *testing.T) {
	captioned := MediaMetadata{SubtitleTracks: []SubtitleTrack{{Index: 2, Codec: "subrip", Language: "eng"}}}

	tests := []struct {
		name    string
		program ScheduledProgram
		want    bool
	}{
		{
			name: "captioned movie",
			program: ScheduledProgram{
				MediaType: "movie",
				Media:     MovieRecord{MediaMetadata: captioned},
			},
			want: true,
		},
		{
			name: "movie without captions",
			program: ScheduledProgram{
				MediaType: "movie",
				Media:     MovieRecord{},
			},
		},
		{
			name: "captioned scheduled episode",
			program: ScheduledProgram{
				MediaType:   "tv",
				EpisodePath: "/shows/example/s01e02.mkv",
				TVMedia: TVShowRecord{EpisodeMetadata: []TVEpisode{
					{Path: "/shows/example/s01e01.mkv"},
					{Path: "/shows/example/s01e02.mkv", MediaMetadata: captioned},
				}},
			},
			want: true,
		},
		{
			name: "different episode has captions",
			program: ScheduledProgram{
				MediaType:   "tv",
				EpisodePath: "/shows/example/s01e01.mkv",
				TVMedia: TVShowRecord{EpisodeMetadata: []TVEpisode{
					{Path: "/shows/example/s01e01.mkv"},
					{Path: "/shows/example/s01e02.mkv", MediaMetadata: captioned},
				}},
			},
		},
		{
			name: "captioned asset is excluded",
			program: ScheduledProgram{
				MediaType: "asset",
				Asset:     ChannelAsset{MediaMetadata: captioned},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.program.HasClosedCaptions(); got != test.want {
				t.Fatalf("HasClosedCaptions() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestScheduledProgramIsHD(t *testing.T) {
	tests := []struct {
		name    string
		program ScheduledProgram
		want    bool
	}{
		{
			name: "720p movie",
			program: ScheduledProgram{
				MediaType: "movie",
				Media:     MovieRecord{MediaMetadata: MediaMetadata{Height: 720}},
			},
			want: true,
		},
		{
			name: "standard definition movie",
			program: ScheduledProgram{
				MediaType: "movie",
				Media:     MovieRecord{MediaMetadata: MediaMetadata{Height: 480}},
			},
		},
		{
			name: "HD scheduled episode",
			program: ScheduledProgram{
				MediaType:   "tv",
				EpisodePath: "/shows/example/s01e02.mkv",
				TVMedia: TVShowRecord{EpisodeMetadata: []TVEpisode{
					{Path: "/shows/example/s01e01.mkv", MediaMetadata: MediaMetadata{Height: 480}},
					{Path: "/shows/example/s01e02.mkv", MediaMetadata: MediaMetadata{Height: 1080}},
				}},
			},
			want: true,
		},
		{
			name: "HD asset is excluded",
			program: ScheduledProgram{
				MediaType: "asset",
				Asset:     ChannelAsset{MediaMetadata: MediaMetadata{Height: 1080}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.program.IsHD(); got != test.want {
				t.Fatalf("IsHD() = %v, want %v", got, test.want)
			}
		})
	}
}
