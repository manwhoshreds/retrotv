package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanMoviesParsesKodiMovieDirectoryNames(t *testing.T) {
	root := t.TempDir()
	movieDir := filepath.Join(root, "Alien (1979) [tmdb-348]")
	if err := os.Mkdir(movieDir, 0o700); err != nil {
		t.Fatal(err)
	}
	moviePath := filepath.Join(movieDir, "main_feature.MKV")
	if err := os.WriteFile(moviePath, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "poster.jpg"), []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	movies, err := ScanMovies(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 {
		t.Fatalf("len(movies) = %d, want 1", len(movies))
	}

	movie := movies[0]
	if movie.Path != moviePath {
		t.Fatalf("Path = %q, want %q", movie.Path, moviePath)
	}
	if movie.Filename != "main_feature.MKV" {
		t.Fatalf("Filename = %q", movie.Filename)
	}
	if movie.Title != "Alien" {
		t.Fatalf("Title = %q, want Alien", movie.Title)
	}
	if movie.Year != 1979 {
		t.Fatalf("Year = %d, want 1979", movie.Year)
	}
	if movie.TMDBID != "348" {
		t.Fatalf("TMDBID = %q, want 348", movie.TMDBID)
	}
}

func TestScanMoviesCapturesFilesystemFingerprint(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Alien (1979)", "Alien.mkv")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	movies, err := ScanMovies(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 || movies[0].MediaMetadata.FileSize != 5 || movies[0].MediaMetadata.ModTimeUnix <= 0 {
		t.Fatalf("fingerprint = %#v", movies)
	}
}

func TestScanMoviesFallsBackToKodiMovieFilenames(t *testing.T) {
	root := t.TempDir()
	moviePath := filepath.Join(root, "Aliens (1986).mkv")
	if err := os.WriteFile(moviePath, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	movies, err := ScanMovies(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 {
		t.Fatalf("len(movies) = %d, want 1", len(movies))
	}

	movie := movies[0]
	if movie.Title != "Aliens" {
		t.Fatalf("Title = %q, want Aliens", movie.Title)
	}
	if movie.Year != 1986 {
		t.Fatalf("Year = %d, want 1986", movie.Year)
	}
}

func TestScanTVShowsParsesKodiShowFolders(t *testing.T) {
	root := t.TempDir()
	showDir := filepath.Join(root, "Family_Guy_(1999)", "Season_01")
	if err := os.MkdirAll(showDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "S01E01.mkv"), []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	shows, err := ScanTVShows(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(shows) != 1 {
		t.Fatalf("len(shows) = %d, want 1", len(shows))
	}

	show := shows[0]
	if show.Title != "Family Guy" {
		t.Fatalf("Title = %q, want Family Guy", show.Title)
	}
	if show.Year != 1999 {
		t.Fatalf("Year = %d, want 1999", show.Year)
	}
	if show.EpisodeCount != 1 {
		t.Fatalf("EpisodeCount = %d, want 1", show.EpisodeCount)
	}
}

func TestScanTVShowsKeepsLocalOnlyShowFolders(t *testing.T) {
	root := t.TempDir()
	showDir := filepath.Join(root, "Mystery_Show", "Season_01")
	if err := os.MkdirAll(showDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "S01E01.m4v"), []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	shows, err := ScanTVShows(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(shows) != 1 {
		t.Fatalf("len(shows) = %d, want 1", len(shows))
	}
	if shows[0].Title != "Mystery Show" || shows[0].Year != 0 || shows[0].EpisodeCount != 1 {
		t.Fatalf("show = %#v", shows[0])
	}
}
