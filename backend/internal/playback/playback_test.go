package playback

import "testing"

func TestDirectFileRendererResolvesMovieURL(t *testing.T) {
	renderer := DirectFileRenderer{}

	playback := renderer.Render("http://retrotv.test/", Request{
		Program: Program{
			MediaType: MediaTypeMovie,
			MovieID:   42,
		},
		SeekPosition: 120,
	})

	if playback.URL != "http://retrotv.test/media/movies/42" {
		t.Fatalf("URL = %q", playback.URL)
	}
	if playback.SeekPosition != 120 {
		t.Fatalf("SeekPosition = %d", playback.SeekPosition)
	}
}

func TestDirectFileRendererResolvesTVEpisodeURL(t *testing.T) {
	renderer := DirectFileRenderer{}

	playback := renderer.Render("http://retrotv.test", Request{
		Program: Program{
			MediaType:   MediaTypeTV,
			TVShowID:    7,
			EpisodePath: "/media/Daria/Season 01/S01E01.mkv",
		},
		SeekPosition: 300,
	})

	want := "http://retrotv.test/media/tvshows/7?episode_path=%2Fmedia%2FDaria%2FSeason+01%2FS01E01.mkv"
	if playback.URL != want {
		t.Fatalf("URL = %q, want %q", playback.URL, want)
	}
	if playback.SeekPosition != 300 {
		t.Fatalf("SeekPosition = %d", playback.SeekPosition)
	}
}
