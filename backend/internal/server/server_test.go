package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"retrotv/internal/config"
	"retrotv/internal/database"
	"retrotv/internal/models"
	"retrotv/internal/playback"
)

func TestSettingsCanBeSavedAndRead(t *testing.T) {
	mediaDir := t.TempDir()
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              "/tv/Martin (1992)",
		LocalTitle:        "Martin",
		LocalYear:         1992,
		LocalEpisodeCount: 1,
		EpisodePaths:      []string{"/tv/Martin (1992)/Season_01/S01E01.mkv"},
		TVShow: models.TVShow{
			TMDBID:       171,
			Name:         "Martin",
			OriginalName: "Martin",
			Year:         1992,
			FirstAirDate: "1992-08-27",
			Overview:     "Overview",
			PosterPath:   "/poster.jpg",
			BackdropPath: "/backdrop.jpg",
			Genres:       []string{"Comedy"},
			Rating:       "TV-PG",
			Seasons:      1,
			Episodes:     1,
			Network:      "FOX",
		},
		PosterLocalPath:   "data/artwork/tv-poster.jpg",
		BackdropLocalPath: "data/artwork/tv-backdrop.jpg",
	}); err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	assetDir := filepath.Join(t.TempDir(), "assets")
	body := bytes.NewBufferString(`{"media_path":` + quote(t, mediaDir) + `,"tv_media_path":` + quote(t, mediaDir) + `,"asset_sources":[{"path":` + quote(t, assetDir) + `,"scope":"global"}],"streaming_mode":"hls"}`)
	request := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var settings struct {
		MediaPath    string `json:"media_path"`
		AssetSources []struct {
			Path  string `json:"path"`
			Scope string `json:"scope"`
		} `json:"asset_sources"`
		StreamingMode string `json:"streaming_mode"`
	}
	if err := json.NewDecoder(response.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if settings.MediaPath != mediaDir {
		t.Fatalf("MediaPath = %q, want %q", settings.MediaPath, mediaDir)
	}
	if len(settings.AssetSources) != 1 || settings.AssetSources[0].Path != assetDir || settings.AssetSources[0].Scope != "global" {
		t.Fatalf("AssetSources = %#v", settings.AssetSources)
	}
	if settings.StreamingMode != "hls" {
		t.Fatalf("StreamingMode = %q, want hls", settings.StreamingMode)
	}
}

