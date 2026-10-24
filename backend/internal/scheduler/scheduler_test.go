package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"retrotv/internal/database"
	"retrotv/internal/models"
)

func TestPlayableMoviesKeepsScannedPaths(t *testing.T) {
	dir := t.TempDir()
	existingPath := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(existingPath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}

	movies := playableMovies([]models.MovieRecord{
		{ID: 1, FilePath: filepath.Join(dir, "missing.mkv")},
		{ID: 2, FilePath: existingPath},
		{ID: 3, FilePath: dir},
		{ID: 4, FilePath: ""},
	})

	if len(movies) != 3 {
		t.Fatalf("playableMovies returned %d movies, want 3", len(movies))
	}
	if movies[0].ID != 1 || movies[1].ID != 2 || movies[2].ID != 3 {
		t.Fatalf("playableMovies kept IDs %#v, want 1, 2, 3", []int64{movies[0].ID, movies[1].ID, movies[2].ID})
	}
}

func TestProgrammingBlockShuffleIsStableAndChannelSpecific(t *testing.T) {
	pools := mediaPools{}
	for id := int64(1); id <= 8; id++ {
		pools.movies = append(pools.movies, models.MovieRecord{ID: id})
		pools.shows = append(pools.shows, models.TVShowRecord{ID: id})
	}
	block := models.ProgrammingBlock{ID: 42, PlaybackMode: "shuffle"}
	day := time.Date(2026, 7, 24, 5, 0, 0, 0, time.UTC)

	first := orderProgrammingBlockPools(73, block, day, pools)
	repeated := orderProgrammingBlockPools(73, block, day, pools)
	otherChannel := orderProgrammingBlockPools(74, block, day, pools)

	movieIDs := func(items []models.MovieRecord) string {
		result := ""
		for _, item := range items {
			result += fmt.Sprintf("%d,", item.ID)
		}
		return result
	}
	if movieIDs(first.movies) != movieIDs(repeated.movies) {
		t.Fatalf("shuffle is not reproducible: %s != %s", movieIDs(first.movies), movieIDs(repeated.movies))
	}
	if movieIDs(first.movies) == movieIDs(otherChannel.movies) {
		t.Fatalf("channels received the same shuffled order: %s", movieIDs(first.movies))
	}
	if movieIDs(pools.movies) != "1,2,3,4,5,6,7,8," {
		t.Fatalf("shuffle mutated the source pool: %s", movieIDs(pools.movies))
	}
}

func TestProgrammingBlockInOrderPreservesPoolOrder(t *testing.T) {
	pools := mediaPools{movies: []models.MovieRecord{{ID: 3}, {ID: 1}, {ID: 2}}}
	ordered := orderProgrammingBlockPools(73, models.ProgrammingBlock{ID: 42, PlaybackMode: "in_order"}, time.Now(), pools)
	for index, expected := range []int64{3, 1, 2} {
		if ordered.movies[index].ID != expected {
			t.Fatalf("movies[%d] = %d, want %d", index, ordered.movies[index].ID, expected)
		}
	}
}

func TestPlayableTVShowsUsesExistingEpisodePath(t *testing.T) {
	dir := t.TempDir()
	showDir := filepath.Join(dir, "Show")
	if err := os.MkdirAll(filepath.Join(showDir, "Season_01"), 0o755); err != nil {
		t.Fatal(err)
	}
	episodePath := filepath.Join(showDir, "Season_01", "S01E01.mkv")
	if err := os.WriteFile(episodePath, []byte("episode"), 0o644); err != nil {
		t.Fatal(err)
	}

	shows := playableTVShows([]models.TVShowRecord{
		{ID: 1, Path: filepath.Join(dir, "Missing"), EpisodePaths: []string{filepath.Join(dir, "missing.mkv")}},
		{ID: 2, Path: showDir, EpisodePaths: []string{episodePath}},
		{ID: 3, Path: filepath.Join(dir, "Empty"), EpisodePaths: nil},
	})

	if len(shows) != 2 {
		t.Fatalf("playableTVShows returned %d shows, want 2", len(shows))
	}
	if shows[0].ID != 1 || shows[1].ID != 2 {
		t.Fatalf("playableTVShows kept IDs %#v, want 1, 2", []int64{shows[0].ID, shows[1].ID})
	}
}

