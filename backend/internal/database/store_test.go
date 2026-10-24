package database

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"retrotv/internal/models"
)

func TestStoreUpsertMovie(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	record := models.MovieRecord{
		FilePath:   "/movies/Die Hard (1988)/feature.mkv",
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
			Genres:         []string{"Action", "Thriller"},
			VoteAverage:    7.8,
		},
		PosterLocalPath:   "data/artwork/562-poster.jpg",
		BackdropLocalPath: "data/artwork/562-backdrop.jpg",
	}

	saved, err := store.UpsertMovie(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == 0 {
		t.Fatal("saved ID is 0")
	}

	record.Overview = "Updated"
	updated, err := store.UpsertMovie(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != saved.ID {
		t.Fatalf("updated ID = %d, want %d", updated.ID, saved.ID)
	}
	if updated.Overview != "Updated" {
		t.Fatalf("Overview = %q, want Updated", updated.Overview)
	}

	movies, err := store.Movies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 {
		t.Fatalf("len(movies) = %d, want 1", len(movies))
	}
}

func TestRelocateMoviePreservesIDAndClearsMissing(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	original := testMovieRecord("/old/Heat.mkv", "Heat", 1995, 170, 8.3, []string{"Crime"})
	original.MediaMetadata = models.MediaMetadata{FileSize: 1234, ModTimeUnix: 99}
	saved, err := store.UpsertMovie(ctx, original)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReconcileMoviePresence(ctx, []string{"/old"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}

	relocated, err := store.RelocateMovie(ctx, saved.ID, "/new/Heat.mkv", "Heat.mkv", "Heat", 1995, original.MediaMetadata)
	if err != nil {
		t.Fatal(err)
	}
	if relocated.ID != saved.ID || relocated.FilePath != "/new/Heat.mkv" || relocated.MissingSince != nil {
		t.Fatalf("relocated movie = %#v", relocated)
	}
}

func TestMarkMoviesOutsideRootsMissing(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	inside, err := store.UpsertMovie(ctx, testMovieRecord("/current/Inside.mkv", "Inside", 2000, 90, 7, nil))
	if err != nil {
		t.Fatal(err)
	}
	outside, err := store.UpsertMovie(ctx, testMovieRecord("/retired/Outside.mkv", "Outside", 2001, 90, 7, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkMoviesOutsideRootsMissing(ctx, []string{"/current"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	inside, _ = store.MovieByID(ctx, inside.ID)
	outside, _ = store.MovieByID(ctx, outside.ID)
	if inside.MissingSince != nil || outside.MissingSince == nil {
		t.Fatalf("inside missing=%v, outside missing=%v", inside.MissingSince, outside.MissingSince)
	}
}

func TestMergeMovieRecordsTransfersReferences(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	stale, err := store.UpsertMovie(ctx, testMovieRecord("/old/Heat.mkv", "Heat", 1995, 170, 8, nil))
	if err != nil {
		t.Fatal(err)
	}
	active, err := store.UpsertMovie(ctx, testMovieRecord("/new/Heat.mkv", "Heat", 1995, 170, 8, nil))
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.db.ExecContext(ctx, `INSERT INTO collections (name) VALUES ('Favorites')`)
	if err != nil {
		t.Fatal(err)
	}
	collectionID, _ := result.LastInsertId()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO collection_media (collection_id, media_type, media_id) VALUES (?, 'movie', ?)`, collectionID, stale.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO channels (name, logo_path, content_type) VALUES ('Movies', '', 'movie')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO scheduled_programs (channel_id, media_id, media_type, start_time, end_time) SELECT id, ?, 'movie', '2026-01-01T00:00:00Z', '2026-01-01T02:00:00Z' FROM channels WHERE name = 'Movies'`, stale.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.MergeMovieRecords(ctx, stale.ID, active.ID); err != nil {
		t.Fatal(err)
	}
	var collectionMediaID, scheduledMediaID int64
	if err := store.db.QueryRowContext(ctx, `SELECT media_id FROM collection_media WHERE collection_id = ?`, collectionID).Scan(&collectionMediaID); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT media_id FROM scheduled_programs LIMIT 1`).Scan(&scheduledMediaID); err != nil {
		t.Fatal(err)
	}
	var staleCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM movies WHERE id = ?`, stale.ID).Scan(&staleCount); err != nil {
		t.Fatal(err)
	}
	if collectionMediaID != active.ID || scheduledMediaID != active.ID || staleCount != 0 {
		t.Fatalf("collection=%d schedule=%d stale_count=%d, want active=%d", collectionMediaID, scheduledMediaID, staleCount, active.ID)
	}
}

func TestStoreDeleteMovieRemovesSchedules(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	movie, err := store.UpsertMovie(context.Background(), testMovieRecord("/movies/Missing.mkv", "Missing", 1988, 90, 6.5, []string{"Action"}))
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(context.Background(), models.Channel{Name: "Action", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC()
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: movie.ID, StartTime: start, EndTime: start.Add(90 * time.Minute)},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteMovie(context.Background(), movie.ID); err != nil {
		t.Fatal(err)
	}

	movies, err := store.Movies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 0 {
		t.Fatalf("len(movies) = %d, want 0", len(movies))
	}
	programs, err := store.ScheduledPrograms(context.Background(), channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 0 {
		t.Fatalf("len(programs) = %d, want 0", len(programs))
	}
}

func TestReconcileMoviePresenceMarksMissingAndRestores(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := filepath.Join(t.TempDir(), "movies")
	presentPath := filepath.Join(root, "Present (2000).mkv")
	missingPath := filepath.Join(root, "Renamed.mkv")
	outsidePath := filepath.Join(t.TempDir(), "other", "Outside.mkv")
	for _, path := range []string{presentPath, missingPath, outsidePath} {
		if _, err := store.UpsertMovie(context.Background(), testMovieRecord(path, filepath.Base(path), 2000, 90, 7, nil)); err != nil {
			t.Fatal(err)
		}
	}

	firstScan := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	if err := store.ReconcileMoviePresence(context.Background(), []string{root}, []string{presentPath}, firstScan); err != nil {
		t.Fatal(err)
	}
	missing, err := store.MovieByFilePath(context.Background(), missingPath)
	if err != nil {
		t.Fatal(err)
	}
	if missing.MissingSince == nil || !missing.MissingSince.Equal(firstScan) {
		t.Fatalf("missing_since = %v, want %v", missing.MissingSince, firstScan)
	}
	outside, err := store.MovieByFilePath(context.Background(), outsidePath)
	if err != nil {
		t.Fatal(err)
	}
	if outside.MissingSince != nil {
		t.Fatalf("outside source was marked missing: %v", outside.MissingSince)
	}

	secondScan := firstScan.Add(time.Hour)
	if err := store.ReconcileMoviePresence(context.Background(), []string{root}, []string{presentPath, missingPath}, secondScan); err != nil {
		t.Fatal(err)
	}
	restored, err := store.MovieByFilePath(context.Background(), missingPath)
	if err != nil {
		t.Fatal(err)
	}
	if restored.MissingSince != nil || !restored.LastSeenAt.Equal(secondScan) {
		t.Fatalf("restored presence = missing %v, last seen %v", restored.MissingSince, restored.LastSeenAt)
	}
}

func TestReconcileMoviesWithNoSuccessfulRootsLeavesPresenceUnchanged(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	path := filepath.Join(t.TempDir(), "offline", "Movie.mkv")
	if _, err := store.UpsertMovie(context.Background(), testMovieRecord(path, "Movie", 2000, 90, 7, nil)); err != nil {
		t.Fatal(err)
	}
	if err := store.ReconcileMoviePresence(context.Background(), nil, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	movie, err := store.MovieByFilePath(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if movie.MissingSince != nil {
		t.Fatalf("offline source record was marked missing: %v", movie.MissingSince)
	}
}

func TestDeleteMovieNeverDeletesMedia(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	moviePath := filepath.Join(t.TempDir(), "Movie (2000).mkv")
	if err := os.WriteFile(moviePath, []byte("media"), 0o644); err != nil {
		t.Fatal(err)
	}
	movie, err := store.UpsertMovie(context.Background(), testMovieRecord(moviePath, "Movie", 2000, 90, 7, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteMovie(context.Background(), movie.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(moviePath); err != nil {
		t.Fatalf("media file changed after database deletion: %v", err)
	}
}

func TestGuideScheduledProgramsWindowLeavesAssetsLightweight(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	movie, err := store.UpsertMovie(context.Background(), testMovieRecord("/movies/Guide.mkv", "Guide Movie", 2026, 90, 7.0, []string{"Drama"}))
	if err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(context.Background(), models.Channel{Name: "Guide", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Truncate(time.Second)
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: movie.ID, MediaType: "movie", StartTime: start, EndTime: start.Add(22 * time.Minute)},
		{ChannelID: channel.ID, MediaID: 999999, MediaType: "asset", StartTime: start.Add(22 * time.Minute), EndTime: start.Add(30 * time.Minute)},
	}); err != nil {
		t.Fatal(err)
	}

	programsByChannel, err := store.GuideScheduledProgramsWindow(
		context.Background(), []int64{channel.ID}, start.Add(-time.Minute), start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	programs := programsByChannel[channel.ID]
	if len(programs) != 2 {
		t.Fatalf("len(programs) = %d, want 2", len(programs))
	}
	if programs[0].Media.Title != "Guide Movie" {
		t.Fatalf("movie title = %q, want Guide Movie", programs[0].Media.Title)
	}
	if programs[1].MediaType != "asset" || programs[1].Asset.ID != 0 {
		t.Fatalf("asset row was unexpectedly hydrated: %#v", programs[1])
	}
}

func TestStoreDeleteTVShow(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	show, err := store.UpsertTVShow(context.Background(), testTVShowRecord("/tv/Columbo (1971)", "Columbo", 1971))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTVShow(context.Background(), show.ID); err != nil {
		t.Fatal(err)
	}

	shows, err := store.TVShows(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(shows) != 0 {
		t.Fatalf("len(shows) = %d, want 0", len(shows))
	}
}

func TestReconcileTVShowPresenceMarksMissingAndRestores(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	root := filepath.Join(t.TempDir(), "tv")
	presentPath := filepath.Join(root, "Present (2000)")
	missingPath := filepath.Join(root, "Renamed")
	outsidePath := filepath.Join(t.TempDir(), "other", "Outside")
	for _, path := range []string{presentPath, missingPath, outsidePath} {
		if _, err := store.UpsertTVShow(context.Background(), testTVShowRecord(path, filepath.Base(path), 2000)); err != nil {
			t.Fatal(err)
		}
	}

	firstScan := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	if err := store.ReconcileTVShowPresence(context.Background(), []string{root}, []string{presentPath}, firstScan); err != nil {
		t.Fatal(err)
	}
	missing, err := store.TVShowByPath(context.Background(), missingPath)
	if err != nil {
		t.Fatal(err)
	}
	if missing.MissingSince == nil || !missing.MissingSince.Equal(firstScan) {
		t.Fatalf("missing_since = %v, want %v", missing.MissingSince, firstScan)
	}
	outside, err := store.TVShowByPath(context.Background(), outsidePath)
	if err != nil {
		t.Fatal(err)
	}
	if outside.MissingSince != nil {
		t.Fatalf("outside source was marked missing: %v", outside.MissingSince)
	}

	secondScan := firstScan.Add(time.Hour)
	if err := store.ReconcileTVShowPresence(context.Background(), []string{root}, []string{presentPath, missingPath}, secondScan); err != nil {
		t.Fatal(err)
	}
	restored, err := store.TVShowByPath(context.Background(), missingPath)
	if err != nil {
		t.Fatal(err)
	}
	if restored.MissingSince != nil || !restored.LastSeenAt.Equal(secondScan) {
		t.Fatalf("restored presence = missing %v, last seen %v", restored.MissingSince, restored.LastSeenAt)
	}
}

func TestRelocateTVShowPreservesIDAndEpisodeMetadata(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	record := testTVShowRecord("/old/Columbo", "Columbo", 1971)
	record.EpisodePaths = []string{"/old/Columbo/S01E01.mkv"}
	record.LocalEpisodeCount = 1
	record.EpisodeMetadata = []models.TVEpisode{{
		Path: "/old/Columbo/S01E01.mkv", Season: 1, Episode: 1,
		MediaMetadata: models.MediaMetadata{FileSize: 4321, ModTimeUnix: 88},
	}}
	saved, err := store.UpsertTVShow(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReconcileTVShowPresence(ctx, []string{"/old"}, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	episodes := record.EpisodeMetadata
	episodes[0].Path = "/new/Columbo/S01E01.mkv"
	relocated, err := store.RelocateTVShow(ctx, saved.ID, "/new/Columbo", "Columbo", 1971, []string{episodes[0].Path}, episodes)
	if err != nil {
		t.Fatal(err)
	}
	if relocated.ID != saved.ID || relocated.Path != "/new/Columbo" || relocated.MissingSince != nil {
		t.Fatalf("relocated show = %#v", relocated)
	}
	if len(relocated.EpisodeMetadata) != 1 || relocated.EpisodeMetadata[0].Path != episodes[0].Path {
		t.Fatalf("episode metadata = %#v", relocated.EpisodeMetadata)
	}
}

func TestReconcileWithNoSuccessfulRootsLeavesPresenceUnchanged(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	path := filepath.Join(t.TempDir(), "offline", "Friends")
	if _, err := store.UpsertTVShow(context.Background(), testTVShowRecord(path, "Friends", 1994)); err != nil {
		t.Fatal(err)
	}
	if err := store.ReconcileTVShowPresence(context.Background(), nil, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	show, err := store.TVShowByPath(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if show.MissingSince != nil {
		t.Fatalf("offline source record was marked missing: %v", show.MissingSince)
	}
}

func TestDeleteTVShowNeverDeletesMedia(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	showPath := filepath.Join(t.TempDir(), "Friends (1994)")
	episodePath := filepath.Join(showPath, "S01E01.mkv")
	if err := os.MkdirAll(showPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(episodePath, []byte("media"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := testTVShowRecord(showPath, "Friends", 1994)
	record.EpisodePaths = []string{episodePath}
	show, err := store.UpsertTVShow(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTVShow(context.Background(), show.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(episodePath); err != nil {
		t.Fatalf("media file changed after database deletion: %v", err)
	}
}

func TestStoreMetadataFacets(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	records := []models.MovieRecord{
		testMovieRecord("/movies/Die Hard (1988)/feature.mkv", "Die Hard", 1988, 132, 7.8, []string{"Action", "Thriller"}),
		testMovieRecord("/movies/Heat (1995)/feature.mkv", "Heat", 1995, 170, 7.9, []string{"Action", "Crime"}),
	}
	records[0].Rating = "R"
	records[0].Actors = []string{"Bruce Willis"}
	records[0].Directors = []string{"John McTiernan"}
	records[0].Writers = []string{"Jeb Stuart"}
	records[0].Countries = []string{"US"}
	records[0].Languages = []string{"English"}
	records[0].Tags = []string{"christmas"}
	for _, record := range records {
		if _, err := store.UpsertMovie(context.Background(), record); err != nil {
			t.Fatal(err)
		}
	}
	columbo := testTVShowRecord("/tv/Columbo (1971)", "Columbo", 1971)
	columbo.Studio = "Universal Television"
	columbo.Actors = []string{"Peter Falk"}
	columbo.Directors = []string{"Steven Spielberg"}
	columbo.Writers = []string{"Richard Levinson"}
	columbo.Countries = []string{"US"}
	columbo.Languages = []string{"English"}
	columbo.SeriesStatus = "Ended"
	columbo.Tags = []string{"detective"}
	columbo.EpisodeMetadata = []models.TVEpisode{{Path: "/tv/Columbo/S01E01.mkv", RuntimeMinutes: 45}}
	if _, err := store.UpsertTVShow(context.Background(), columbo); err != nil {
		t.Fatal(err)
	}

	facets, err := store.MetadataFacets(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(facets.Titles, []string{"Columbo", "Die Hard", "Heat"}) {
		t.Fatalf("Titles = %#v", facets.Titles)
	}
	if !reflect.DeepEqual(facets.Genres, []string{"Action", "Crime", "Mystery", "Thriller"}) {
		t.Fatalf("Genres = %#v", facets.Genres)
	}
	if !reflect.DeepEqual(facets.ReleaseYears, []int{
		1970, 1971, 1972, 1973, 1974, 1975, 1976, 1977, 1978, 1979,
		1980, 1981, 1982, 1983, 1984, 1985, 1986, 1987, 1988, 1989,
		1990, 1991, 1992, 1993, 1994, 1995, 1996,
	}) {
		t.Fatalf("ReleaseYears = %#v", facets.ReleaseYears)
	}
	if !reflect.DeepEqual(facets.Runtimes, []int{45, 132, 170}) {
		t.Fatalf("Runtimes = %#v", facets.Runtimes)
	}
	if !reflect.DeepEqual(facets.VoteAverages, []float64{7.8, 7.9}) {
		t.Fatalf("VoteAverages = %#v", facets.VoteAverages)
	}
	if !reflect.DeepEqual(facets.Ratings, []string{"R", "TV-PG"}) {
		t.Fatalf("Ratings = %#v", facets.Ratings)
	}
	if !reflect.DeepEqual(facets.Studios, []string{"Die Hard Studio", "Heat Studio", "Universal Television"}) {
		t.Fatalf("Studios = %#v", facets.Studios)
	}
	if !reflect.DeepEqual(facets.Networks, []string{"NBC"}) {
		t.Fatalf("Networks = %#v", facets.Networks)
	}
	if !reflect.DeepEqual(facets.Actors, []string{"Bruce Willis", "Peter Falk"}) {
		t.Fatalf("Actors = %#v", facets.Actors)
	}
	if !reflect.DeepEqual(facets.Directors, []string{"John McTiernan", "Steven Spielberg"}) {
		t.Fatalf("Directors = %#v", facets.Directors)
	}
	if !reflect.DeepEqual(facets.Writers, []string{"Jeb Stuart", "Richard Levinson"}) {
		t.Fatalf("Writers = %#v", facets.Writers)
	}
	if !reflect.DeepEqual(facets.Countries, []string{"US"}) {
		t.Fatalf("Countries = %#v", facets.Countries)
	}
	if !reflect.DeepEqual(facets.Languages, []string{"English"}) {
		t.Fatalf("Languages = %#v", facets.Languages)
	}
	if !reflect.DeepEqual(facets.SeriesStatuses, []string{"Ended"}) {
		t.Fatalf("SeriesStatuses = %#v", facets.SeriesStatuses)
	}
	if !reflect.DeepEqual(facets.Tags, []string{"christmas", "detective"}) {
		t.Fatalf("Tags = %#v", facets.Tags)
	}
	if !reflect.DeepEqual(facets.ReleaseDates, []string{"1988-07-15"}) {
		t.Fatalf("ReleaseDates = %#v", facets.ReleaseDates)
	}
	if !reflect.DeepEqual(facets.FirstAirDates, []string{"1971-09-15"}) {
		t.Fatalf("FirstAirDates = %#v", facets.FirstAirDates)
	}
	if len(facets.DateAddedDates) != 1 || facets.DateAddedDates[0] != time.Now().UTC().Format("2006-01-02") {
		t.Fatalf("DateAddedDates = %#v", facets.DateAddedDates)
	}
}

func TestStoreSaveChannelReplacesRules(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "80s Action",
		LogoPath:    "/logos/80s.png",
		ContentType: "movie",
		Rules: []models.ChannelRule{
			{MediaType: "movie", Field: "genre", Operator: "is", Value: "Action"},
			{MediaType: "tv", Field: "year", Operator: "greater_than", Value: "1979"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if channel.ID == 0 {
		t.Fatal("channel ID is 0")
	}
	if len(channel.Rules) != 2 {
		t.Fatalf("len(channel.Rules) = %d, want 2", len(channel.Rules))
	}

	if channel.Rules[0].MediaType != "movie" || channel.Rules[1].MediaType != "tv" {
		t.Fatalf("channel.Rules = %#v", channel.Rules)
	}

	channel.Rules = []models.ChannelRule{{MediaType: "movie", Field: "year", Operator: "less_than", Value: "1990"}}
	updated, err := store.SaveChannel(context.Background(), channel)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Rules) != 1 {
		t.Fatalf("len(updated.Rules) = %d, want 1", len(updated.Rules))
	}
	if updated.Rules[0].MediaType != "movie" || updated.Rules[0].Field != "year" || updated.Rules[0].Operator != "less_than" {
		t.Fatalf("updated.Rules[0] = %#v", updated.Rules[0])
	}
}

func TestStoreSaveChannelPersistsProgrammingBlocks(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "Nick",
		ContentType: "mixed",
		Rules:       []models.ChannelRule{{MediaType: "tv", Field: "title", Operator: "is", Value: "Rugrats"}},
		ProgrammingBlocks: []models.ProgrammingBlock{{
			Title:         "Morning Cartoons",
			StartTime:     "06:00",
			EndTime:       "09:00",
			Days:          []string{"mon", "tue", "wed", "thu", "fri"},
			ContentSource: "media",
			RuleSet:       "Selected Media",
			Priority:      10,
			Enabled:       true,
			MediaItems: []models.ProgrammingBlockMedia{
				{MediaType: "tv", MediaID: 12},
				{MediaType: "movie", MediaID: 34},
			},
		}, {
			Title: "Default Programming", Placement: models.ProgrammingBlockPlacementAllDay,
			Days:          []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
			ContentSource: "collection", CollectionID: 42, Enabled: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(channel.ProgrammingBlocks) != 2 {
		t.Fatalf("len(channel.ProgrammingBlocks) = %d, want 2", len(channel.ProgrammingBlocks))
	}
	block := channel.ProgrammingBlocks[0]
	if block.Title != "Morning Cartoons" || block.StartTime != "06:00" || block.EndTime != "09:00" || block.Priority != 10 || !block.Enabled {
		t.Fatalf("block = %#v", block)
	}
	if !reflect.DeepEqual(block.Days, []string{"mon", "tue", "wed", "thu", "fri"}) {
		t.Fatalf("block.Days = %#v", block.Days)
	}
	if len(block.MediaItems) != 2 || block.MediaItems[0].MediaType != "tv" || block.MediaItems[1].MediaID != 34 {
		t.Fatalf("block.MediaItems = %#v", block.MediaItems)
	}
	allDay := channel.ProgrammingBlocks[1]
	if allDay.Placement != models.ProgrammingBlockPlacementAllDay || allDay.CollectionID != 42 || allDay.StartTime != "" || allDay.EndTime != "" {
		t.Fatalf("all-day block = %#v", allDay)
	}
}

func TestStoreSaveChannelPersistsPresentationSettings(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	channel, err := store.SaveChannel(context.Background(), models.Channel{
		Name:        "Branded",
		LogoPath:    "/artwork/channels/1/logo.png",
		ContentType: "movie",
		Presentation: models.ChannelPresentation{
			ChannelBugEnabled:            true,
			ChannelBugAssetPath:          "/artwork/channels/1/bug.png",
			ChannelBugPosition:           "top_right",
			ChannelBugWidth:              144,
			ChannelBugOpacity:            0.6,
			ChannelBugMarginX:            20,
			ChannelBugMarginY:            24,
			RatingOverlayEnabled:         true,
			RatingOverlayPosition:        "top_left",
			RatingOverlayDurationSeconds: 12,
			NowPlayingEnabled:            true,
			NowPlayingPosition:           "bottom_left",
			NowPlayingDurationSeconds:    9,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.Channel(context.Background(), channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Presentation != channel.Presentation {
		t.Fatalf("Presentation = %#v, want %#v", got.Presentation, channel.Presentation)
	}
}

func TestStoreReorderChannels(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	first, err := store.SaveChannel(context.Background(), models.Channel{Name: "First", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.SaveChannel(context.Background(), models.Channel{Name: "Second", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	third, err := store.SaveChannel(context.Background(), models.Channel{Name: "Third", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.ReorderChannels(context.Background(), []int64{third.ID, first.ID, second.ID}); err != nil {
		t.Fatal(err)
	}

	channels, err := store.Channels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, channel := range channels {
		names = append(names, channel.Name)
	}
	if !reflect.DeepEqual(names, []string{"Third", "First", "Second"}) {
		t.Fatalf("channel order = %#v", names)
	}
	if channels[0].SortOrder != 1 || channels[1].SortOrder != 2 || channels[2].SortOrder != 3 {
		t.Fatalf("sort orders = %#v", channels)
	}
}

func TestStoreSetChannelAiring(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	channel, err := store.SaveChannel(context.Background(), models.Channel{Name: "NBC", ContentType: "tv"})
	if err != nil {
		t.Fatal(err)
	}
	if !channel.Airing {
		t.Fatal("new channel should air by default")
	}
	if err := store.SetChannelAiring(context.Background(), channel.ID, false); err != nil {
		t.Fatal(err)
	}
	channel, err = store.Channel(context.Background(), channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if channel.Airing {
		t.Fatal("channel airing state was not disabled")
	}
}

func TestStoreMatchingMoviesAndSchedules(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dieHardRecord := testMovieRecord("/movies/Die Hard (1988)/feature.mkv", "Die Hard", 1988, 132, 7.8, []string{"Action", "Thriller"})
	dieHardRecord.Rating = "R"
	dieHardRecord.Actors = []string{"Bruce Willis"}
	dieHardRecord.Directors = []string{"John McTiernan"}
	dieHardRecord.Writers = []string{"Jeb Stuart"}
	dieHardRecord.Countries = []string{"US"}
	dieHardRecord.Languages = []string{"English"}
	dieHardRecord.Tags = []string{"christmas"}
	dieHard, err := store.UpsertMovie(context.Background(), dieHardRecord)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertMovie(context.Background(), testMovieRecord("/movies/Heat (1995)/feature.mkv", "Heat", 1995, 170, 7.9, []string{"Action", "Crime"})); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingMovies(context.Background(), []models.ChannelRule{
		{Field: "title", Operator: "is", Value: "Die Hard"},
		{Field: "genre", Operator: "is", Value: "Action"},
		{Field: "year", Operator: "less_than", Value: "1990"},
		{Field: "rating", Operator: "is", Value: "R"},
		{Field: "studio", Operator: "is", Value: "Die Hard Studio"},
		{Field: "studio", Operator: "is_not", Value: "Heat Studio"},
		{Field: "actor", Operator: "is", Value: "Bruce Willis"},
		{Field: "director", Operator: "is", Value: "John McTiernan"},
		{Field: "writer", Operator: "is", Value: "Jeb Stuart"},
		{Field: "country", Operator: "is", Value: "US"},
		{Field: "language", Operator: "is", Value: "English"},
		{Field: "release_date", Operator: "equals", Value: "1988-07-15"},
		{Field: "date_added", Operator: "equals", Value: time.Now().UTC().Format("2006-01-02")},
		{Field: "tag", Operator: "is", Value: "christmas"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Title != "Die Hard" {
		t.Fatalf("matches = %#v", matches)
	}

	matches, err = store.MatchingMovies(context.Background(), []models.ChannelRule{
		{Field: "title", Operator: "is_not", Value: "Die Hard"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Title != "Heat" {
		t.Fatalf("title is_not matches = %#v", matches)
	}

	matches, err = store.MatchingMovies(context.Background(), []models.ChannelRule{
		{Field: "title", Operator: "is", Value: "Die Hard"},
		{Field: "title", Operator: "is", Value: "Heat"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("title include matches = %#v", matches)
	}

	matches, err = store.MatchingMovies(context.Background(), []models.ChannelRule{
		{Field: "genre", Operator: "is", Value: "Action"},
		{Field: "genre", Operator: "is", Value: "Adventure"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("genre include fallback matches = %#v", matches)
	}

	matches, err = store.MatchingMovies(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("empty rules matched movies = %#v, want none", matches)
	}

	channel, err := store.SaveChannel(context.Background(), models.Channel{Name: "Test", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	start := dieHard.CreatedAt
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: dieHard.ID, StartTime: start, EndTime: start.Add(2 * time.Hour)},
	}); err != nil {
		t.Fatal(err)
	}
	programs, err := store.ScheduledPrograms(context.Background(), channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 || programs[0].Media.Title != "Die Hard" {
		t.Fatalf("programs = %#v", programs)
	}
}

func TestStoreMatchingTVShowsByTitle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := store.UpsertTVShow(context.Background(), testTVShowRecord("/tv/Columbo (1971)", "Columbo", 1971)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertTVShow(context.Background(), testTVShowRecord("/tv/Martin (1992)", "Martin", 1992)); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{Field: "title", Operator: "is", Value: "Martin"},
		{Field: "title", Operator: "is", Value: "Columbo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("matches = %#v", matches)
	}

	matches, err = store.MatchingTVShows(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("empty rules matched TV shows = %#v, want none", matches)
	}
}

func TestStoreMatchingTVShowsByLocalTitle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	localOnly := testTVShowRecord("/tv/Tom and Jerry", "Tom and Jerry", 0)
	localOnly.TMDBID = 0
	localOnly.Name = ""
	localOnly.OriginalName = ""
	localOnly.LocalTitle = "Tom and Jerry"
	if _, err := store.UpsertTVShow(context.Background(), localOnly); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{MediaType: "tv", Field: "title", Operator: "is", Value: "Tom and Jerry"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].LocalTitle != "Tom and Jerry" {
		t.Fatalf("local title matches = %#v", matches)
	}

	matches, err = store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{MediaType: "tv", Field: "title", Operator: "is_not", Value: "Tom and Jerry"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("local title exclusion matches = %#v, want none", matches)
	}
}

func TestStoreUpsertTVShowPersistsEpisodeMetadata(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	record := testTVShowRecord("/tv/Family Guy (1999)", "Family Guy", 1999)
	record.EpisodePaths = []string{"/tv/Family Guy/Season 04/Family.Guy.S04E01.North.by.North.Quahog.mkv"}
	record.EpisodeMetadata = []models.TVEpisode{
		{
			Path:               record.EpisodePaths[0],
			Season:             4,
			Episode:            1,
			Name:               "North by North Quahog",
			Overview:           "Brian and Stewie have a problem.",
			AirDate:            "2005-05-01",
			StillPath:          "/still.jpg",
			ThumbnailLocalPath: "data/artwork/tvshows/777/seasons/04/episodes/01.jpg",
			RuntimeMinutes:     22,
		},
	}

	saved, err := store.UpsertTVShow(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.EpisodeMetadata, record.EpisodeMetadata) {
		t.Fatalf("saved EpisodeMetadata = %#v, want %#v", saved.EpisodeMetadata, record.EpisodeMetadata)
	}

	shows, err := store.TVShows(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(shows) != 1 {
		t.Fatalf("len(shows) = %d, want 1", len(shows))
	}
	if !reflect.DeepEqual(shows[0].EpisodeMetadata, record.EpisodeMetadata) {
		t.Fatalf("listed EpisodeMetadata = %#v, want %#v", shows[0].EpisodeMetadata, record.EpisodeMetadata)
	}
}

func TestStoreMatchingTVShowsByNetwork(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	adultSwimShow := testTVShowRecord("/tv/Robot Chicken (2005)", "Robot Chicken", 2005)
	adultSwimShow.Network = "Adult Swim"
	if _, err := store.UpsertTVShow(context.Background(), adultSwimShow); err != nil {
		t.Fatal(err)
	}
	foxShow := testTVShowRecord("/tv/Family Guy (1999)", "Family Guy", 1999)
	foxShow.Network = "FOX"
	if _, err := store.UpsertTVShow(context.Background(), foxShow); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{Field: "network", Operator: "is", Value: "Adult Swim"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Name != "Robot Chicken" {
		t.Fatalf("network matches = %#v", matches)
	}
}

func TestStoreMatchingMoviesByQuality(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	hdMovie := testMovieRecord("/movies/Heat (1995)/feature.mkv", "Heat", 1995, 170, 8, []string{"Crime"})
	hdMovie.MediaMetadata.Height = 1080
	sdMovie := testMovieRecord("/movies/Ronin (1998)/feature.mkv", "Ronin", 1998, 122, 7, []string{"Crime"})
	sdMovie.MediaMetadata.Height = 480
	for _, movie := range []models.MovieRecord{hdMovie, sdMovie} {
		if _, err := store.UpsertMovie(context.Background(), movie); err != nil {
			t.Fatal(err)
		}
	}

	matches, err := store.MatchingMovies(context.Background(), []models.ChannelRule{
		{MediaType: "movie", Field: "quality", Operator: "is", Value: "hd"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].LocalTitle != "Heat" {
		t.Fatalf("hd matches = %#v", matches)
	}

	// Quality stays a hard constraint when other rules fall back to any-include.
	matches, err = store.MatchingMovies(context.Background(), []models.ChannelRule{
		{MediaType: "movie", Field: "genre", Operator: "is", Value: "Crime"},
		{MediaType: "movie", Field: "year", Operator: "equals", Value: "1970"},
		{MediaType: "movie", Field: "quality", Operator: "is_not", Value: "hd"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].LocalTitle != "Ronin" {
		t.Fatalf("fallback sd matches = %#v", matches)
	}
}

func TestStoreMatchingTVShowsByQualityNarrowsEpisodes(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mixed := testTVShowRecord("/tv/Seinfeld (1989)", "Seinfeld", 1989)
	mixed.EpisodePaths = []string{"/tv/Seinfeld (1989)/s01e01.mkv", "/tv/Seinfeld (1989)/s09e01.mkv"}
	mixed.EpisodeMetadata = []models.TVEpisode{
		{Path: "/tv/Seinfeld (1989)/s01e01.mkv", Season: 1, Episode: 1, MediaMetadata: models.MediaMetadata{Height: 480}},
		{Path: "/tv/Seinfeld (1989)/s09e01.mkv", Season: 9, Episode: 1, MediaMetadata: models.MediaMetadata{Height: 1080}},
	}
	sdOnly := testTVShowRecord("/tv/Cheers (1982)", "Cheers", 1982)
	sdOnly.EpisodePaths = []string{"/tv/Cheers (1982)/s01e01.mkv"}
	sdOnly.EpisodeMetadata = []models.TVEpisode{
		{Path: "/tv/Cheers (1982)/s01e01.mkv", Season: 1, Episode: 1, MediaMetadata: models.MediaMetadata{Height: 480}},
	}
	for _, show := range []models.TVShowRecord{mixed, sdOnly} {
		if _, err := store.UpsertTVShow(context.Background(), show); err != nil {
			t.Fatal(err)
		}
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{MediaType: "tv", Field: "title", Operator: "is", Value: "Seinfeld"},
		{MediaType: "tv", Field: "title", Operator: "is", Value: "Cheers"},
		{MediaType: "tv", Field: "quality", Operator: "is", Value: "hd"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].LocalTitle != "Seinfeld" {
		t.Fatalf("hd matches = %#v", matches)
	}
	if !matches[0].RestrictToEpisodePaths || len(matches[0].EpisodePaths) != 1 || matches[0].EpisodePaths[0] != "/tv/Seinfeld (1989)/s09e01.mkv" {
		t.Fatalf("hd episodes = %#v restricted=%v", matches[0].EpisodePaths, matches[0].RestrictToEpisodePaths)
	}
}

func TestStoreMatchingTVShowsByRuntimeAndDates(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	shortShow := testTVShowRecord("/tv/Daria (1997)", "Daria", 1997)
	shortShow.FirstAirDate = "1997-03-03"
	shortShow.EpisodeMetadata = []models.TVEpisode{
		{Path: "/tv/Daria/S01E01.mkv", RuntimeMinutes: 22},
		{Path: "/tv/Daria/S01E02.mkv", RuntimeMinutes: 22},
	}
	if _, err := store.UpsertTVShow(context.Background(), shortShow); err != nil {
		t.Fatal(err)
	}
	longShow := testTVShowRecord("/tv/Columbo (1971)", "Columbo", 1971)
	longShow.FirstAirDate = "1971-09-15"
	longShow.EpisodeMetadata = []models.TVEpisode{{Path: "/tv/Columbo/S01E01.mkv", RuntimeMinutes: 75}}
	if _, err := store.UpsertTVShow(context.Background(), longShow); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{MediaType: "tv", Field: "runtime", Operator: "less_than", Value: "30"},
		{MediaType: "tv", Field: "first_air_date", Operator: "greater_than", Value: "1990-01-01"},
		{MediaType: "tv", Field: "date_added", Operator: "equals", Value: time.Now().UTC().Format("2006-01-02")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Name != "Daria" {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestStoreMatchingTVShowsByExtendedMetadata(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	show := testTVShowRecord("/tv/Columbo (1971)", "Columbo", 1971)
	show.Studio = "Universal Television"
	show.Actors = []string{"Peter Falk"}
	show.Directors = []string{"Steven Spielberg"}
	show.Writers = []string{"Richard Levinson", "William Link"}
	show.Countries = []string{"US"}
	show.Languages = []string{"English"}
	show.SeriesStatus = "Ended"
	show.Tags = []string{"detective", "mystery"}
	if _, err := store.UpsertTVShow(context.Background(), show); err != nil {
		t.Fatal(err)
	}
	other := testTVShowRecord("/tv/Martin (1992)", "Martin", 1992)
	other.Studio = "HBO Independent Productions"
	other.Actors = []string{"Martin Lawrence"}
	other.Countries = []string{"US"}
	other.Languages = []string{"English"}
	other.SeriesStatus = "Ended"
	other.Tags = []string{"sitcom"}
	if _, err := store.UpsertTVShow(context.Background(), other); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{MediaType: "tv", Field: "studio", Operator: "is", Value: "Universal Television"},
		{MediaType: "tv", Field: "actor", Operator: "is", Value: "Peter Falk"},
		{MediaType: "tv", Field: "director", Operator: "is", Value: "Steven Spielberg"},
		{MediaType: "tv", Field: "writer", Operator: "is", Value: "William Link"},
		{MediaType: "tv", Field: "country", Operator: "is", Value: "US"},
		{MediaType: "tv", Field: "language", Operator: "is", Value: "English"},
		{MediaType: "tv", Field: "series_status", Operator: "is", Value: "Ended"},
		{MediaType: "tv", Field: "tag", Operator: "is", Value: "detective"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Name != "Columbo" {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestStoreMatchingTVShowsFallsBackToTitleOrNetworkIncludes(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	familyGuy := testTVShowRecord("/tv/Family Guy (1999)", "Family Guy", 1999)
	familyGuy.Network = "FOX"
	if _, err := store.UpsertTVShow(context.Background(), familyGuy); err != nil {
		t.Fatal(err)
	}
	robotChicken := testTVShowRecord("/tv/Robot Chicken (2005)", "Robot Chicken", 2005)
	robotChicken.Network = "Adult Swim"
	if _, err := store.UpsertTVShow(context.Background(), robotChicken); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{Field: "title", Operator: "is", Value: "Family Guy"},
		{Field: "network", Operator: "is", Value: "Adult Swim"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("fallback matches = %#v", matches)
	}
}

func TestStoreMatchingTVShowsUnionsIncludesWhenOneShowMatchesMultipleRules(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mrBean := testTVShowRecord("/tv/Mr Bean (1990)", "Mr. Bean", 1990)
	mrBean.Network = "ITV1"
	itCrowd := testTVShowRecord("/tv/The IT Crowd (2006)", "The IT Crowd", 2006)
	itCrowd.Network = "Channel 4"
	for _, show := range []models.TVShowRecord{mrBean, itCrowd} {
		if _, err := store.UpsertTVShow(context.Background(), show); err != nil {
			t.Fatal(err)
		}
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{MediaType: "tv", Field: "title", Operator: "is", Value: "Mr. Bean"},
		{MediaType: "tv", Field: "title", Operator: "is", Value: "The IT Crowd"},
		{MediaType: "tv", Field: "network", Operator: "is", Value: "ITV1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("matches = %#v, want both explicitly included shows", matches)
	}
}

func TestStoreMatchingTVShowsDoesNotFallbackForNonIncludeRules(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	foxShow := testTVShowRecord("/tv/Family Guy (1999)", "Family Guy", 1999)
	foxShow.Network = "FOX"
	if _, err := store.UpsertTVShow(context.Background(), foxShow); err != nil {
		t.Fatal(err)
	}
	hboShow := testTVShowRecord("/tv/The Wire (2002)", "The Wire", 2002)
	hboShow.Network = "HBO"
	if _, err := store.UpsertTVShow(context.Background(), hboShow); err != nil {
		t.Fatal(err)
	}

	matches, err := store.MatchingTVShows(context.Background(), []models.ChannelRule{
		{Field: "genre", Operator: "is", Value: "Action"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("genre fallback matches = %#v, want none", matches)
	}
}

func TestStoreMatchingRulesRespectMediaType(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dieHard := testMovieRecord("/movies/Die Hard (1988)/feature.mkv", "Die Hard", 1988, 132, 8.2, []string{"Action"})
	dieHard.Studio = "Fox"
	if _, err := store.UpsertMovie(context.Background(), dieHard); err != nil {
		t.Fatal(err)
	}
	heat := testMovieRecord("/movies/Heat (1995)/feature.mkv", "Heat", 1995, 170, 7.9, []string{"Crime"})
	heat.Studio = "Warner Bros"
	if _, err := store.UpsertMovie(context.Background(), heat); err != nil {
		t.Fatal(err)
	}

	familyGuy := testTVShowRecord("/tv/Family Guy (1999)", "Family Guy", 1999)
	familyGuy.Network = "FOX"
	if _, err := store.UpsertTVShow(context.Background(), familyGuy); err != nil {
		t.Fatal(err)
	}
	theWire := testTVShowRecord("/tv/The Wire (2002)", "The Wire", 2002)
	theWire.Network = "HBO"
	if _, err := store.UpsertTVShow(context.Background(), theWire); err != nil {
		t.Fatal(err)
	}

	rules := []models.ChannelRule{
		{MediaType: "movie", Field: "genre", Operator: "is", Value: "Action"},
		{MediaType: "tv", Field: "network", Operator: "is", Value: "FOX"},
	}
	movies, err := store.MatchingMovies(context.Background(), rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 || movies[0].Title != "Die Hard" {
		t.Fatalf("movie matches = %#v", movies)
	}

	shows, err := store.MatchingTVShows(context.Background(), rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(shows) != 1 || shows[0].Name != "Family Guy" {
		t.Fatalf("tv matches = %#v", shows)
	}
}

func TestStoreChannelScheduleStatePersistsEpisodeIndexes(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	channel, err := store.SaveChannel(context.Background(), models.Channel{Name: "TV", ContentType: "tv"})
	if err != nil {
		t.Fatal(err)
	}
	lastEnd := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	err = store.SaveChannelScheduleState(context.Background(), models.ChannelScheduleState{
		ChannelID:           channel.ID,
		NextMediaIndex:      2,
		LastMediaID:         42,
		EpisodeIndexes:      map[int64]int{42: 3, 99: 1},
		LastScheduleEndTime: lastEnd,
	})
	if err != nil {
		t.Fatal(err)
	}

	state, ok, err := store.ChannelScheduleState(context.Background(), channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("missing schedule state")
	}
	if state.NextMediaIndex != 2 || state.LastMediaID != 42 || !state.LastScheduleEndTime.Equal(lastEnd) {
		t.Fatalf("state = %#v", state)
	}
	if !reflect.DeepEqual(state.EpisodeIndexes, map[int64]int{42: 3, 99: 1}) {
		t.Fatalf("EpisodeIndexes = %#v", state.EpisodeIndexes)
	}
}

func TestMetadataFacetsExpandsReleaseYearRange(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := store.UpsertMovie(context.Background(), testMovieRecord("/movies/First.mkv", "First", 1991, 100, 6.5, []string{"Action"})); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertMovie(context.Background(), testMovieRecord("/movies/Second.mkv", "Second", 2006, 105, 6.8, []string{"Action"})); err != nil {
		t.Fatal(err)
	}

	facets, err := store.MetadataFacets(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(facets.ReleaseYears, []int{
		1990, 1991, 1992, 1993, 1994, 1995, 1996, 1997, 1998,
		1999, 2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007,
	}) {
		t.Fatalf("ReleaseYears = %#v", facets.ReleaseYears)
	}
}

func TestStorePersistsMediaMetadata(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	movie := testMovieRecord("/movies/Die Hard (1988)/feature.mkv", "Die Hard", 1988, 132, 7.8, []string{"Action"})
	movie.MediaMetadata = models.MediaMetadata{
		DurationSeconds: 7920.5,
		VideoCodec:      "h264",
		AudioCodec:      "aac",
		Width:           1920,
		Height:          1080,
		FrameRate:       23.976,
		FileSize:        1234,
		ModTimeUnix:     5678,
	}
	savedMovie, err := store.UpsertMovie(context.Background(), movie)
	if err != nil {
		t.Fatal(err)
	}
	if savedMovie.MediaMetadata.VideoCodec != "h264" || savedMovie.MediaMetadata.DurationSeconds != 7920.5 {
		t.Fatalf("movie media metadata = %#v", savedMovie.MediaMetadata)
	}

	show := testTVShowRecord("/tv/Daria (1997)", "Daria", 1997)
	show.EpisodePaths = []string{"/tv/Daria (1997)/Season_01/S01E01.mkv"}
	show.EpisodeMetadata = []models.TVEpisode{
		{
			Path:           show.EpisodePaths[0],
			Season:         1,
			Episode:        1,
			Name:           "Esteemsters",
			RuntimeMinutes: 22,
			MediaMetadata: models.MediaMetadata{
				DurationSeconds: 1320,
				VideoCodec:      "mpeg4",
				AudioTracks: []models.AudioTrack{
					{Index: 1, Codec: "mp3", Language: "eng", Channels: 2},
				},
				SubtitleTracks: []models.SubtitleTrack{
					{Index: 2, Codec: "subrip", Language: "eng"},
				},
				FileSize:    4321,
				ModTimeUnix: 8765,
			},
		},
	}
	savedShow, err := store.UpsertTVShow(context.Background(), show)
	if err != nil {
		t.Fatal(err)
	}
	if len(savedShow.EpisodeMetadata) != 1 {
		t.Fatalf("episodes = %#v", savedShow.EpisodeMetadata)
	}
	episodeMetadata := savedShow.EpisodeMetadata[0].MediaMetadata
	if episodeMetadata.VideoCodec != "mpeg4" || len(episodeMetadata.AudioTracks) != 1 || len(episodeMetadata.SubtitleTracks) != 1 {
		t.Fatalf("episode media metadata = %#v", episodeMetadata)
	}
}

func TestStoreUpsertsChannelAssets(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	asset, err := store.UpsertChannelAsset(context.Background(), models.ChannelAsset{
		ChannelID:       12,
		FilePath:        "/assets/bumpers/cold-open.mp4",
		Filename:        "cold-open.mp4",
		AssetType:       "bumper",
		Tag:             "any",
		Scope:           "global",
		DurationSeconds: 15,
		Enabled:         true,
		MediaMetadata:   models.MediaMetadata{DurationSeconds: 15.2, VideoCodec: "h264"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if asset.ID == 0 || asset.ChannelID != 12 || !asset.Enabled || asset.DurationSeconds != 15 || asset.MediaMetadata.VideoCodec != "h264" {
		t.Fatalf("asset = %#v", asset)
	}

	updated, err := store.UpsertChannelAsset(context.Background(), models.ChannelAsset{
		ChannelID:       34,
		FilePath:        "/assets/bumpers/cold-open.mp4",
		Filename:        "cold-open-v2.mp4",
		AssetType:       "bumper",
		Scope:           "global",
		DurationSeconds: 20,
		Enabled:         true,
		MediaMetadata:   models.MediaMetadata{DurationSeconds: 20},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != asset.ID || updated.ChannelID != 34 || updated.Filename != "cold-open-v2.mp4" || updated.DurationSeconds != 20 {
		t.Fatalf("updated = %#v, original = %#v", updated, asset)
	}

	assets, err := store.ChannelAssets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 {
		t.Fatalf("assets = %#v", assets)
	}
}

func TestStoreScheduledProgramsCanReferenceAssets(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	channel, err := store.SaveChannel(context.Background(), models.Channel{Name: "Assets", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	asset, err := store.UpsertChannelAsset(context.Background(), models.ChannelAsset{
		FilePath:        "/assets/commercials/ad.mp4",
		Filename:        "ad.mp4",
		AssetType:       "commercial",
		Scope:           "global",
		DurationSeconds: 30,
		Enabled:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	if err := store.ReplaceScheduledPrograms(context.Background(), channel.ID, []models.ScheduledProgram{
		{ChannelID: channel.ID, MediaID: asset.ID, MediaType: "asset", StartTime: start, EndTime: start.Add(30 * time.Second)},
	}); err != nil {
		t.Fatal(err)
	}
	programs, err := store.ScheduledPrograms(context.Background(), channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 || programs[0].MediaType != "asset" || programs[0].Asset.Filename != "ad.mp4" {
		t.Fatalf("programs = %#v", programs)
	}
}

func testMovieRecord(filePath string, title string, year int, runtime int, voteAverage float64, genres []string) models.MovieRecord {
	return models.MovieRecord{
		FilePath:   filePath,
		Filename:   "feature.mkv",
		LocalTitle: title,
		LocalYear:  year,
		Movie: models.Movie{
			TMDBID:         year,
			Title:          title,
			OriginalTitle:  title,
			Year:           year,
			ReleaseDate:    "1988-07-15",
			Overview:       "Overview",
			PosterPath:     "/poster.jpg",
			BackdropPath:   "/backdrop.jpg",
			RuntimeMinutes: runtime,
			Genres:         genres,
			VoteAverage:    voteAverage,
			Studio:         title + " Studio",
		},
		PosterLocalPath:   "data/artwork/poster.jpg",
		BackdropLocalPath: "data/artwork/backdrop.jpg",
	}
}

func TestStoreCollectionRoundTrip(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	saved, err := store.SaveCollection(context.Background(), models.Collection{
		Name:          "Favorites",
		Description:   "Things worth keeping together.",
		ArtworkMode:   "auto",
		ArtworkMedia:  []models.CollectionMedia{{MediaType: "movie", MediaID: 42}, {MediaType: "tv", MediaID: 7}},
		Rules:         []models.ChannelRule{{MediaType: "movie", Field: "genre", Operator: "is", Value: "Comedy"}},
		PinnedMedia:   []models.CollectionMedia{{MediaType: "movie", MediaID: 42}, {MediaType: "tv", MediaID: 7}},
		ExcludedMedia: []models.CollectionMedia{{MediaType: "movie", MediaID: 99}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == 0 || saved.ArtworkMode != "auto" || len(saved.ArtworkMedia) != 2 || len(saved.Rules) != 1 || len(saved.PinnedMedia) != 2 || len(saved.ExcludedMedia) != 1 {
		t.Fatalf("saved collection = %#v", saved)
	}
	if saved.ExcludedMedia[0].MediaType != "movie" || saved.ExcludedMedia[0].MediaID != 99 {
		t.Fatalf("excluded media = %#v", saved.ExcludedMedia)
	}

	collections, err := store.Collections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(collections) != 1 || collections[0].Name != "Favorites" {
		t.Fatalf("collections = %#v", collections)
	}
	if err := store.DeleteCollection(context.Background(), saved.ID); err != nil {
		t.Fatal(err)
	}
	collections, err = store.Collections(context.Background())
	if err != nil || len(collections) != 0 {
		t.Fatalf("collections after delete = %#v, err = %v", collections, err)
	}
}

func testTVShowRecord(path string, title string, year int) models.TVShowRecord {
	return models.TVShowRecord{
		Path:              path,
		LocalTitle:        title,
		LocalYear:         year,
		LocalEpisodeCount: 10,
		TVShow: models.TVShow{
			TMDBID:       year,
			Name:         title,
			OriginalName: title,
			Year:         year,
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
	}
}