func TestSettingsRejectsInvalidStreamingMode(t *testing.T) {
	mediaDir := t.TempDir()
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	body := bytes.NewBufferString(`{"media_path":` + quote(t, mediaDir) + `,"streaming_mode":"rtsp"}`)
	request := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestSettingsRejectsInvalidAssetSourceScope(t *testing.T) {
	mediaDir := t.TempDir()
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	body := bytes.NewBufferString(`{"media_path":` + quote(t, mediaDir) + `,"asset_sources":[{"path":"/assets","scope":"channel"}]}`)
	request := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestSettingsRejectsMissingDirectory(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	missingDir := filepath.Join(t.TempDir(), "missing")
	body := bytes.NewBufferString(`{"media_path":` + quote(t, missingDir) + `}`)
	request := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAssetScanPreservesAssetSources(t *testing.T) {
	mediaDir := t.TempDir()
	assetDir := t.TempDir()
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	body := bytes.NewBufferString(`{"media_path":` + quote(t, mediaDir) + `,"asset_sources":[{"path":` + quote(t, assetDir) + `,"scope":"global"}]}`)
	request := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("settings status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/assets/scan", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("scan status = %d, body = %s", response.Code, response.Body.String())
	}
	var scanPayload struct {
		AssetSources []config.AssetSource `json:"asset_sources"`
	}
	if err := json.NewDecoder(response.Body).Decode(&scanPayload); err != nil {
		t.Fatal(err)
	}
	if len(scanPayload.AssetSources) != 1 || scanPayload.AssetSources[0].Path != assetDir {
		t.Fatalf("scan asset_sources = %#v", scanPayload.AssetSources)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("settings status = %d, body = %s", response.Code, response.Body.String())
	}
	var settings config.Settings
	if err := json.NewDecoder(response.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.AssetSources) != 1 || settings.AssetSources[0].Path != assetDir || settings.AssetSources[0].Scope != config.AssetScopeGlobal {
		t.Fatalf("settings asset_sources = %#v", settings.AssetSources)
	}
}

func TestLegacyScanRoutesReturnJSON(t *testing.T) {
	mediaDir := t.TempDir()
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		MediaDir:         mediaDir,
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	tests := []struct {
		path  string
		field string
	}{
		{path: "/api/scan", field: "movies"},
		{path: "/api/scan/tv", field: "tv_shows"},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodPost, test.path, bytes.NewBufferString(`{}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body = %s", test.path, response.Code, response.Body.String())
		}
		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatalf("%s response is not JSON: %v; body = %s", test.path, err, response.Body.String())
		}
		if _, ok := payload[test.field]; !ok {
			t.Fatalf("%s missing %q in payload %#v", test.path, test.field, payload)
		}
	}
}

func TestMovieScanDoesNotGenerateChannelSchedules(t *testing.T) {
	mediaDir := t.TempDir()
	moviePath := filepath.Join(mediaDir, "Scan_Movie_(1999).mkv")
	if err := os.WriteFile(moviePath, []byte("not really media"), 0o644); err != nil {
		t.Fatal(err)
	}
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		MediaDir:         mediaDir,
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{
		Name:        "Scan Movies",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{MediaType: "movie", Field: "title", Operator: "is", Value: "Scan Movie"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/scan/movies", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("scan status = %d, body = %s", response.Code, response.Body.String())
	}

	programs, err := testServer.db.ScheduledPrograms(context.Background(), channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 0 {
		t.Fatalf("movie scan generated %d schedule programs, want 0", len(programs))
	}
}

func TestClassifyMoviesReportsIncrementalScanCounters(t *testing.T) {
	files := []models.MovieFile{
		{Path: "/media/unchanged.mkv", MediaMetadata: models.MediaMetadata{FileSize: 10, ModTimeUnix: 20}},
		{Path: "/media/changed.mkv", MediaMetadata: models.MediaMetadata{FileSize: 11, ModTimeUnix: 20}},
		{Path: "/media/new.mkv", MediaMetadata: models.MediaMetadata{FileSize: 12, ModTimeUnix: 20}},
	}
	cached := map[string]models.MovieRecord{
		"/media/unchanged.mkv": {FilePath: "/media/unchanged.mkv", MediaMetadata: models.MediaMetadata{FileSize: 10, ModTimeUnix: 20}},
		"/media/changed.mkv":   {FilePath: "/media/changed.mkv", MediaMetadata: models.MediaMetadata{FileSize: 10, ModTimeUnix: 20}},
		"/media/missing.mkv":   {FilePath: "/media/missing.mkv", MediaMetadata: models.MediaMetadata{FileSize: 10, ModTimeUnix: 20}},
	}

	got := classifyMovies(files, cached, []string{"/media"}, false)
	want := (scanCounters{Discovered: 3, Unchanged: 1, New: 1, Changed: 1, Missing: 1})
	if got != want {
		t.Fatalf("counters = %#v, want %#v", got, want)
	}
}

func TestMetadataFacetsReturnsLocalDatabaseValues(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	_, err = testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   "/movies/1408 (2007)/1408.mkv",
		Filename:   "1408.mkv",
		LocalTitle: "1408",
		LocalYear:  2007,
		Movie: models.Movie{
			TMDBID:         3021,
			Title:          "1408",
			OriginalTitle:  "1408",
			Year:           2007,
			ReleaseDate:    "2007-06-22",
			Overview:       "Overview",
			PosterPath:     "/poster.jpg",
			BackdropPath:   "/backdrop.jpg",
			RuntimeMinutes: 104,
			Genres:         []string{"Horror", "Mystery"},
			VoteAverage:    6.7,
			Studio:         "Dimension Films",
		},
		PosterLocalPath:   "data/artwork/3021-poster.jpg",
		BackdropLocalPath: "data/artwork/3021-backdrop.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              "/tv/Martin (1992)",
		LocalTitle:        "Martin",
		LocalYear:         1992,
		LocalEpisodeCount: 1,
		EpisodePaths:      []string{"/tv/Martin (1992)/Season_01/S01E01.mkv"},
		TVShow: models.TVShow{
			TMDBID:       171,
			Name:         "Martin",
			OriginalName: "Martin",
			Year:         1992,
			FirstAirDate: "1992-08-27",
			Overview:     "Overview",
			PosterPath:   "/poster.jpg",
			BackdropPath: "/backdrop.jpg",
			Genres:       []string{"Comedy"},
			Rating:       "TV-PG",
			Seasons:      1,
			Episodes:     1,
			Network:      "FOX",
		},
		PosterLocalPath:   "data/artwork/tv-poster.jpg",
		BackdropLocalPath: "data/artwork/tv-backdrop.jpg",
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/metadata/facets", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var facets models.MetadataFacets
	if err := json.NewDecoder(response.Body).Decode(&facets); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(facets.Titles, []string{"1408", "Martin"}) {
		t.Fatalf("Titles = %#v", facets.Titles)
	}
	if !reflect.DeepEqual(facets.Genres, []string{"Comedy", "Horror", "Mystery"}) {
		t.Fatalf("Genres = %#v", facets.Genres)
	}
	if !reflect.DeepEqual(facets.ReleaseYears, []int{
		1991, 1992, 1993, 1994, 1995, 1996, 1997, 1998, 1999,
		2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008,
	}) {
		t.Fatalf("ReleaseYears = %#v", facets.ReleaseYears)
	}
	if !reflect.DeepEqual(facets.Runtimes, []int{104}) {
		t.Fatalf("Runtimes = %#v", facets.Runtimes)
	}
	if !reflect.DeepEqual(facets.VoteAverages, []float64{6.7}) {
		t.Fatalf("VoteAverages = %#v", facets.VoteAverages)
	}
	if !reflect.DeepEqual(facets.Studios, []string{"Dimension Films"}) {
		t.Fatalf("Studios = %#v", facets.Studios)
	}
	if !reflect.DeepEqual(facets.Networks, []string{"FOX"}) {
		t.Fatalf("Networks = %#v", facets.Networks)
	}
}

func TestImportMoviesReusesCachedMetadataAndArtwork(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	posterPath := filepath.Join(testServer.artworkDir, "movies", "562", "poster.jpg")
	backdropPath := filepath.Join(testServer.artworkDir, "movies", "562", "backdrop.jpg")
	writeTestFile(t, posterPath, "poster")
	writeTestFile(t, backdropPath, "backdrop")

	file := models.MovieFile{
		Path:     "/movies/Die Hard (1988)/feature.mkv",
		Filename: "feature.mkv",
		Title:    "Die Hard",
		Year:     1988,
	}
	cached, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   file.Path,
		Filename:   file.Filename,
		LocalTitle: file.Title,
		LocalYear:  file.Year,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			ReleaseDate:    "1988-07-15",
			Overview:       "Cached overview",
			PosterPath:     "/poster.jpg",
			BackdropPath:   "/backdrop.jpg",
			RuntimeMinutes: 132,
			Genres:         []string{"Action"},
		},
		PosterLocalPath:   posterPath,
		BackdropLocalPath: backdropPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	records, errors := testServer.importMovies(context.Background(), []models.MovieFile{file})
	if len(errors) > 0 {
		t.Fatalf("errors = %#v", errors)
	}
	if len(records) != 1 || records[0].ID != cached.ID {
		t.Fatalf("records = %#v, want cached movie id %d", records, cached.ID)
	}
}

func TestImportTVShowsReusesCachedMetadataAndArtwork(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	posterPath := filepath.Join(testServer.artworkDir, "tvshows", "171", "poster.jpg")
	backdropPath := filepath.Join(testServer.artworkDir, "tvshows", "171", "backdrop.jpg")
	thumbnailPath := filepath.Join(testServer.artworkDir, "tvshows", "171", "seasons", "01", "episodes", "01.jpg")
	writeTestFile(t, posterPath, "poster")
	writeTestFile(t, backdropPath, "backdrop")
	writeTestFile(t, thumbnailPath, "thumbnail")

	episodePath := "/tv/Martin (1992)/Season_01/S01E01.mkv"
	file := models.TVShowFile{
		Path:         "/tv/Martin (1992)",
		Title:        "Martin",
		Year:         1992,
		EpisodeCount: 1,
		EpisodePaths: []string{episodePath},
	}
	cached, err := testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              file.Path,
		LocalTitle:        file.Title,
		LocalYear:         file.Year,
		LocalEpisodeCount: file.EpisodeCount,
		EpisodePaths:      file.EpisodePaths,
		EpisodeMetadata: []models.TVEpisode{
			{
				Path:               episodePath,
				Season:             1,
				Episode:            1,
				Name:               "Beauty and the Beast",
				Overview:           "Cached episode overview",
				AirDate:            "1992-08-27",
				StillPath:          "/still.jpg",
				ThumbnailLocalPath: thumbnailPath,
				RuntimeMinutes:     22,
			},
		},
		TVShow: models.TVShow{
			TMDBID:          171,
			MetadataVersion: currentTVShowMetadataVersion,
			Name:            "Martin",
			OriginalName:    "Martin",
			Year:            1992,
			FirstAirDate:    "1992-08-27",
			Overview:        "Cached overview",
			PosterPath:      "/poster.jpg",
			BackdropPath:    "/backdrop.jpg",
			Genres:          []string{"Comedy"},
			Rating:          "TV-PG",
			Seasons:         1,
			Episodes:        1,
			Network:         "FOX",
		},
		PosterLocalPath:   posterPath,
		BackdropLocalPath: backdropPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	records, errors := testServer.importTVShows(context.Background(), []models.TVShowFile{file})
	if len(errors) > 0 {
		t.Fatalf("errors = %#v", errors)
	}
	if len(records) != 1 || records[0].ID != cached.ID {
		t.Fatalf("records = %#v, want cached TV show id %d", records, cached.ID)
	}
}

func TestMovieCanBeDeleted(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	movie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   "/movies/Missing.mkv",
		Filename:   "Missing.mkv",
		LocalTitle: "Missing",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         100,
			Title:          "Missing",
			OriginalTitle:  "Missing",
			Year:           1988,
			ReleaseDate:    "1988-01-01",
			Overview:       "Gone",
			RuntimeMinutes: 90,
			Genres:         []string{"Action"},
			VoteAverage:    6.5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/movies/"+strconv.FormatInt(movie.ID, 10), nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/movies", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"count":0`) {
		t.Fatalf("movies = %s", response.Body.String())
	}
}

func TestImportMovieSavesLocalRecordWhenMetadataLookupFails(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	moviePath := filepath.Join(t.TempDir(), "Unmatched_Showcase_(1994)", "Unmatched_Showcase_(1994).mkv")
	if err := os.MkdirAll(filepath.Dir(moviePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moviePath, []byte("not really media"), 0o644); err != nil {
		t.Fatal(err)
	}

	records, errors := testServer.importMovies(context.Background(), []models.MovieFile{{
		Path:     moviePath,
		Filename: filepath.Base(moviePath),
		Title:    "Unmatched Showcase",
		Year:     1994,
	}})
	if len(records) != 1 {
		t.Fatalf("records len = %d errors = %#v", len(records), errors)
	}
	if records[0].TMDBID != 0 || records[0].Title != "Unmatched Showcase" || records[0].LocalTitle != "Unmatched Showcase" {
		t.Fatalf("local movie record = %#v", records[0])
	}
}

func TestImportMoviePreservesCachedMetadataWhenLookupFails(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	posterPath := filepath.Join(testServer.artworkDir, "movies", "456", "poster.jpg")
	backdropPath := filepath.Join(testServer.artworkDir, "movies", "456", "backdrop.jpg")
	if err := os.MkdirAll(filepath.Dir(posterPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(posterPath, []byte("poster"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backdropPath, []byte("backdrop"), 0o644); err != nil {
		t.Fatal(err)
	}

	moviePath := filepath.Join(t.TempDir(), "Cached_Movie_(1994)", "Cached_Movie_(1994).mkv")
	if err := os.MkdirAll(filepath.Dir(moviePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moviePath, []byte("not really media"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := models.MovieFile{
		Path:     moviePath,
		Filename: filepath.Base(moviePath),
		Title:    "Cached Movie",
		Year:     1994,
	}
	if _, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:          file.Path,
		Filename:          file.Filename,
		LocalTitle:        file.Title,
		LocalYear:         file.Year,
		PosterLocalPath:   posterPath,
		BackdropLocalPath: backdropPath,
		Movie: models.Movie{
			TMDBID:        456,
			Title:         "Cached Movie",
			OriginalTitle: "Cached Movie",
			Year:          1994,
			Overview:      "Cached overview",
			PosterPath:    "/poster.jpg",
			BackdropPath:  "/backdrop.jpg",
		},
	}); err != nil {
		t.Fatal(err)
	}

	records, errors := testServer.importMovies(context.Background(), []models.MovieFile{file})
	if len(records) != 1 {
		t.Fatalf("records len = %d errors = %#v", len(records), errors)
	}
	if records[0].TMDBID != 456 || records[0].PosterLocalPath != posterPath || records[0].BackdropLocalPath != backdropPath {
		t.Fatalf("cached metadata was not preserved: %#v", records[0])
	}
}

func TestImportTVShowSavesLocalRecordWhenMetadataLookupFails(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	episodePath := filepath.Join(t.TempDir(), "Mystery_Show_(1998)", "Season_01", "S01E01.mkv")
	if err := os.MkdirAll(filepath.Dir(episodePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(episodePath, []byte("not really media"), 0o644); err != nil {
		t.Fatal(err)
	}

	records, errors := testServer.importTVShows(context.Background(), []models.TVShowFile{{
		Path:         filepath.Dir(filepath.Dir(episodePath)),
		Title:        "Mystery Show",
		Year:         1998,
		EpisodeCount: 1,
		EpisodePaths: []string{episodePath},
	}})
	if len(records) != 1 {
		t.Fatalf("records len = %d errors = %#v", len(records), errors)
	}
	if records[0].TMDBID != 0 || records[0].Name != "Mystery Show" || records[0].LocalEpisodeCount != 1 {
		t.Fatalf("local TV record = %#v", records[0])
	}
	if len(records[0].EpisodePaths) != 1 || len(records[0].EpisodeMetadata) != 1 {
		t.Fatalf("episode data = %#v %#v", records[0].EpisodePaths, records[0].EpisodeMetadata)
	}
}

func TestImportTVShowSavesNoYearFolderAsLocalRecord(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	showPath := filepath.Join(t.TempDir(), "Tom and Jerry")
	episodePath := filepath.Join(showPath, "Season_01", "S01E01.mkv")
	if err := os.MkdirAll(filepath.Dir(episodePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(episodePath, []byte("not really media"), 0o644); err != nil {
		t.Fatal(err)
	}

	records, errors := testServer.importTVShows(context.Background(), []models.TVShowFile{{
		Path:         showPath,
		Title:        "Tom and Jerry",
		EpisodeCount: 1,
		EpisodePaths: []string{episodePath},
	}})
	if len(records) != 1 {
		t.Fatalf("records len = %d errors = %#v", len(records), errors)
	}
	if records[0].TMDBID != 0 || records[0].Name != "Tom and Jerry" || records[0].LocalTitle != "Tom and Jerry" {
		t.Fatalf("local TV record = %#v", records[0])
	}
	if len(records[0].EpisodePaths) != 1 {
		t.Fatalf("episode paths = %#v", records[0].EpisodePaths)
	}
}

func TestImportTVShowPreservesCachedMetadataWhenLookupFails(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	posterPath := filepath.Join(testServer.artworkDir, "tvshows", "123", "poster.jpg")
	backdropPath := filepath.Join(testServer.artworkDir, "tvshows", "123", "backdrop.jpg")
	if err := os.MkdirAll(filepath.Dir(posterPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(posterPath, []byte("poster"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backdropPath, []byte("backdrop"), 0o644); err != nil {
		t.Fatal(err)
	}

	episodePath := filepath.Join(t.TempDir(), "Cached_Show_(1998)", "Season_01", "S01E01.mkv")
	if err := os.MkdirAll(filepath.Dir(episodePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(episodePath, []byte("not really media"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := models.TVShowFile{
		Path:         filepath.Dir(filepath.Dir(episodePath)),
		Title:        "Cached Show",
		Year:         1998,
		EpisodeCount: 1,
		EpisodePaths: []string{episodePath},
	}
	if _, err := testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              file.Path,
		LocalTitle:        file.Title,
		LocalYear:         file.Year,
		LocalEpisodeCount: file.EpisodeCount,
		EpisodePaths:      file.EpisodePaths,
		EpisodeMetadata:   []models.TVEpisode{{Path: episodePath, Season: 1, Episode: 1, Name: "Pilot"}},
		PosterLocalPath:   posterPath,
		BackdropLocalPath: backdropPath,
		TVShow: models.TVShow{
			TMDBID:          123,
			MetadataVersion: currentTVShowMetadataVersion - 1,
			Name:            "Cached Show",
			OriginalName:    "Cached Show",
			Year:            1998,
			Overview:        "Cached overview",
			PosterPath:      "/poster.jpg",
			BackdropPath:    "/backdrop.jpg",
			Episodes:        1,
		},
	}); err != nil {
		t.Fatal(err)
	}

	records, errors := testServer.importTVShows(context.Background(), []models.TVShowFile{file})
	if len(records) != 1 {
		t.Fatalf("records len = %d errors = %#v", len(records), errors)
	}
	if records[0].TMDBID != 123 || records[0].PosterLocalPath != posterPath || records[0].BackdropLocalPath != backdropPath {
		t.Fatalf("cached metadata was not preserved: %#v", records[0])
	}
}

func TestChannelCanBeSavedAndListed(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	body := bytes.NewBufferString(`{
		"name":"80s Action",
		"description":"Explosions and one-liners.",
		"logo_path":"/logos/80s.png",
		"content_type":"movie",
		"rules":[
			{"media_type":"movie","field":"genre","operator":"is","value":"Action"},
			{"media_type":"tv","field":"year","operator":"greater_than","value":"1979"}
		]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/channels", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var channel models.Channel
	if err := json.NewDecoder(response.Body).Decode(&channel); err != nil {
		t.Fatal(err)
	}
	if channel.ID == 0 || len(channel.Rules) != 2 {
		t.Fatalf("channel = %#v", channel)
	}
	if channel.Description != "Explosions and one-liners." {
		t.Fatalf("description = %q", channel.Description)
	}
	if channel.Rules[0].MediaType != "movie" || channel.Rules[1].MediaType != "tv" {
		t.Fatalf("channel rules = %#v", channel.Rules)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestChannelProgrammingEndpointSavesAllDayAndTimelineBlocks(t *testing.T) {
	testServer, err := NewServer(Config{StaticDir: ".", ConfigDir: t.TempDir(), DisableScheduler: true})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Programming", ContentType: "mixed"})
	if err != nil {
		t.Fatal(err)
	}
	collection, err := testServer.db.SaveCollection(context.Background(), models.Collection{Name: "Classics"})
	if err != nil {
		t.Fatal(err)
	}
	body := bytes.NewBufferString(fmt.Sprintf(`{"programming_blocks":[{"name":"Default Programming","placement":"all_day","days":["mon","tue","wed","thu","fri","sat","sun"],"content_source":"collection","collection_id":%d,"enabled":true},{"name":"Prime Time","placement":"timeline","start_time":"20:00","end_time":"22:00","days":["fri"],"content_source":"collection","collection_id":%d,"enabled":true}]}`, collection.ID, collection.ID))
	request := httptest.NewRequest(http.MethodPut, "/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/programming", body)
	response := httptest.NewRecorder()
	testServer.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var saved models.Channel
	if err := json.NewDecoder(response.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.ProgrammingBlocks) != 2 || saved.ProgrammingBlocks[0].Placement != models.ProgrammingBlockPlacementAllDay && saved.ProgrammingBlocks[1].Placement != models.ProgrammingBlockPlacementAllDay {
		t.Fatalf("programming blocks = %#v", saved.ProgrammingBlocks)
	}
}

func TestChannelLogoCanBeUploadedAndServed(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("logo", "Action Logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/channel-logos", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		LogoPath string `json:"logo_path"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(payload.LogoPath, "/artwork/channels/uploads/") || !strings.HasSuffix(payload.LogoPath, ".png") {
		t.Fatalf("logo_path = %q", payload.LogoPath)
	}

	request = httptest.NewRequest(http.MethodGet, payload.LogoPath, nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestChannelXMLTVAndM3UAreGeneratedFromScheduledPrograms(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	moviePath := filepath.Join(t.TempDir(), "Die Hard (1988)", "feature.mkv")
	if err := os.MkdirAll(filepath.Dir(moviePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	movie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   moviePath,
		Filename:   "feature.mkv",
		LocalTitle: "Die Hard",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			ReleaseDate:    "1988-07-15",
			Overview:       "Overview",
			PosterPath:     "/poster.jpg",
			BackdropPath:   "/backdrop.jpg",
			RuntimeMinutes: 132,
			Genres:         []string{"Action"},
			VoteAverage:    7.8,
		},
		PosterLocalPath:   "data/artwork/562-poster.jpg",
		BackdropLocalPath: "data/artwork/562-backdrop.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}

	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		LogoPath:    "/logos/80s.png",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{Field: "genre", Operator: "is", Value: "Action"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := testServer.scheduler.GenerateChannel(context.Background(), channel.ID); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/channels/"+strconv.FormatInt(channel.ID, 10)+"/xmltv", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Die Hard") {
		t.Fatalf("xmltv = %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "562-poster.jpg") {
		t.Fatalf("xmltv missing poster icon = %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `src="http://retrotv.test/logos/80s.png"`) {
		t.Fatalf("xmltv missing channel logo = %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "http://retrotv.test/channels/"+strconv.FormatInt(channel.ID, 10)+"/m3u", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	absolutePath, err := filepath.Abs(moviePath)
	if err != nil {
		t.Fatal(err)
	}
	streamURL := "http://retrotv.test/media/movies/" + strconv.FormatInt(movie.ID, 10)
	if !strings.Contains(response.Body.String(), "#EXTM3U") ||
		!strings.Contains(response.Body.String(), "80s Action") ||
		!strings.Contains(response.Body.String(), `tvg-logo="http://retrotv.test/logos/80s.png"`) ||
		!strings.Contains(response.Body.String(), streamURL) {
		t.Fatalf("m3u = %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/schedule", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	absolutePosterPath, err := filepath.Abs("data/artwork/562-poster.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Body.String(), `"count":`) ||
		!strings.Contains(response.Body.String(), absolutePath) ||
		!strings.Contains(response.Body.String(), `"release_date":"1988-07-15"`) ||
		!strings.Contains(response.Body.String(), `"year":1988`) ||
		!strings.Contains(response.Body.String(), `"stream_url":"http://example.com/media/movies/`+strconv.FormatInt(movie.ID, 10)+`"`) ||
		!strings.Contains(response.Body.String(), absolutePosterPath) {
		t.Fatalf("schedule = %s", response.Body.String())
	}
	var schedulePayload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &schedulePayload); err != nil {
		t.Fatal(err)
	}
	for _, unexpected := range []string{"channel", "server_time", "current_program"} {
		if _, ok := schedulePayload[unexpected]; ok {
			t.Fatalf("schedule unexpectedly included %q: %s", unexpected, response.Body.String())
		}
	}
	strictScheduleDecoder := json.NewDecoder(strings.NewReader(response.Body.String()))
	strictScheduleDecoder.DisallowUnknownFields()
	var strictSchedule struct {
		Programs []struct {
			ID        int64     `json:"id"`
			ChannelID int64     `json:"channel_id"`
			MediaID   int64     `json:"media_id"`
			StartTime time.Time `json:"start_time"`
			EndTime   time.Time `json:"end_time"`
			Media     struct {
				ID              int64    `json:"id"`
				Type            string   `json:"type"`
				Title           string   `json:"title"`
				Overview        string   `json:"overview"`
				ReleaseDate     string   `json:"release_date,omitempty"`
				Year            int      `json:"year,omitempty"`
				FilePath        string   `json:"file_path"`
				PosterLocalPath string   `json:"poster_local_path"`
				StreamURL       string   `json:"stream_url"`
				Genres          []string `json:"genres,omitempty"`
				Rating          string   `json:"rating,omitempty"`
			} `json:"media"`
			ChannelAsset *models.ChannelAsset `json:"channel_asset,omitempty"`
		} `json:"programs"`
		ChannelAssets []models.ChannelAsset `json:"channel_assets"`
		Count         int                   `json:"count"`
	}
	if err := strictScheduleDecoder.Decode(&strictSchedule); err != nil {
		t.Fatalf("strict schedule decode failed: %v; body = %s", err, response.Body.String())
	}
}

func TestChannelScheduleIncludesChannelAssets(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	assetPath := filepath.Join(t.TempDir(), "bumpers", "coming-up.mp4")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	asset, err := testServer.db.UpsertChannelAsset(context.Background(), models.ChannelAsset{
		FilePath:        assetPath,
		Filename:        "coming-up.mp4",
		AssetType:       "bumper",
		Tag:             "up-next",
		Scope:           "global",
		DurationSeconds: 30,
		Enabled:         true,
		MediaMetadata: models.MediaMetadata{
			DurationSeconds: 30,
			VideoCodec:      "h264",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Assets", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := testServer.db.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: asset.ID, MediaType: "asset", StartTime: now, EndTime: now.Add(30 * time.Second)},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/schedule", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Programs []struct {
			Media struct {
				Type      string `json:"type"`
				Title     string `json:"title"`
				FilePath  string `json:"file_path"`
				StreamURL string `json:"stream_url"`
			} `json:"media"`
			ChannelAsset *models.ChannelAsset `json:"channel_asset"`
		} `json:"programs"`
		ChannelAssets []models.ChannelAsset `json:"channel_assets"`
		Count         int                   `json:"count"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Count != 1 || len(payload.Programs) != 1 {
		t.Fatalf("schedule count = %d programs = %d, body = %s", payload.Count, len(payload.Programs), response.Body.String())
	}
	if len(payload.ChannelAssets) != 1 {
		t.Fatalf("channel_assets len = %d, body = %s", len(payload.ChannelAssets), response.Body.String())
	}
	program := payload.Programs[0]
	if program.ChannelAsset == nil {
		t.Fatalf("program missing channel_asset: %s", response.Body.String())
	}
	if payload.ChannelAssets[0].ID != asset.ID || program.ChannelAsset.ID != asset.ID {
		t.Fatalf("wrong asset included: %#v %#v", payload.ChannelAssets[0], program.ChannelAsset)
	}
	if program.Media.Type != "asset" || program.Media.Title != "Bumper: coming-up.mp4" || program.Media.FilePath != assetPath {
		t.Fatalf("asset media = %#v", program.Media)
	}
	if program.Media.StreamURL != "http://retrotv.test/media/assets/"+strconv.FormatInt(asset.ID, 10) {
		t.Fatalf("stream_url = %q", program.Media.StreamURL)
	}
	if payload.ChannelAssets[0].Tag != "up-next" || payload.ChannelAssets[0].MediaMetadata.VideoCodec != "h264" {
		t.Fatalf("channel_asset = %#v", payload.ChannelAssets[0])
	}
}

func TestMediaEndpointsExposeTechnicalMetadata(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	if _, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   "/movies/Technical (1999)/Technical.mkv",
		Filename:   "Technical.mkv",
		LocalTitle: "Technical",
		LocalYear:  1999,
		Movie: models.Movie{
			TMDBID:         99,
			Title:          "Technical",
			OriginalTitle:  "Technical",
			Year:           1999,
			ReleaseDate:    "1999-01-01",
			Overview:       "Overview",
			RuntimeMinutes: 90,
			Genres:         []string{"Drama"},
		},
		MediaMetadata: models.MediaMetadata{
			DurationSeconds: 5400,
			VideoCodec:      "h264",
			AudioCodec:      "aac",
			Width:           1920,
			Height:          1080,
			FrameRate:       23.976,
			FileSize:        123456789,
			AudioTracks: []models.AudioTrack{
				{Index: 1, Codec: "aac", Language: "eng", Channels: 2},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              "/tv/Technical Show (2001)",
		LocalTitle:        "Technical Show",
		LocalYear:         2001,
		LocalEpisodeCount: 1,
		EpisodePaths:      []string{"/tv/Technical Show (2001)/Season_01/S01E01.mkv"},
		EpisodeMetadata: []models.TVEpisode{
			{
				Path:           "/tv/Technical Show (2001)/Season_01/S01E01.mkv",
				Season:         1,
				Episode:        1,
				Name:           "Pilot",
				RuntimeMinutes: 44,
				MediaMetadata:  models.MediaMetadata{VideoCodec: "hevc", AudioCodec: "ac3", Width: 1280, Height: 720, FileSize: 987654321},
			},
		},
		TVShow: models.TVShow{
			TMDBID:       101,
			Name:         "Technical Show",
			OriginalName: "Technical Show",
			Year:         2001,
			FirstAirDate: "2001-01-01",
			Overview:     "Overview",
			Genres:       []string{"Drama"},
			Seasons:      1,
			Episodes:     1,
			Network:      "FOX",
		},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/movies", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("movies status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"media_metadata"`) ||
		!strings.Contains(response.Body.String(), `"video_codec":"h264"`) ||
		!strings.Contains(response.Body.String(), `"file_size":123456789`) {
		t.Fatalf("movie technical metadata missing from response: %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/tvshows", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("tv status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"media_metadata"`) ||
		!strings.Contains(response.Body.String(), `"video_codec":"hevc"`) ||
		!strings.Contains(response.Body.String(), `"file_size":987654321`) {
		t.Fatalf("TV episode technical metadata missing from response: %s", response.Body.String())
	}
}

func TestCurrentAndFutureProgramsDropsPastPrograms(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	programs := []models.ScheduledProgram{
		{MediaID: 1, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-1 * time.Hour)},
		{MediaID: 2, StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(1 * time.Hour)},
		{MediaID: 3, StartTime: now.Add(1 * time.Hour), EndTime: now.Add(2 * time.Hour)},
	}

	filtered := currentAndFuturePrograms(programs, now)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
	if filtered[0].MediaID != 2 || filtered[1].MediaID != 3 {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestTuneChannelReturnsCurrentPlaybackPayload(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	moviePath := filepath.Join(t.TempDir(), "Die Hard (1988)", "feature.mkv")
	writeTestFile(t, moviePath, "movie")
	movie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   moviePath,
		Filename:   "feature.mkv",
		LocalTitle: "Die Hard",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			ReleaseDate:    "1988-07-15",
			Overview:       "Current overview",
			PosterPath:     "/poster.jpg",
			BackdropPath:   "/backdrop.jpg",
			RuntimeMinutes: 132,
			Genres:         []string{"Action"},
			Rating:         "R",
		},
		MediaMetadata: models.MediaMetadata{
			SubtitleTracks: []models.SubtitleTrack{{Index: 2, Codec: "subrip", Language: "eng"}},
			Height:         1080,
		},
		PosterLocalPath: "data/artwork/movies/562/poster.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	nextMovie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Predator (1987)", "feature.mkv"),
		Filename:   "feature.mkv",
		LocalTitle: "Predator",
		LocalYear:  1987,
		Movie: models.Movie{
			TMDBID:         106,
			Title:          "Predator",
			OriginalTitle:  "Predator",
			Year:           1987,
			ReleaseDate:    "1987-06-12",
			Overview:       "Next overview",
			RuntimeMinutes: 107,
			Genres:         []string{"Action"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		LogoPath:    "/artwork/channels/1/logo.png",
		ContentType: "movie",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	currentStart := now.Add(-10 * time.Minute)
	currentEnd := now.Add(30 * time.Minute)
	nextStart := currentEnd
	nextEnd := nextStart.Add(2 * time.Hour)
	if err := testServer.db.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: movie.ID, MediaType: "movie", StartTime: currentStart, EndTime: currentEnd},
		{ChannelID: channel.ID, MediaID: nextMovie.ID, MediaType: "movie", StartTime: nextStart, EndTime: nextEnd},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Channel struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			LogoURL string `json:"logo_url"`
		} `json:"channel"`
		Program *struct {
			ID                int64    `json:"id"`
			Title             string   `json:"title"`
			Description       string   `json:"description"`
			Year              int      `json:"year"`
			Rating            string   `json:"rating"`
			Genres            []string `json:"genres"`
			HasClosedCaptions bool     `json:"has_closed_captions"`
			IsHD              bool     `json:"is_hd"`
			PlayURL           string   `json:"play_url"`
			PosterURL         string   `json:"poster_url"`
			SeekPosition      int      `json:"seek_position"`
		} `json:"program"`
		NextProgram *struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
		} `json:"next_program"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Channel.ID != channel.ID || payload.Channel.Name != "80s Action" || payload.Channel.LogoURL != "http://retrotv.test/artwork/channels/1/logo.png" {
		t.Fatalf("channel = %#v", payload.Channel)
	}
	if payload.Program == nil {
		t.Fatal("program = nil")
	}
	if payload.Program.Title != "Die Hard" || payload.Program.Description != "Current overview" {
		t.Fatalf("program = %#v", payload.Program)
	}
	if payload.Program.Year != 1988 || payload.Program.Rating != "R" || !reflect.DeepEqual(payload.Program.Genres, []string{"Action"}) {
		t.Fatalf("program metadata = %#v", payload.Program)
	}
	if !payload.Program.HasClosedCaptions {
		t.Fatalf("has_closed_captions = false, want true")
	}
	if !payload.Program.IsHD {
		t.Fatalf("is_hd = false, want true")
	}
	if payload.Program.PlayURL != "http://retrotv.test/media/movies/"+strconv.FormatInt(movie.ID, 10) {
		t.Fatalf("play_url = %q", payload.Program.PlayURL)
	}
	if payload.Program.PosterURL != "http://retrotv.test/artwork/movies/562/poster.jpg" {
		t.Fatalf("poster_url = %q", payload.Program.PosterURL)
	}
	if payload.Program.SeekPosition < 590 || payload.Program.SeekPosition > 610 {
		t.Fatalf("seek_position = %d, want about 600", payload.Program.SeekPosition)
	}
	if payload.NextProgram == nil || payload.NextProgram.Title != "Predator" {
		t.Fatalf("next_program = %#v", payload.NextProgram)
	}
}

func TestResolveTVEpisodePathAfterSourceMove(t *testing.T) {
	show := models.TVShowRecord{EpisodePaths: []string{
		"/new/library/Show/Season_01/S01E07.mkv",
		"/new/library/Show/Season_01/S01E08.mkv",
	}}
	resolved, ok := resolveTVEpisodePath(show, "data/tvshows/Show/Season_01/S01E07.mkv")
	if !ok || resolved != show.EpisodePaths[0] {
		t.Fatalf("resolveTVEpisodePath = %q, %v", resolved, ok)
	}
}

func TestResolveTVEpisodePathRejectsAmbiguousSuffix(t *testing.T) {
	show := models.TVShowRecord{EpisodePaths: []string{
		"/library/Show/Season_01/S01E07.mkv",
		"/backup/Show/Season_01/S01E07.mkv",
	}}
	if resolved, ok := resolveTVEpisodePath(show, "data/tvshows/Show/Season_01/S01E07.mkv"); ok {
		t.Fatalf("resolveTVEpisodePath = %q, want ambiguous match rejected", resolved)
	}
}

func TestServerStartsChannelStreamOnTuneAndReusesExistingStream(t *testing.T) {
	configDir := t.TempDir()
	databasePath := filepath.Join(configDir, "retrotv.sqlite")
	store, err := database.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	movie, err := store.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Die Hard (1988)", "feature.mkv"),
		Filename:   "feature.mkv",
		LocalTitle: "Die Hard",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			ReleaseDate:    "1988-07-15",
			Overview:       "Current overview",
			RuntimeMinutes: 132,
			Genres:         []string{"Action"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		ContentType: "movie",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{
			ChannelID: channel.ID,
			MediaID:   movie.ID,
			MediaType: "movie",
			StartTime: now.Add(-10 * time.Minute),
			EndTime:   now.Add(30 * time.Minute),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	starter := &fakePersistentStreamStarter{}
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        configDir,
		DatabasePath:     databasePath,
		DisableScheduler: true,
		StreamStarter:    starter,
		StreamBaseURL:    "http://retrotv.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	testServer.settings.StreamingMode = config.StreamingModeHLS
	server := testServer.Handler()
	if starter.starts != 0 {
		t.Fatalf("starts = %d, want 0 before tune", starter.starts)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Program *struct {
			PlayURL string `json:"play_url"`
		} `json:"program"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Program == nil {
		t.Fatal("program = nil")
	}
	if payload.Program.PlayURL != "http://retrotv.test/live/channel/"+strconv.FormatInt(channel.ID, 10) {
		t.Fatalf("play_url = %q", payload.Program.PlayURL)
	}
	if starter.starts != 1 {
		t.Fatalf("starts = %d, want 1 after first tune", starter.starts)
	}

	secondResponse := httptest.NewRecorder()
	server.ServeHTTP(secondResponse, request)
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second status = %d, body = %s", secondResponse.Code, secondResponse.Body.String())
	}
	if starter.starts != 1 {
		t.Fatalf("starts = %d, want 1 after second tune reuse", starter.starts)
	}
}

func TestSettingsDirectModeStopsActiveChannelStreams(t *testing.T) {
	configDir := t.TempDir()
	databasePath := filepath.Join(configDir, "retrotv.sqlite")
	store, err := database.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	movie, err := store.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Die Hard (1988)", "feature.mkv"),
		Filename:   "feature.mkv",
		LocalTitle: "Die Hard",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			RuntimeMinutes: 132,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		ContentType: "movie",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{
			ChannelID: channel.ID,
			MediaID:   movie.ID,
			MediaType: "movie",
			StartTime: now.Add(-10 * time.Minute),
			EndTime:   now.Add(30 * time.Minute),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	starter := &fakePersistentStreamStarter{}
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        configDir,
		DatabasePath:     databasePath,
		DisableScheduler: true,
		StreamStarter:    starter,
		StreamBaseURL:    "http://retrotv.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	testServer.settings.StreamingMode = config.StreamingModeHLS
	server := testServer.Handler()

	tuneRequest := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	tuneResponse := httptest.NewRecorder()
	server.ServeHTTP(tuneResponse, tuneRequest)
	if tuneResponse.Code != http.StatusOK {
		t.Fatalf("tune status = %d, body = %s", tuneResponse.Code, tuneResponse.Body.String())
	}
	if starter.starts != 1 || len(starter.processes) != 1 {
		t.Fatalf("starts = %d processes = %d, want one active stream", starter.starts, len(starter.processes))
	}
	if testServer.sessions.ActiveSessions() != 1 {
		t.Fatalf("active sessions = %d, want 1", testServer.sessions.ActiveSessions())
	}

	mediaDir := t.TempDir()
	body := bytes.NewBufferString(`{"media_path":` + quote(t, mediaDir) + `,"tv_media_path":` + quote(t, mediaDir) + `,"streaming_mode":"direct"}`)
	settingsRequest := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	settingsRequest.Header.Set("Content-Type", "application/json")
	settingsResponse := httptest.NewRecorder()
	server.ServeHTTP(settingsResponse, settingsRequest)
	if settingsResponse.Code != http.StatusOK {
		t.Fatalf("settings status = %d, body = %s", settingsResponse.Code, settingsResponse.Body.String())
	}
	if !starter.processes[0].stopped {
		t.Fatal("active stream was not stopped")
	}
	if testServer.sessions.ActiveSessions() != 0 {
		t.Fatalf("active sessions = %d, want 0", testServer.sessions.ActiveSessions())
	}
}

func TestTuneChannelUsesDirectPlaybackByDefault(t *testing.T) {
	configDir := t.TempDir()
	databasePath := filepath.Join(configDir, "retrotv.sqlite")
	store, err := database.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	movie, err := store.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Die Hard (1988)", "feature.mkv"),
		Filename:   "feature.mkv",
		LocalTitle: "Die Hard",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			RuntimeMinutes: 132,
			Genres:         []string{"Action"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		ContentType: "movie",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{
			ChannelID: channel.ID,
			MediaID:   movie.ID,
			MediaType: "movie",
			StartTime: now.Add(-10 * time.Minute),
			EndTime:   now.Add(30 * time.Minute),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	starter := &fakePersistentStreamStarter{}
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        configDir,
		DatabasePath:     databasePath,
		DisableScheduler: true,
		StreamStarter:    starter,
		StreamBaseURL:    "http://retrotv.test",
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	response := httptest.NewRecorder()
	testServer.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Program *struct {
			PlayURL string `json:"play_url"`
		} `json:"program"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Program == nil {
		t.Fatal("program = nil")
	}
	if payload.Program.PlayURL != "http://retrotv.test/media/movies/"+strconv.FormatInt(movie.ID, 10) {
		t.Fatalf("play_url = %q", payload.Program.PlayURL)
	}
	if starter.starts != 0 {
		t.Fatalf("starts = %d, want 0 in direct mode", starter.starts)
	}
}

func TestSavingDirectModeStopsActiveHLSStreams(t *testing.T) {
	configDir := t.TempDir()
	databasePath := filepath.Join(configDir, "retrotv.sqlite")
	store, err := database.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	movie, err := store.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Die Hard (1988)", "feature.mkv"),
		Filename:   "feature.mkv",
		LocalTitle: "Die Hard",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			RuntimeMinutes: 132,
			Genres:         []string{"Action"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		ContentType: "movie",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{
			ChannelID: channel.ID,
			MediaID:   movie.ID,
			MediaType: "movie",
			StartTime: now.Add(-10 * time.Minute),
			EndTime:   now.Add(30 * time.Minute),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	starter := &fakePersistentStreamStarter{}
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        configDir,
		DatabasePath:     databasePath,
		DisableScheduler: true,
		StreamStarter:    starter,
		StreamBaseURL:    "http://retrotv.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	testServer.settings.StreamingMode = config.StreamingModeHLS
	server := testServer.Handler()

	tuneRequest := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	tuneResponse := httptest.NewRecorder()
	server.ServeHTTP(tuneResponse, tuneRequest)
	if tuneResponse.Code != http.StatusOK {
		t.Fatalf("tune status = %d, body = %s", tuneResponse.Code, tuneResponse.Body.String())
	}
	if starter.starts != 1 || len(starter.processes) != 1 {
		t.Fatalf("starts = %d processes = %d, want one active stream", starter.starts, len(starter.processes))
	}

	mediaDir := t.TempDir()
	body := bytes.NewBufferString(`{"media_sources":[{"path":` + quote(t, mediaDir) + `,"content_type":"movie"}],"streaming_mode":"direct"}`)
	settingsRequest := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	settingsRequest.Header.Set("Content-Type", "application/json")
	settingsResponse := httptest.NewRecorder()
	server.ServeHTTP(settingsResponse, settingsRequest)
	if settingsResponse.Code != http.StatusOK {
		t.Fatalf("settings status = %d, body = %s", settingsResponse.Code, settingsResponse.Body.String())
	}
	if !starter.processes[0].stopped {
		t.Fatal("active HLS process was not stopped after switching to direct mode")
	}

	secondTuneResponse := httptest.NewRecorder()
	server.ServeHTTP(secondTuneResponse, tuneRequest)
	if secondTuneResponse.Code != http.StatusOK {
		t.Fatalf("second tune status = %d, body = %s", secondTuneResponse.Code, secondTuneResponse.Body.String())
	}
	if starter.starts != 1 {
		t.Fatalf("starts = %d, want no new stream after switching to direct mode", starter.starts)
	}
}

func TestCurrentChannelRequestDisabledInDirectMode(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	testServer.settings.StreamingMode = config.StreamingModeDirect

	request, ok, err := testServer.CurrentChannelRequest(context.Background(), 1, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if ok || request.ChannelID != 0 {
		t.Fatalf("request = %#v ok=%v, want no HLS request in direct mode", request, ok)
	}
}

func TestChannelScheduleUsesHLSPlaylistInHLSMode(t *testing.T) {
	configDir := t.TempDir()
	databasePath := filepath.Join(configDir, "retrotv.sqlite")
	store, err := database.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	movie, err := store.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Die Hard (1988)", "feature.mkv"),
		Filename:   "feature.mkv",
		LocalTitle: "Die Hard",
		LocalYear:  1988,
		Movie: models.Movie{
			TMDBID:         562,
			Title:          "Die Hard",
			OriginalTitle:  "Die Hard",
			Year:           1988,
			RuntimeMinutes: 132,
			Genres:         []string{"Action"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		ContentType: "movie",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{
			ChannelID: channel.ID,
			MediaID:   movie.ID,
			MediaType: "movie",
			StartTime: now.Add(-10 * time.Minute),
			EndTime:   now.Add(30 * time.Minute),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	starter := &fakePersistentStreamStarter{}
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        configDir,
		DatabasePath:     databasePath,
		DisableScheduler: true,
		StreamStarter:    starter,
	})
	if err != nil {
		t.Fatal(err)
	}
	testServer.settings.StreamingMode = config.StreamingModeHLS

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/schedule", nil)
	response := httptest.NewRecorder()
	testServer.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"count":1`) ||
		!strings.Contains(response.Body.String(), `"stream_url":"http://retrotv.test/streams/channels/`+strconv.FormatInt(channel.ID, 10)+`/index.m3u8"`) {
		t.Fatalf("schedule = %s", response.Body.String())
	}
	if starter.starts != 0 {
		t.Fatalf("starts = %d, want 0 before tune", starter.starts)
	}
}

func TestChannelScheduleCurrentQueryStillReturnsFullSchedule(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	firstMovie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Current.mkv"),
		Filename:   "Current.mkv",
		LocalTitle: "Current",
		LocalYear:  2026,
		Movie: models.Movie{
			TMDBID:         201,
			Title:          "Current",
			OriginalTitle:  "Current",
			Year:           2026,
			RuntimeMinutes: 30,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	secondMovie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Next.mkv"),
		Filename:   "Next.mkv",
		LocalTitle: "Next",
		LocalYear:  2026,
		Movie: models.Movie{
			TMDBID:         202,
			Title:          "Next",
			OriginalTitle:  "Next",
			Year:           2026,
			RuntimeMinutes: 30,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	thirdMovie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Later.mkv"),
		Filename:   "Later.mkv",
		LocalTitle: "Later",
		LocalYear:  2026,
		Movie: models.Movie{
			TMDBID:         203,
			Title:          "Later",
			OriginalTitle:  "Later",
			Year:           2026,
			RuntimeMinutes: 30,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Movies", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := testServer.db.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: firstMovie.ID, MediaType: "movie", StartTime: now.Add(-10 * time.Minute), EndTime: now.Add(20 * time.Minute)},
		{ChannelID: channel.ID, MediaID: secondMovie.ID, MediaType: "movie", StartTime: now.Add(20 * time.Minute), EndTime: now.Add(50 * time.Minute)},
		{ChannelID: channel.ID, MediaID: thirdMovie.ID, MediaType: "movie", StartTime: now.Add(50 * time.Minute), EndTime: now.Add(80 * time.Minute)},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/schedule?current=1", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Programs []json.RawMessage `json:"programs"`
		Count    int               `json:"count"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Count != 1 || len(payload.Programs) != 1 {
		t.Fatalf("schedule count = %d programs = %d, body = %s", payload.Count, len(payload.Programs), response.Body.String())
	}
}

func TestTuneChannelWithNoCurrentProgramReturnsNullProgram(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	movie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   "/movies/Future.mkv",
		Filename:   "Future.mkv",
		LocalTitle: "Future",
		LocalYear:  2026,
		Movie: models.Movie{
			TMDBID:         100,
			Title:          "Future",
			OriginalTitle:  "Future",
			Year:           2026,
			RuntimeMinutes: 90,
			Genres:         []string{"Action"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Future Channel", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := testServer.db.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: movie.ID, MediaType: "movie", StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour)},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"program":null`) || !strings.Contains(response.Body.String(), `"next_program":null`) {
		t.Fatalf("tune response = %s", response.Body.String())
	}
}

func TestTuneChannelSkipsPaddedProgramAfterPlayableDuration(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	movie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   "/movies/Short.mkv",
		Filename:   "Short.mkv",
		LocalTitle: "Short",
		LocalYear:  2026,
		Movie: models.Movie{
			TMDBID:         101,
			Title:          "Short",
			OriginalTitle:  "Short",
			Year:           2026,
			RuntimeMinutes: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Padded", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := testServer.db.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: movie.ID, MediaType: "movie", StartTime: now.Add(-2 * time.Minute), EndTime: now.Add(28 * time.Minute)},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"program":null`) {
		t.Fatalf("tune response = %s", response.Body.String())
	}
}

func TestTuneChannelNotFound(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	request := httptest.NewRequest(http.MethodGet, "/api/channels/999/tune", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func TestDeleteTVShowEndpoint(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	show, err := testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              "/tv/Columbo (1971)",
		LocalTitle:        "Columbo",
		LocalYear:         1971,
		LocalEpisodeCount: 10,
		TVShow: models.TVShow{
			TMDBID:       873,
			Name:         "Columbo",
			OriginalName: "Columbo",
			Year:         1971,
			FirstAirDate: "1971-09-15",
			Overview:     "Overview",
			PosterPath:   "/poster.jpg",
			BackdropPath: "/backdrop.jpg",
			Genres:       []string{"Mystery"},
			Rating:       "TV-PG",
			Seasons:      1,
			Episodes:     10,
			Network:      "NBC",
		},
		PosterLocalPath:   "data/artwork/tv-poster.jpg",
		BackdropLocalPath: "data/artwork/tv-backdrop.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/tvshows/"+strconv.FormatInt(show.ID, 10), nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/tvshows", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "Columbo") {
		t.Fatalf("deleted TV show still listed: %s", response.Body.String())
	}
}

func TestTVChannelScheduleUsesTVShows(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	showDir := t.TempDir()
	episodePath := filepath.Join(showDir, "Columbo.S01E01.mkv")
	if err := os.WriteFile(episodePath, []byte("episode"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              showDir,
		LocalTitle:        "Columbo",
		LocalYear:         1971,
		LocalEpisodeCount: 1,
		EpisodePaths:      []string{episodePath},
		TVShow: models.TVShow{
			TMDBID:       873,
			Name:         "Columbo",
			OriginalName: "Columbo",
			Year:         1971,
			FirstAirDate: "1971-09-15",
			Overview:     "Detective stories",
			PosterPath:   "/poster.jpg",
			BackdropPath: "/backdrop.jpg",
			Genres:       []string{"Mystery"},
			Rating:       "TV-PG",
			Seasons:      1,
			Episodes:     1,
			Network:      "NBC",
		},
		PosterLocalPath:   "data/artwork/tv-poster.jpg",
		BackdropLocalPath: "data/artwork/tv-backdrop.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{
		Name:        "Mystery TV",
		ContentType: "tv",
		Rules:       []models.ChannelRule{{Field: "genre", Operator: "is", Value: "Mystery"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := testServer.scheduler.GenerateChannel(context.Background(), channel.ID); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/schedule", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"type":"tv"`) ||
		!strings.Contains(response.Body.String(), "Columbo") ||
		!strings.Contains(response.Body.String(), "http://retrotv.test/media/tvshows/") {
		t.Fatalf("schedule = %s", response.Body.String())
	}
}

func TestTVTuneAndXMLTVUseScheduledEpisodeMetadata(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	showDir := t.TempDir()
	episodePath := filepath.Join(showDir, "Martin.S01E02.mkv")
	if err := os.WriteFile(episodePath, []byte("episode"), 0o644); err != nil {
		t.Fatal(err)
	}
	thumbnailPath := "data/artwork/tvshows/171/seasons/01/episodes/02.jpg"
	show, err := testServer.db.UpsertTVShow(context.Background(), models.TVShowRecord{
		Path:              showDir,
		LocalTitle:        "Martin",
		LocalYear:         1992,
		LocalEpisodeCount: 1,
		EpisodePaths:      []string{episodePath},
		EpisodeMetadata: []models.TVEpisode{
			{
				Path:               episodePath,
				Season:             1,
				Episode:            2,
				Name:               "The Gift Rapper",
				Overview:           "Martin tries to pick the right anniversary gift.",
				ThumbnailLocalPath: thumbnailPath,
				RuntimeMinutes:     22,
			},
		},
		TVShow: models.TVShow{
			TMDBID:       171,
			Name:         "Martin",
			OriginalName: "Martin",
			Year:         1992,
			FirstAirDate: "1992-08-27",
			Overview:     "Show-level description",
			PosterPath:   "/poster.jpg",
			BackdropPath: "/backdrop.jpg",
			Genres:       []string{"Comedy"},
			Rating:       "TV-PG",
			Seasons:      1,
			Episodes:     1,
			Network:      "FOX",
		},
		PosterLocalPath: "data/artwork/tv-poster.jpg",
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "FOX", ContentType: "tv"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := testServer.db.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: show.ID, MediaType: "tv", EpisodePath: episodePath, StartTime: now.Add(-5 * time.Minute), EndTime: now.Add(17 * time.Minute)},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/api/channels/"+strconv.FormatInt(channel.ID, 10)+"/tune", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Martin S01E02 - The Gift Rapper") ||
		!strings.Contains(response.Body.String(), "Martin tries to pick the right anniversary gift.") ||
		!strings.Contains(response.Body.String(), "tv-poster.jpg") ||
		strings.Contains(response.Body.String(), "tvshows/171/seasons/01/episodes/02.jpg") ||
		strings.Contains(response.Body.String(), "Show-level description") {
		t.Fatalf("tune response = %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "http://retrotv.test/channels/"+strconv.FormatInt(channel.ID, 10)+"/xmltv", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Martin S01E02 - The Gift Rapper") ||
		!strings.Contains(response.Body.String(), "Martin tries to pick the right anniversary gift.") ||
		!strings.Contains(response.Body.String(), "tv-poster.jpg") ||
		strings.Contains(response.Body.String(), "tvshows/171/seasons/01/episodes/02.jpg") ||
		strings.Contains(response.Body.String(), "Show-level description") {
		t.Fatalf("xmltv response = %s", response.Body.String())
	}
}

func TestAggregateFeedsAreGeneratedFromChannels(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	moviePath := filepath.Join(t.TempDir(), "Big Trouble in Little China (1986)", "feature.mkv")
	if err := os.MkdirAll(filepath.Dir(moviePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   moviePath,
		Filename:   "feature.mkv",
		LocalTitle: "Big Trouble in Little China",
		LocalYear:  1986,
		Movie: models.Movie{
			TMDBID:         6978,
			Title:          "Big Trouble in Little China",
			OriginalTitle:  "Big Trouble in Little China",
			Year:           1986,
			ReleaseDate:    "1986-07-02",
			Overview:       "Adventure overview",
			RuntimeMinutes: 99,
			Genres:         []string{"Adventure"},
			VoteAverage:    7.2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{
		Name:        "Adventure",
		LogoPath:    "/logos/adventure.png",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{Field: "genre", Operator: "is", Value: "Adventure"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := testServer.scheduler.GenerateChannel(context.Background(), channel.ID); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "http://retrotv.test/channels.m3u", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Adventure") ||
		!strings.Contains(response.Body.String(), `tvg-logo="http://retrotv.test/logos/adventure.png"`) ||
		!strings.Contains(response.Body.String(), "http://retrotv.test/channels/"+strconv.FormatInt(channel.ID, 10)+"/m3u") {
		t.Fatalf("channels.m3u = %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/guide.xml", nil)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `channel id="retrotv-`+strconv.FormatInt(channel.ID, 10)+`"`) ||
		!strings.Contains(response.Body.String(), `src="http://example.com/logos/adventure.png"`) ||
		!strings.Contains(response.Body.String(), "Big Trouble in Little China") {
		t.Fatalf("guide.xml = %s", response.Body.String())
	}
}

func TestXMLTVGuideHidesAssetsAndExtendsPreviousProgram(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	firstMovie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "First.mkv"),
		Filename:   "First.mkv",
		LocalTitle: "First Movie",
		LocalYear:  2026,
		Movie: models.Movie{
			TMDBID:         901,
			Title:          "First Movie",
			OriginalTitle:  "First Movie",
			Year:           2026,
			RuntimeMinutes: 22,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	secondMovie, err := testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   filepath.Join(t.TempDir(), "Second.mkv"),
		Filename:   "Second.mkv",
		LocalTitle: "Second Movie",
		LocalYear:  2026,
		Movie: models.Movie{
			TMDBID:         902,
			Title:          "Second Movie",
			OriginalTitle:  "Second Movie",
			Year:           2026,
			RuntimeMinutes: 30,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	asset, err := testServer.db.UpsertChannelAsset(context.Background(), models.ChannelAsset{
		FilePath:        filepath.Join(t.TempDir(), "station-id.mp4"),
		Filename:        "station-id.mp4",
		AssetType:       "station_id",
		Scope:           "global",
		DurationSeconds: 8 * 60,
		Enabled:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	channel, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Guide", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now().UTC().Truncate(time.Second).Add(-time.Minute)
	firstEnd := start.Add(22 * time.Minute)
	slotEnd := start.Add(30 * time.Minute)
	secondEnd := slotEnd.Add(30 * time.Minute)
	if err := testServer.db.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: firstMovie.ID, MediaType: "movie", StartTime: start, EndTime: firstEnd},
		{ChannelID: channel.ID, MediaID: asset.ID, MediaType: "asset", StartTime: firstEnd, EndTime: slotEnd},
		{ChannelID: channel.ID, MediaID: secondMovie.ID, MediaType: "movie", StartTime: slotEnd, EndTime: secondEnd},
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/guide.xml?hours=2", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "station-id.mp4") || strings.Contains(body, "Station ID") {
		t.Fatalf("guide.xml exposed asset row: %s", body)
	}
	if strings.Count(body, "<programme ") != 2 {
		t.Fatalf("guide.xml programme count mismatch: %s", body)
	}
	if !strings.Contains(body, `start="`+xmltvTime(start)+`" stop="`+xmltvTime(slotEnd)+`"`) ||
		!strings.Contains(body, "First Movie") ||
		!strings.Contains(body, "Second Movie") {
		t.Fatalf("guide.xml did not merge asset time into visible programming: %s", body)
	}
}

func TestXMLTVWindowAcceptsExplicitStart(t *testing.T) {
	now := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	wantStart := time.Date(2026, time.July, 21, 5, 0, 0, 0, time.UTC)
	request := httptest.NewRequest(http.MethodGet, "/guide.xml?start="+wantStart.Format(time.RFC3339)+"&hours=24", nil)

	start, end := xmltvWindow(request, now)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %s, want %s", start, wantStart)
	}
	if wantEnd := wantStart.Add(24 * time.Hour); !end.Equal(wantEnd) {
		t.Fatalf("end = %s, want %s", end, wantEnd)
	}
}

func TestSavingChannelQueuesScheduleGeneration(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	moviePath := filepath.Join(t.TempDir(), "16_Blocks_(2006)", "16_Blocks_(2006).mkv")
	if err := os.MkdirAll(filepath.Dir(moviePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = testServer.db.UpsertMovie(context.Background(), models.MovieRecord{
		FilePath:   moviePath,
		Filename:   "16_Blocks_(2006).mkv",
		LocalTitle: "16 Blocks",
		LocalYear:  2006,
		Movie: models.Movie{
			TMDBID:         2207,
			Title:          "16 Blocks",
			OriginalTitle:  "16 Blocks",
			Year:           2006,
			ReleaseDate:    "2006-03-01",
			Overview:       "Overview",
			RuntimeMinutes: 105,
			Genres:         []string{"Action"},
			VoteAverage:    6.4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{
		"name":"Action",
		"content_type":"movie",
		"rules":[{"field":"genre","operator":"is","value":"Action"}]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/channels", body)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var channel models.Channel
	if err := json.NewDecoder(response.Body).Decode(&channel); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		request = httptest.NewRequest(http.MethodGet, "/channels/"+strconv.FormatInt(channel.ID, 10)+"/xmltv", nil)
		response = httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "16 Blocks") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("xmltv did not include queued schedule item before timeout")
}

func TestClientStartupNoChannels(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	request := httptest.NewRequest(http.MethodGet, "/api/client/startup", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Status        string `json:"status"`
		ServerVersion string `json:"server_version"`
		ChannelCount  int    `json:"channel_count"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "no_channels" || payload.ChannelCount != 0 || payload.ServerVersion == "" {
		t.Fatalf("startup payload = %#v", payload)
	}
}

func TestClientStartupReady(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Action", ContentType: "movie"}); err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	request := httptest.NewRequest(http.MethodGet, "/api/client/startup", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Status       string `json:"status"`
		ChannelCount int    `json:"channel_count"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "ready" || payload.ChannelCount != 1 {
		t.Fatalf("startup payload = %#v", payload)
	}
}

func TestClientStartupScheduleGenerating(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Action", ContentType: "movie"}); err != nil {
		t.Fatal(err)
	}
	testServer.beginScan("movie", "Generating channel schedules", "Refreshing lineups")
	server := testServer.Handler()

	request := httptest.NewRequest(http.MethodGet, "/api/client/startup", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Status       string `json:"status"`
		ChannelCount int    `json:"channel_count"`
		ScanActive   bool   `json:"scan_active"`
		ScanPhase    string `json:"scan_phase"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "schedule_generating" || payload.ChannelCount != 1 || !payload.ScanActive || payload.ScanPhase != "Generating channel schedules" {
		t.Fatalf("startup payload = %#v", payload)
	}
}

func TestChannelsCanBeReordered(t *testing.T) {
	testServer, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := testServer.Handler()

	first, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "First", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Second", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	third, err := testServer.db.SaveChannel(context.Background(), models.Channel{Name: "Third", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{"channel_ids":[` +
		strconv.FormatInt(third.ID, 10) + `,` +
		strconv.FormatInt(first.ID, 10) + `,` +
		strconv.FormatInt(second.ID, 10) + `]}`)
	request := httptest.NewRequest(http.MethodPut, "/api/channels/reorder", body)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"name":"Third"`) ||
		strings.Index(response.Body.String(), `"name":"Third"`) > strings.Index(response.Body.String(), `"name":"First"`) ||
		strings.Index(response.Body.String(), `"name":"First"`) > strings.Index(response.Body.String(), `"name":"Second"`) {
		t.Fatalf("channels not reordered = %s", response.Body.String())
	}
}

func quote(t *testing.T, value string) string {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type fakePersistentStreamStarter struct {
	starts    int
	processes []*fakePersistentStreamProcess
}

func (s *fakePersistentStreamStarter) StartSession(ctx context.Context, request playback.ChannelRequest) (playback.SessionProcess, error) {
	s.starts++
	process := &fakePersistentStreamProcess{
		playback: playback.Playback{
			URL:          request.BaseURL + "/live/channel/" + strconv.FormatInt(request.ChannelID, 10),
			SeekPosition: request.Request.SeekPosition,
		},
		done: make(chan error),
	}
	s.processes = append(s.processes, process)
	return process, nil
}

type fakePersistentStreamProcess struct {
	playback playback.Playback
	done     chan error
	stopped  bool
}

func (p *fakePersistentStreamProcess) Playback() playback.Playback {
	return p.playback
}

func (p *fakePersistentStreamProcess) Stop(ctx context.Context) error {
	p.stopped = true
	select {
	case <-p.done:
	default:
		close(p.done)
	}
	return nil
}

func (p *fakePersistentStreamProcess) Done() <-chan error {
	return p.done
}