func TestFirstPlayableEpisodePathFallsBackToShowDirectory(t *testing.T) {
	dir := t.TempDir()
	showDir := filepath.Join(dir, "Show")
	if err := os.MkdirAll(filepath.Join(showDir, "Season_01"), 0o755); err != nil {
		t.Fatal(err)
	}
	episodePath := filepath.Join(showDir, "Season_01", "S01E01.mp4")
	if err := os.WriteFile(episodePath, []byte("episode"), 0o644); err != nil {
		t.Fatal(err)
	}

	path := firstPlayableEpisodePath(models.TVShowRecord{
		Path:         showDir,
		EpisodePaths: []string{filepath.Join(dir, "missing.mkv")},
	})

	if path != episodePath {
		t.Fatalf("firstPlayableEpisodePath = %q, want %q", path, episodePath)
	}
}

func TestMaintainChannelWindowAppendsFromPersistedMovieState(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	var movies []models.MovieRecord
	for index, title := range []string{"A Movie", "B Movie", "C Movie", "D Movie", "E Movie"} {
		path := filepath.Join(dir, title+".mkv")
		if err := os.WriteFile(path, []byte("movie"), 0o644); err != nil {
			t.Fatal(err)
		}
		movie, err := store.UpsertMovie(ctx, schedulerTestMovie(path, title, index+1))
		if err != nil {
			t.Fatal(err)
		}
		movies = append(movies, movie)
	}

	channel, err := store.SaveChannel(ctx, models.Channel{
		Name:        "Movies",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{MediaType: "movie", Field: "genre", Operator: "is", Value: "Action"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	if err := service.MaintainChannelWindow(ctx, channel.ID, now); err != nil {
		t.Fatal(err)
	}
	stateBeforeAppend, hasState, err := store.ChannelScheduleState(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasState {
		t.Fatal("missing movie schedule state")
	}
	expectedMovie := movies[stateBeforeAppend.NextMediaIndex]
	if err := service.MaintainChannelWindow(ctx, channel.ID, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	appendedStart := now.Add(scheduleWindow)
	var appended models.ScheduledProgram
	found := false
	for _, program := range programs {
		if program.StartTime.Equal(appendedStart) {
			appended = program
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no appended program starting at %s", appendedStart)
	}
	if appended.MediaID != expectedMovie.ID {
		t.Fatalf("first appended media ID = %d, want %d", appended.MediaID, expectedMovie.ID)
	}
}

func TestRegenerateChannelUsesStableInitialMovieOffset(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	var movies []models.MovieRecord
	for index, title := range []string{"16 Blocks", "2012", "Alien", "Blade", "Columbo"} {
		path := filepath.Join(dir, title+".mkv")
		if err := os.WriteFile(path, []byte("movie"), 0o644); err != nil {
			t.Fatal(err)
		}
		movie, err := store.UpsertMovie(ctx, schedulerTestMovie(path, title, index+1))
		if err != nil {
			t.Fatal(err)
		}
		movies = append(movies, movie)
	}

	channel, err := store.SaveChannel(ctx, models.Channel{
		Name:        "Movies",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{MediaType: "movie", Field: "genre", Operator: "is", Value: "Action"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	if err := service.RegenerateChannel(ctx, channel.ID); err != nil {
		t.Fatal(err)
	}

	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) == 0 {
		t.Fatal("no programs generated")
	}
	if programs[0].MediaID == movies[0].ID {
		t.Fatalf("regenerated schedule started at sorted index 0 movie %q", movies[0].Title)
	}
}

func TestTimelineProgrammingDoesNotUseChannelRulesAsFallback(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	generalPath := filepath.Join(dir, "General.mkv")
	if err := os.WriteFile(generalPath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	general, err := store.UpsertMovie(ctx, schedulerTestMovie(generalPath, "General", 1))
	if err != nil {
		t.Fatal(err)
	}
	blockPath := filepath.Join(dir, "Block.mkv")
	if err := os.WriteFile(blockPath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	blockMovie, err := store.UpsertMovie(ctx, schedulerTestMovie(blockPath, "Block", 2))
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Hour)
	start := now.Add(time.Hour).Format("15:04")
	end := now.Add(2 * time.Hour).Format("15:04")
	day := strings.ToLower(now.Add(time.Hour).Weekday().String()[:3])
	channel, err := store.SaveChannel(ctx, models.Channel{
		Name:        "Movies",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{MediaType: "movie", Field: "title", Operator: "is", Value: general.Title}},
		ProgrammingBlocks: []models.ProgrammingBlock{{
			Title:         "Block Hour",
			StartTime:     start,
			EndTime:       end,
			Days:          []string{day},
			ContentSource: "media",
			Enabled:       true,
			MediaItems: []models.ProgrammingBlockMedia{{
				MediaType: "movie",
				MediaID:   blockMovie.ID,
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	service := New(store)
	if err := service.RegenerateChannel(ctx, channel.ID); err != nil {
		t.Fatal(err)
	}
	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	var blockFound bool
	var generalFound bool
	for _, program := range programs {
		if program.MediaID == blockMovie.ID {
			blockFound = true
		}
		if program.MediaID == general.ID {
			generalFound = true
		}
	}
	if !blockFound {
		t.Fatalf("block movie was not scheduled: %#v", programs)
	}
	if generalFound {
		t.Fatalf("channel rules unexpectedly filled timeline gaps: %#v", programs)
	}
}

func TestProgrammingIntervalsUseLocalWallClockTime(t *testing.T) {
	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	blocks := []models.ProgrammingBlock{{
		Title:     "Overnight Movies",
		StartTime: "00:00",
		EndTime:   "09:00",
		Days:      []string{"thu"},
		Enabled:   true,
	}}
	windowStart := time.Date(2026, time.July, 16, 0, 0, 0, 0, time.UTC)
	windowEnd := windowStart.Add(48 * time.Hour)

	intervals := programmingIntervalsInLocation(blocks, windowStart, windowEnd, location)
	if len(intervals) != 1 {
		t.Fatalf("interval count = %d, want 1: %#v", len(intervals), intervals)
	}
	wantStart := time.Date(2026, time.July, 16, 5, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, time.July, 16, 14, 0, 0, 0, time.UTC)
	if !intervals[0].start.Equal(wantStart) || !intervals[0].end.Equal(wantEnd) {
		t.Fatalf("interval = %s to %s, want %s to %s", intervals[0].start, intervals[0].end, wantStart, wantEnd)
	}
}

func TestAllDayProgrammingFillsTimelineGaps(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	dir := t.TempDir()
	defaultPath := filepath.Join(dir, "Default.mkv")
	primePath := filepath.Join(dir, "Prime.mkv")
	if err := os.WriteFile(defaultPath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(primePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	defaultMovie, err := store.UpsertMovie(ctx, schedulerTestMovie(defaultPath, "Default", 101))
	if err != nil {
		t.Fatal(err)
	}
	primeMovie, err := store.UpsertMovie(ctx, schedulerTestMovie(primePath, "Prime", 102))
	if err != nil {
		t.Fatal(err)
	}
	defaultCollection, err := store.SaveCollection(ctx, models.Collection{Name: "Default", PinnedMedia: []models.CollectionMedia{{MediaType: "movie", MediaID: defaultMovie.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	primeCollection, err := store.SaveCollection(ctx, models.Collection{Name: "Prime", PinnedMedia: []models.CollectionMedia{{MediaType: "movie", MediaID: primeMovie.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Hour)
	timelineStart := now.Add(2 * time.Hour)
	timelineEnd := timelineStart.Add(2 * time.Hour)
	day := strings.ToLower(timelineStart.In(time.Local).Weekday().String()[:3])
	channel, err := store.SaveChannel(ctx, models.Channel{Name: "Blocks Only", ContentType: "mixed", ProgrammingBlocks: []models.ProgrammingBlock{
		{Title: "Default Programming", Placement: models.ProgrammingBlockPlacementAllDay, Days: []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}, ContentSource: "collection", CollectionID: defaultCollection.ID, Enabled: true},
		{Title: "Prime Time", Placement: models.ProgrammingBlockPlacementTimeline, StartTime: timelineStart.In(time.Local).Format("15:04"), EndTime: timelineEnd.In(time.Local).Format("15:04"), Days: []string{day}, ContentSource: "collection", CollectionID: primeCollection.ID, Enabled: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := New(store).RegenerateChannel(ctx, channel.ID); err != nil {
		t.Fatal(err)
	}
	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	var defaultBefore, primeDuring, defaultAfter bool
	for _, program := range programs {
		if program.MediaID == defaultMovie.ID && program.StartTime.Before(timelineStart) {
			defaultBefore = true
		}
		if program.MediaID == primeMovie.ID && !program.StartTime.Before(timelineStart) && program.StartTime.Before(timelineEnd) {
			primeDuring = true
		}
		if program.MediaID == defaultMovie.ID && !program.StartTime.Before(timelineEnd) {
			defaultAfter = true
		}
	}
	if !defaultBefore || !primeDuring || !defaultAfter {
		t.Fatalf("all-day/timeline precedence failed: before=%v during=%v after=%v programs=%#v", defaultBefore, primeDuring, defaultAfter, programs)
	}
}

func TestAllDayProgrammingDoesNotRunIntoTimelineBlock(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	dir := t.TempDir()
	longPath := filepath.Join(dir, "Long.mkv")
	shortPath := filepath.Join(dir, "Short.mkv")
	primePath := filepath.Join(dir, "Prime.mkv")
	for _, path := range []string{longPath, shortPath, primePath} {
		if err := os.WriteFile(path, []byte("movie"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	long := schedulerTestMovie(longPath, "Long", 201)
	long.RuntimeMinutes = 110
	short := schedulerTestMovie(shortPath, "Short", 202)
	short.RuntimeMinutes = 22
	prime := schedulerTestMovie(primePath, "Prime", 203)
	prime.RuntimeMinutes = 90
	longMovie, err := store.UpsertMovie(ctx, long)
	if err != nil {
		t.Fatal(err)
	}
	shortMovie, err := store.UpsertMovie(ctx, short)
	if err != nil {
		t.Fatal(err)
	}
	primeMovie, err := store.UpsertMovie(ctx, prime)
	if err != nil {
		t.Fatal(err)
	}
	regularCollection, err := store.SaveCollection(ctx, models.Collection{Name: "Regular", PinnedMedia: []models.CollectionMedia{
		{MediaType: "movie", MediaID: longMovie.ID},
		{MediaType: "movie", MediaID: shortMovie.ID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	primeCollection, err := store.SaveCollection(ctx, models.Collection{Name: "Prime", PinnedMedia: []models.CollectionMedia{{MediaType: "movie", MediaID: primeMovie.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	// Three hours ahead leaves room for the 110 minute movie to start in the
	// regular block and, without a deadline, run past the timeline start.
	now := time.Now().UTC().Truncate(time.Hour)
	timelineStart := now.Add(3 * time.Hour)
	timelineEnd := timelineStart.Add(2 * time.Hour)
	day := strings.ToLower(timelineStart.In(time.Local).Weekday().String()[:3])
	channel, err := store.SaveChannel(ctx, models.Channel{Name: "Fixed Start", ContentType: "movie", ProgrammingBlocks: []models.ProgrammingBlock{
		{Title: "Regular Programming", Placement: models.ProgrammingBlockPlacementAllDay, Days: []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}, ContentSource: "collection", CollectionID: regularCollection.ID, Enabled: true},
		{Title: "Movie Night", Placement: models.ProgrammingBlockPlacementTimeline, StartTime: timelineStart.In(time.Local).Format("15:04"), EndTime: timelineEnd.In(time.Local).Format("15:04"), Days: []string{day}, ContentSource: "collection", CollectionID: primeCollection.ID, Enabled: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := New(store).RegenerateChannel(ctx, channel.ID); err != nil {
		t.Fatal(err)
	}
	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}

	primeOnTime := false
	for _, program := range programs {
		if program.StartTime.Before(timelineStart) && program.EndTime.After(timelineStart) {
			t.Fatalf("program %d (%s-%s) runs into the %s timeline block", program.MediaID, program.StartTime, program.EndTime, timelineStart)
		}
		if program.MediaID == primeMovie.ID && program.StartTime.Equal(timelineStart) {
			primeOnTime = true
		}
	}
	if !primeOnTime {
		t.Fatalf("timeline block did not start at %s: %#v", timelineStart, programs)
	}
}

func TestMovieScheduleLeavesGapWhenNothingFitsBeforeDeadline(t *testing.T) {
	dir := t.TempDir()
	moviePath := filepath.Join(dir, "Long.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	movie := schedulerTestMovie(moviePath, "Long", 1)
	movie.RuntimeMinutes = 110
	assets := []models.ChannelAsset{{ID: 1, Filename: "promo.mp4", AssetType: "promo", DurationSeconds: 60, Enabled: true}}

	start := time.Date(2026, 7, 7, 17, 0, 0, 0, time.UTC)
	deadline := start.Add(time.Hour)
	programs, _, _, lastEnd := buildMovieScheduleWithin(1, []models.MovieRecord{movie}, start, deadline, deadline, 0, assets)
	if len(programs) == 0 {
		t.Fatal("want filler before the deadline")
	}
	for _, program := range programs {
		if program.MediaType != "asset" {
			t.Fatalf("program = %#v, want only filler since the movie cannot finish by the deadline", program)
		}
		if program.EndTime.After(deadline) {
			t.Fatalf("filler %#v runs past the deadline", program)
		}
	}
	if !lastEnd.Equal(deadline) {
		t.Fatalf("lastEnd = %s, want %s", lastEnd, deadline)
	}
}

func TestRegenerateAllRebuildsScheduleWithNewAssets(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	moviePath := filepath.Join(dir, "Short Movie.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	movie := schedulerTestMovie(moviePath, "Short Movie", 1)
	movie.RuntimeMinutes = 22
	if _, err := store.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	channel, err := store.SaveChannel(ctx, models.Channel{
		Name:        "Movies",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{MediaType: "movie", Field: "genre", Operator: "is", Value: "Action"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	if err := service.GenerateAll(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertChannelAsset(ctx, models.ChannelAsset{
		FilePath:        filepath.Join(dir, "commercial.mp4"),
		Filename:        "commercial.mp4",
		AssetType:       "commercial",
		Scope:           "global",
		DurationSeconds: 60,
		Enabled:         true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.RegenerateAll(ctx); err != nil {
		t.Fatal(err)
	}

	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, program := range programs {
		if program.MediaType == "asset" {
			return
		}
	}
	t.Fatalf("regenerated schedule has no asset rows; programs=%#v", programs[:min(len(programs), 6)])
}

func TestMaintainChannelWindowDoesNotRewriteFullFutureSchedule(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	for index, title := range []string{"A Movie", "B Movie"} {
		path := filepath.Join(dir, title+".mkv")
		if err := os.WriteFile(path, []byte("movie"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpsertMovie(ctx, schedulerTestMovie(path, title, index+1)); err != nil {
			t.Fatal(err)
		}
	}

	channel, err := store.SaveChannel(ctx, models.Channel{Name: "Movies", ContentType: "movie"})
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	if err := service.MaintainChannelWindow(ctx, channel.ID, now); err != nil {
		t.Fatal(err)
	}
	firstPrograms, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.MaintainChannelWindow(ctx, channel.ID, now); err != nil {
		t.Fatal(err)
	}
	secondPrograms, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPrograms) != len(firstPrograms) {
		t.Fatalf("program count after second maintenance = %d, want %d", len(secondPrograms), len(firstPrograms))
	}
	for index := range firstPrograms {
		if secondPrograms[index].ID != firstPrograms[index].ID {
			t.Fatalf("program at index %d was rewritten: got ID %d, want %d", index, secondPrograms[index].ID, firstPrograms[index].ID)
		}
	}
}

func TestPatchChannelSchedulePreservesProgramsOutsideChangedWindow(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	movies := make([]models.MovieRecord, 0, 4)
	for index, title := range []string{"Pool One", "Pool Two", "Pool Three", "Friday Special"} {
		path := filepath.Join(dir, title+".mkv")
		if err := os.WriteFile(path, []byte("movie"), 0o644); err != nil {
			t.Fatal(err)
		}
		movie := schedulerTestMovie(path, title, index+1)
		movie.RuntimeMinutes = 90
		saved, err := store.UpsertMovie(ctx, movie)
		if err != nil {
			t.Fatal(err)
		}
		movies = append(movies, saved)
	}
	collection, err := store.SaveCollection(ctx, models.Collection{
		Name: "Default Pool",
		PinnedMedia: []models.CollectionMedia{
			{MediaType: "movie", MediaID: movies[0].ID},
			{MediaType: "movie", MediaID: movies[1].ID},
			{MediaType: "movie", MediaID: movies[2].ID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	allDays := []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
	channel, err := store.SaveChannel(ctx, models.Channel{Name: "Stable FOX", ContentType: "movie", Airing: true, ProgrammingBlocks: []models.ProgrammingBlock{{
		Title: "Default Pool", Placement: models.ProgrammingBlockPlacementAllDay, Days: allDays,
		ContentSource: "collection", CollectionID: collection.ID, ScheduleMode: "weekly", PlaybackMode: "shuffle", Enabled: true,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	if err := service.RegenerateChannel(ctx, channel.ID); err != nil {
		t.Fatal(err)
	}
	before, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) == 0 {
		t.Fatal("initial schedule is empty")
	}

	now := time.Now().UTC().Truncate(time.Hour)
	protectedAfter := now.Add(4 * 24 * time.Hour)
	var protected models.ScheduledProgram
	for _, program := range before {
		if !program.StartTime.Before(protectedAfter) {
			protected = program
			break
		}
	}
	if protected.ID == 0 {
		t.Fatal("no later program available to protect")
	}

	eventDate := time.Now().In(time.Local).AddDate(0, 0, 1)
	eventDay := strings.ToLower(eventDate.Weekday().String()[:3])
	previousBlocks := channel.ProgrammingBlocks
	nextBlocks := append([]models.ProgrammingBlock{}, previousBlocks...)
	nextBlocks = append(nextBlocks, models.ProgrammingBlock{
		Title: "Friday Special", Placement: models.ProgrammingBlockPlacementTimeline,
		StartTime: "20:00", EndTime: "22:00", Days: []string{eventDay}, ScheduleMode: "once",
		ExactDate: eventDate.Format("2006-01-02"), PlaybackMode: "in_order", ContentSource: "media", Enabled: true,
		MediaItems: []models.ProgrammingBlockMedia{{MediaType: "movie", MediaID: movies[3].ID}},
	})
	channel.ProgrammingBlocks = nextBlocks
	if _, err := store.SaveChannel(ctx, channel); err != nil {
		t.Fatal(err)
	}
	if err := service.PatchChannelSchedule(ctx, channel.ID, previousBlocks, nextBlocks); err != nil {
		t.Fatal(err)
	}

	after, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, program := range after {
		if program.StartTime.Equal(protected.StartTime) && program.EndTime.Equal(protected.EndTime) {
			if program.MediaID != protected.MediaID || program.MediaType != protected.MediaType {
				t.Fatalf("protected program changed: before=%#v after=%#v", protected, program)
			}
			return
		}
	}
	t.Fatalf("protected program at %s-%s disappeared after localized patch", protected.StartTime, protected.EndTime)
}

func TestMaintainTVChannelWindowAppendsFromPersistedEpisodeState(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	var shows []models.TVShowRecord
	for _, title := range []string{"Alpha Show", "Beta Show"} {
		show, err := store.UpsertTVShow(ctx, schedulerTestTVShow(t, dir, title, 3))
		if err != nil {
			t.Fatal(err)
		}
		shows = append(shows, show)
	}

	channel, err := store.SaveChannel(ctx, models.Channel{
		Name:        "TV",
		ContentType: "tv",
		Rules: []models.ChannelRule{
			{MediaType: "tv", Field: "title", Operator: "is", Value: "Alpha Show"},
			{MediaType: "tv", Field: "title", Operator: "is", Value: "Beta Show"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := New(store)
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	if err := service.MaintainChannelWindow(ctx, channel.ID, now); err != nil {
		t.Fatal(err)
	}
	stateBeforeAppend, hasState, err := store.ChannelScheduleState(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !hasState {
		t.Fatal("missing TV schedule state")
	}
	expectedShow := shows[stateBeforeAppend.NextMediaIndex]
	expectedEpisodes := playableEpisodePaths(expectedShow)
	expectedEpisode := expectedEpisodes[stateBeforeAppend.EpisodeIndexes[expectedShow.ID]%len(expectedEpisodes)]

	if err := service.MaintainChannelWindow(ctx, channel.ID, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	appendedStart := now.Add(scheduleWindow)
	for _, program := range programs {
		if program.StartTime.Equal(appendedStart) {
			if program.MediaID != expectedShow.ID {
				t.Fatalf("first appended show ID = %d, want %d", program.MediaID, expectedShow.ID)
			}
			if program.EpisodePath != expectedEpisode {
				t.Fatalf("first appended episode = %q, want %q", program.EpisodePath, expectedEpisode)
			}
			return
		}
	}
	t.Fatalf("no appended TV program starting at %s", appendedStart)
}

func TestMovieScheduleKeepsMediaRuntimeWithinAlignedSlots(t *testing.T) {
	dir := t.TempDir()
	moviePath := filepath.Join(dir, "Odd Runtime.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	movie := schedulerTestMovie(moviePath, "Odd Runtime", 1)
	movie.RuntimeMinutes = 109

	start := time.Date(2026, 7, 7, 20, 0, 0, 0, time.UTC)
	programs := buildSchedule(1, []models.MovieRecord{movie}, start, 3*time.Hour)
	if len(programs) == 0 {
		t.Fatal("expected movie programs")
	}
	if !programs[0].StartTime.Equal(start) {
		t.Fatalf("movie start = %s, want %s", programs[0].StartTime, start)
	}
	wantEnd := start.Add(109 * time.Minute)
	if !programs[0].EndTime.Equal(wantEnd) {
		t.Fatalf("movie end = %s, want %s", programs[0].EndTime, wantEnd)
	}
}

func TestMovieScheduleFillsSlotGapWithAssets(t *testing.T) {
	dir := t.TempDir()
	moviePath := filepath.Join(dir, "Short Movie.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	movie := schedulerTestMovie(moviePath, "Short Movie", 1)
	movie.RuntimeMinutes = 22
	assets := []models.ChannelAsset{
		{ID: 1, Filename: "bumper.mp4", AssetType: "bumper", DurationSeconds: 15, Enabled: true},
		{ID: 2, Filename: "commercial.mp4", AssetType: "commercial", DurationSeconds: 30, Enabled: true},
		{ID: 3, Filename: "promo.mp4", AssetType: "promo", DurationSeconds: 60, Enabled: true},
	}

	start := time.Date(2026, 7, 7, 20, 0, 0, 0, time.UTC)
	programs, _, _, _ := buildMovieScheduleUntil(1, []models.MovieRecord{movie}, start, start.Add(30*time.Minute), 0, assets)
	if len(programs) < 2 {
		t.Fatalf("programs = %#v, want media plus asset fillers", programs)
	}
	if programs[0].MediaType != "" || !programs[0].EndTime.Equal(start.Add(22*time.Minute)) {
		t.Fatalf("first program = %#v", programs[0])
	}
	for _, program := range programs[1:] {
		if program.MediaType != "asset" {
			t.Fatalf("filler media type = %q, want asset", program.MediaType)
		}
		if program.EndTime.After(start.Add(30 * time.Minute)) {
			t.Fatalf("asset overruns slot: %#v", program)
		}
	}
}

func TestAssetsForChannelKeepsCommercialsGlobalAndBumpersChannelSpecific(t *testing.T) {
	channel := models.Channel{ID: 7, Name: "FOX Kids"}
	assets := []models.ChannelAsset{
		{ID: 1, AssetType: "commercial", Scope: "global"},
		{ID: 2, ChannelID: 7, AssetType: "bumper", Scope: "channel"},
		{ID: 3, ChannelID: 8, AssetType: "bumper", Scope: "channel"},
		{ID: 4, AssetType: "bumper", Scope: "global"},
	}

	filtered := assetsForChannel(channel, assets)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2: %#v", len(filtered), filtered)
	}
	if filtered[0].ID != 1 || filtered[1].ID != 2 {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestMovieScheduleAlignsOddStartToNextThirtyMinuteBoundary(t *testing.T) {
	dir := t.TempDir()
	moviePath := filepath.Join(dir, "Odd Start.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 7, 7, 20, 9, 0, 0, time.UTC)
	programs := buildSchedule(1, []models.MovieRecord{schedulerTestMovie(moviePath, "Odd Start", 1)}, start, 2*time.Hour)
	if len(programs) == 0 {
		t.Fatal("expected movie programs")
	}
	wantStart := time.Date(2026, 7, 7, 20, 30, 0, 0, time.UTC)
	if !programs[0].StartTime.Equal(wantStart) {
		t.Fatalf("movie start = %s, want %s", programs[0].StartTime, wantStart)
	}
}

func TestTVScheduleAlignsEpisodeMetadataRuntime(t *testing.T) {
	dir := t.TempDir()
	show := schedulerTestTVShow(t, dir, "Runtime Show", 1)
	show.EpisodeMetadata = []models.TVEpisode{
		{
			Path:           show.EpisodePaths[0],
			Season:         1,
			Episode:        1,
			Name:           "Runtime Episode",
			RuntimeMinutes: 27,
			MediaMetadata:  models.MediaMetadata{DurationSeconds: 1605},
		},
	}

	start := time.Date(2026, 7, 7, 13, 0, 0, 0, time.UTC)
	programs := buildTVSchedule(1, []models.TVShowRecord{show}, start, time.Hour)
	if len(programs) == 0 {
		t.Fatal("expected TV programs")
	}
	if got := programs[0].EndTime.Sub(programs[0].StartTime); got != 26*time.Minute+45*time.Second {
		t.Fatalf("TV media duration = %s, want 26m45s", got)
	}
}

func TestFillAssetGapTruncatesFinalAssetToAvoidDeadAir(t *testing.T) {
	start := time.Date(2026, 7, 16, 14, 22, 45, 0, time.UTC)
	end := time.Date(2026, 7, 16, 14, 30, 0, 0, time.UTC)
	assets := []models.ChannelAsset{
		{ID: 1, DurationSeconds: 163},
		{ID: 2, DurationSeconds: 30},
	}
	programs := fillAssetGap(75, start, end, assets, 1)
	if len(programs) == 0 {
		t.Fatal("expected filler programs")
	}
	if got := programs[len(programs)-1].EndTime; !got.Equal(end) {
		t.Fatalf("filler ends at %s, want %s", got, end)
	}
	for index := 1; index < len(programs); index++ {
		if !programs[index-1].EndTime.Equal(programs[index].StartTime) {
			t.Fatalf("gap between programs %d and %d", index-1, index)
		}
	}
}

func TestMixedScheduleAlignsEpisodeMetadataRuntime(t *testing.T) {
	dir := t.TempDir()
	moviePath := filepath.Join(dir, "Movie.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	show := schedulerTestTVShow(t, dir, "Runtime Show", 1)
	show.EpisodeMetadata = []models.TVEpisode{
		{
			Path:           show.EpisodePaths[0],
			Season:         1,
			Episode:        1,
			Name:           "Runtime Episode",
			RuntimeMinutes: 27,
		},
	}

	start := time.Date(2026, 7, 7, 13, 0, 0, 0, time.UTC)
	programs, _, _, _, _, _ := buildMixedScheduleUntil(
		1,
		[]models.MovieRecord{schedulerTestMovie(moviePath, "Movie", 1)},
		[]models.TVShowRecord{show},
		start,
		start.Add(2*time.Hour),
		0,
		0,
		nil,
		nil,
	)
	if len(programs) < 2 {
		t.Fatalf("len(programs) = %d, want at least 2", len(programs))
	}
	if programs[1].MediaType != "tv" {
		t.Fatalf("second media type = %q, want tv", programs[1].MediaType)
	}
	if got := programs[1].EndTime.Sub(programs[1].StartTime); got != 27*time.Minute {
		t.Fatalf("mixed TV media duration = %s, want 27m", got)
	}
}

func TestRegenerateMixedChannelSchedulesMoviesAndTV(t *testing.T) {
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "retrotv.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	moviePath := filepath.Join(dir, "Movie.mkv")
	if err := os.WriteFile(moviePath, []byte("movie"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertMovie(ctx, schedulerTestMovie(moviePath, "Movie", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertTVShow(ctx, schedulerTestTVShow(t, dir, "Show", 2)); err != nil {
		t.Fatal(err)
	}
	channel, err := store.SaveChannel(ctx, models.Channel{
		Name:        "Mixed",
		ContentType: "mixed",
		Rules: []models.ChannelRule{
			{MediaType: "movie", Field: "genre", Operator: "is", Value: "Action"},
			{MediaType: "tv", Field: "title", Operator: "is", Value: "Show"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	service := New(store)
	if err := service.RegenerateChannel(ctx, channel.ID); err != nil {
		t.Fatal(err)
	}

	programs, err := store.ScheduledPrograms(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) < 2 {
		t.Fatalf("len(programs) = %d, want at least 2", len(programs))
	}
	if programs[0].MediaType != "movie" || programs[1].MediaType != "tv" {
		t.Fatalf("media types = %q, %q; want movie, tv", programs[0].MediaType, programs[1].MediaType)
	}
	state, ok, err := store.ChannelScheduleState(ctx, channel.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("missing mixed schedule state")
	}
	if state.EpisodeIndexes == nil {
		t.Fatal("missing mixed episode indexes")
	}
}

func schedulerTestMovie(path string, title string, tmdbID int) models.MovieRecord {
	return models.MovieRecord{
		FilePath:   path,
		Filename:   filepath.Base(path),
		LocalTitle: title,
		LocalYear:  2000 + tmdbID,
		Movie: models.Movie{
			TMDBID:         tmdbID,
			Title:          title,
			OriginalTitle:  title,
			Year:           2000 + tmdbID,
			ReleaseDate:    "2000-01-01",
			Overview:       "Overview",
			RuntimeMinutes: 60,
			Genres:         []string{"Action"},
			VoteAverage:    7,
		},
	}
}

func schedulerTestTVShow(t *testing.T, root string, title string, episodes int) models.TVShowRecord {
	t.Helper()

	showDir := filepath.Join(root, title)
	seasonDir := filepath.Join(showDir, "Season_01")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatal(err)
	}
	episodePaths := make([]string, 0, episodes)
	for episode := 1; episode <= episodes; episode++ {
		path := filepath.Join(seasonDir, fmt.Sprintf("S01E%02d.mkv", episode))
		if err := os.WriteFile(path, []byte("episode"), 0o644); err != nil {
			t.Fatal(err)
		}
		episodePaths = append(episodePaths, path)
	}
	return models.TVShowRecord{
		Path:              showDir,
		LocalTitle:        title,
		LocalYear:         2000,
		LocalEpisodeCount: episodes,
		EpisodePaths:      episodePaths,
		TVShow: models.TVShow{
			TMDBID:       len(title),
			Name:         title,
			OriginalName: title,
			Year:         2000,
			Episodes:     episodes,
		},
	}
}
