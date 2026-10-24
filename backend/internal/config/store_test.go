package config

import (
	"path/filepath"
	"testing"
)

func TestStoreLoadReturnsDefaultsWhenMissing(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.json"))

	settings, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	if settings.MediaPath != filepath.Join("data", "movies") {
		t.Fatalf("MediaPath = %q, want data/movies", settings.MediaPath)
	}
	if settings.TVMediaPath != filepath.Join("data", "tvshows") {
		t.Fatalf("TVMediaPath = %q, want data/tvshows", settings.TVMediaPath)
	}
	if len(settings.MediaSources) != 2 {
		t.Fatalf("len(MediaSources) = %d, want 2", len(settings.MediaSources))
	}
	if len(settings.AssetSources) != 1 || settings.AssetSources[0].Path != filepath.Join("data", "assets") || settings.AssetSources[0].Scope != AssetScopeGlobal {
		t.Fatalf("AssetSources = %#v", settings.AssetSources)
	}
	if settings.StreamingMode != StreamingModeDirect {
		t.Fatalf("StreamingMode = %q, want %q", settings.StreamingMode, StreamingModeDirect)
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "config.json"))

	if err := store.Save(Settings{
		MediaSources: []MediaSource{
			{Path: "/media/movies", ContentType: "movie"},
			{Path: "/media/movies2", ContentType: "movie"},
			{Path: "/media/tvshows", ContentType: "tv"},
		},
		AssetSources: []AssetSource{
			{Path: "/media/assets", Scope: AssetScopeGlobal},
			{Path: "/media/assets", Scope: AssetScopeGlobal},
			{Path: "", Scope: AssetScopeGlobal},
		},
	}); err != nil {
		t.Fatal(err)
	}

	settings, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	if settings.MediaPath != "/media/movies" {
		t.Fatalf("MediaPath = %q, want /media/movies", settings.MediaPath)
	}
	if settings.TVMediaPath != "/media/tvshows" {
		t.Fatalf("TVMediaPath = %q, want /media/tvshows", settings.TVMediaPath)
	}
	if len(settings.MediaSources) != 3 {
		t.Fatalf("MediaSources = %#v", settings.MediaSources)
	}
	if len(settings.AssetSources) != 1 || settings.AssetSources[0].Path != "/media/assets" || settings.AssetSources[0].Scope != AssetScopeGlobal {
		t.Fatalf("AssetSources = %#v", settings.AssetSources)
	}
	if settings.StreamingMode != StreamingModeDirect {
		t.Fatalf("StreamingMode = %q, want %q", settings.StreamingMode, StreamingModeDirect)
	}
}

func TestStoreSaveAndLoadStreamingMode(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.json"))

	if err := store.Save(Settings{
		MediaSources:  []MediaSource{{Path: "/media/movies", ContentType: "movie"}},
		StreamingMode: StreamingModeHLS,
	}); err != nil {
		t.Fatal(err)
	}

	settings, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if settings.StreamingMode != StreamingModeHLS {
		t.Fatalf("StreamingMode = %q, want %q", settings.StreamingMode, StreamingModeHLS)
	}
}

func TestStoreLoadMigratesLegacyPathsToSources(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "config.json"))

	if err := store.Save(Settings{MediaPath: "/legacy/movies", TVMediaPath: "/legacy/tv"}); err != nil {
		t.Fatal(err)
	}

	settings, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}

	if len(settings.MediaSources) != 2 {
		t.Fatalf("MediaSources = %#v", settings.MediaSources)
	}
}
