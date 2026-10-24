package mediapath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesSiblingMovieRoot(t *testing.T) {
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previous)

	actual := filepath.Join("data", "movies2", "Movie_(2001)", "Movie.mkv")
	if err := os.MkdirAll(filepath.Dir(actual), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(actual, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale := filepath.Join("data", "movies", "Movie_(2001)", "Movie.mkv")
	if resolved := Resolve(stale); resolved != actual {
		t.Fatalf("Resolve(%q) = %q, want %q", stale, resolved, actual)
	}
	if !RegularFileExists(stale) {
		t.Fatalf("RegularFileExists(%q) = false, want true", stale)
	}
}
