package scheduler

import (
	"context"
	"hash/fnv"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"retrotv/internal/database"
	"retrotv/internal/mediapath"
	"retrotv/internal/models"
)

const scheduleWindow = 7 * 24 * time.Hour
const scheduleAlignment = 30 * time.Minute

// How far past a schedule window to look for fixed-time blocks that content
// near the end of the window must not run into.
const timelineDeadlineLookahead = 24 * time.Hour
const fallbackTVEpisodeRuntime = 30 * time.Minute

const (
	mixedMovieIndexStateKey int64 = -1
	mixedShowIndexStateKey  int64 = -2
)

type Service struct {
	store *database.Store
	queue chan scheduleJob
	done  chan struct{}
	once  sync.Once
}

type scheduleJob struct {
	channelID  int64
	regenerate bool
}

func New(store *database.Store) *Service {
	return &Service{
		store: store,
		queue: make(chan scheduleJob, 64),
		done:  make(chan struct{}),
	}
}

func (s *Service) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Service) Stop() {
	s.once.Do(func() {
		close(s.done)
	})
}

func (s *Service) QueueChannel(channelID int64) {
	s.queueJob(scheduleJob{channelID: channelID})
}

func (s *Service) QueueRegenerateChannel(channelID int64) {
	s.queueJob(scheduleJob{channelID: channelID, regenerate: true})
}

func (s *Service) queueJob(job scheduleJob) {
	select {
	case s.queue <- job:
	default:
		go func() {
			s.queue <- job
		}()
	}
}

func (s *Service) GenerateAll(ctx context.Context) error {
	channels, err := s.store.Channels(ctx)
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if !channel.Airing {
			if err := s.store.ReplaceScheduledPrograms(ctx, channel.ID, nil); err != nil {
				return err
			}
			continue
		}
		if err := s.GenerateChannel(ctx, channel.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RegenerateAll(ctx context.Context) error {
	channels, err := s.store.Channels(ctx)
	if err != nil {
		return err
	}
	for _, channel := range channels {
		if !channel.Airing {
			if err := s.store.ReplaceScheduledPrograms(ctx, channel.ID, nil); err != nil {
				return err
			}
			continue
		}
		if err := s.RegenerateChannel(ctx, channel.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GenerateChannel(ctx context.Context, channelID int64) error {
	return s.MaintainChannelWindow(ctx, channelID, time.Now().UTC())
}

func (s *Service) MaintainChannelWindow(ctx context.Context, channelID int64, now time.Time) error {
	channel, err := s.store.Channel(ctx, channelID)
	if err != nil {
		return err
	}
	if !channel.Airing {
		return s.store.ReplaceScheduledPrograms(ctx, channel.ID, nil)
	}
	assets, err := s.store.EnabledChannelAssets(ctx)
	if err != nil {
		return err
	}
	assets = assetsForChannel(channel, assets)
	now = now.UTC().Truncate(time.Hour)
	targetEnd := now.Add(scheduleWindow)
	if err := s.store.PruneScheduledProgramsBefore(ctx, channel.ID, now); err != nil {
		return err
	}
	latest, hasLatest, err := s.store.LatestScheduledProgram(ctx, channel.ID)
	if err != nil {
		return err
	}
	start := now
	if hasLatest {
		if !latest.EndTime.Before(targetEnd) {
			return nil
		}
		if latest.EndTime.After(start) {
			start = latest.EndTime
		}
	}

	{
		state, hasState, err := s.store.ChannelScheduleState(ctx, channel.ID)
		if err != nil {
			return err
		}
		programs, scheduleState, err := s.buildBlockAwareSchedule(ctx, channel, start, targetEnd, state, hasState, latest, hasLatest, assets)
		if err != nil {
			return err
		}
		if err := s.store.AppendScheduledPrograms(ctx, channel.ID, programs); err != nil {
			return err
		}
		if len(programs) == 0 {
			return nil
		}
		return s.store.SaveChannelScheduleState(ctx, scheduleState)
	}

	if channel.ContentType == "tv" {
		shows, err := s.store.MatchingTVShows(ctx, channel.Rules)
		if err != nil {
			return err
		}
		shows = playableTVShows(shows)
		state, hasState, err := s.store.ChannelScheduleState(ctx, channel.ID)
		if err != nil {
			return err
		}
		startIndex := nextTVShowIndex(channel, shows, state, hasState, latest, hasLatest)
		programs, nextIndex, lastMediaID, lastEnd, episodeIndexes := buildTVScheduleUntil(channel.ID, shows, start, targetEnd, startIndex, state.EpisodeIndexes, assets)
		if err := s.store.AppendScheduledPrograms(ctx, channel.ID, programs); err != nil {
			return err
		}
		if len(programs) == 0 {
			return nil
		}
		return s.store.SaveChannelScheduleState(ctx, models.ChannelScheduleState{
			ChannelID:           channel.ID,
			NextMediaIndex:      nextIndex,
			LastMediaID:         lastMediaID,
			EpisodeIndexes:      episodeIndexes,
			LastScheduleEndTime: lastEnd,
		})
	}
	if channel.ContentType == "mixed" {
		movies, err := s.store.MatchingMovies(ctx, channel.Rules)
		if err != nil {
			return err
		}
		shows, err := s.store.MatchingTVShows(ctx, channel.Rules)
		if err != nil {
			return err
		}
		movies = playableMovies(movies)
		shows = playableTVShows(shows)
		state, hasState, err := s.store.ChannelScheduleState(ctx, channel.ID)
		if err != nil {
			return err
		}
		movieIndex, showIndex := mixedStartIndexes(channel, movies, shows, state, hasState)
		programs, nextMovieIndex, nextShowIndex, lastMediaID, lastEnd, episodeIndexes := buildMixedScheduleUntil(channel.ID, movies, shows, start, targetEnd, movieIndex, showIndex, state.EpisodeIndexes, assets)
		if err := s.store.AppendScheduledPrograms(ctx, channel.ID, programs); err != nil {
			return err
		}
		if len(programs) == 0 {
			return nil
		}
		episodeIndexes[mixedMovieIndexStateKey] = nextMovieIndex
		episodeIndexes[mixedShowIndexStateKey] = nextShowIndex
		return s.store.SaveChannelScheduleState(ctx, models.ChannelScheduleState{
			ChannelID:           channel.ID,
			NextMediaIndex:      nextMovieIndex,
			LastMediaID:         lastMediaID,
			EpisodeIndexes:      episodeIndexes,
			LastScheduleEndTime: lastEnd,
		})
	}
	movies, err := s.store.MatchingMovies(ctx, channel.Rules)
	if err != nil {
		return err
	}
	movies = playableMovies(movies)

	state, hasState, err := s.store.ChannelScheduleState(ctx, channel.ID)
	if err != nil {
		return err
	}
	startIndex := nextMovieIndex(channel, movies, state, hasState, latest, hasLatest)
	programs, nextIndex, lastMediaID, lastEnd := buildMovieScheduleUntil(channel.ID, movies, start, targetEnd, startIndex, assets)
	if err := s.store.AppendScheduledPrograms(ctx, channel.ID, programs); err != nil {
		return err
	}
	if len(programs) == 0 {
		return nil
	}
	return s.store.SaveChannelScheduleState(ctx, models.ChannelScheduleState{
		ChannelID:           channel.ID,
		NextMediaIndex:      nextIndex,
		LastMediaID:         lastMediaID,
		LastScheduleEndTime: lastEnd,
	})
}

func (s *Service) RegenerateChannel(ctx context.Context, channelID int64) error {
	channel, err := s.store.Channel(ctx, channelID)
	if err != nil {
		return err
	}
	if !channel.Airing {
		return s.store.ReplaceScheduledPrograms(ctx, channel.ID, nil)
	}
	start := time.Now().UTC().Truncate(time.Hour)
	end := start.Add(scheduleWindow)
	assets, err := s.store.EnabledChannelAssets(ctx)
	if err != nil {
		return err
	}
	assets = assetsForChannel(channel, assets)
	{
		programs, scheduleState, err := s.buildBlockAwareSchedule(ctx, channel, start, end, models.ChannelScheduleState{}, false, models.ScheduledProgram{}, false, assets)
		if err != nil {
			return err
		}
		if err := s.store.ReplaceScheduledPrograms(ctx, channel.ID, programs); err != nil {
			return err
		}
		if len(programs) == 0 {
			return nil
		}
		return s.store.SaveChannelScheduleState(ctx, scheduleState)
	}
	if channel.ContentType == "tv" {
		shows, err := s.store.MatchingTVShows(ctx, channel.Rules)
		if err != nil {
			return err
		}
		shows = playableTVShows(shows)
		programs, nextIndex, lastMediaID, lastEnd, episodeIndexes := buildTVScheduleUntil(channel.ID, shows, start, end, initialScheduleOffset(channel, len(shows)), nil, assets)
		if err := s.store.ReplaceScheduledPrograms(ctx, channel.ID, programs); err != nil {
			return err
		}
		if len(programs) == 0 {
			return nil
		}
		return s.store.SaveChannelScheduleState(ctx, models.ChannelScheduleState{
			ChannelID:           channel.ID,
			NextMediaIndex:      nextIndex,
			LastMediaID:         lastMediaID,
			EpisodeIndexes:      episodeIndexes,
			LastScheduleEndTime: lastEnd,
		})
	}
	if channel.ContentType == "mixed" {
		movies, err := s.store.MatchingMovies(ctx, channel.Rules)
		if err != nil {
			return err
		}
		shows, err := s.store.MatchingTVShows(ctx, channel.Rules)
		if err != nil {
			return err
		}
		movies = playableMovies(movies)
		shows = playableTVShows(shows)
		movieIndex := initialScheduleOffset(channel, len(movies))
		showIndex := initialScheduleOffset(channel, len(shows))
		programs, nextMovieIndex, nextShowIndex, lastMediaID, lastEnd, episodeIndexes := buildMixedScheduleUntil(channel.ID, movies, shows, start, end, movieIndex, showIndex, nil, assets)
		if err := s.store.ReplaceScheduledPrograms(ctx, channel.ID, programs); err != nil {
			return err
		}
		if len(programs) == 0 {
			return nil
		}
		episodeIndexes[mixedMovieIndexStateKey] = nextMovieIndex
		episodeIndexes[mixedShowIndexStateKey] = nextShowIndex
		return s.store.SaveChannelScheduleState(ctx, models.ChannelScheduleState{
			ChannelID:           channel.ID,
			NextMediaIndex:      nextMovieIndex,
			LastMediaID:         lastMediaID,
			EpisodeIndexes:      episodeIndexes,
			LastScheduleEndTime: lastEnd,
		})
	}
	movies, err := s.store.MatchingMovies(ctx, channel.Rules)
	if err != nil {
		return err
	}
	movies = playableMovies(movies)
	programs, nextIndex, lastMediaID, lastEnd := buildMovieScheduleUntil(channel.ID, movies, start, end, initialScheduleOffset(channel, len(movies)), assets)
	if err := s.store.ReplaceScheduledPrograms(ctx, channel.ID, programs); err != nil {
		return err
	}
	if len(programs) == 0 {
		return nil
	}
	return s.store.SaveChannelScheduleState(ctx, models.ChannelScheduleState{
		ChannelID:           channel.ID,
		NextMediaIndex:      nextIndex,
		LastMediaID:         lastMediaID,
		LastScheduleEndTime: lastEnd,
	})
}

func playableMovies(movies []models.MovieRecord) []models.MovieRecord {
	playable := []models.MovieRecord{}
	for _, movie := range movies {
		if strings.TrimSpace(movie.FilePath) == "" {
			continue
		}
		if resolvedPath := mediapath.Resolve(movie.FilePath); regularFileExists(resolvedPath) {
			movie.FilePath = resolvedPath
		}
		playable = append(playable, movie)
	}
	return playable
}

func assetsForChannel(channel models.Channel, assets []models.ChannelAsset) []models.ChannelAsset {
	filtered := []models.ChannelAsset{}
	for _, asset := range assets {
		if asset.AssetType == "commercial" && asset.Scope == "global" {
			filtered = append(filtered, asset)
			continue
		}
		if asset.Scope == "channel" && asset.ChannelID == channel.ID {
			filtered = append(filtered, asset)
		}
	}
	return filtered
}

func playableTVShows(shows []models.TVShowRecord) []models.TVShowRecord {
	playable := []models.TVShowRecord{}
	for _, show := range shows {
		if firstPlayableEpisodePath(show) != "" {
			playable = append(playable, show)
		}
	}
	return playable
}

type mediaPools struct {
	movies []models.MovieRecord
	shows  []models.TVShowRecord
}

type segmentScheduleState struct {
	movieIndex     int
	showIndex      int
	episodeIndexes map[int64]int
	lastMediaID    int64
	lastEnd        time.Time
}

type programmingInterval struct {
	block models.ProgrammingBlock
	start time.Time
	end   time.Time
}

// PatchChannelSchedule preserves the published schedule outside windows affected
// by programming edits. It intentionally leaves the existing continuity state
// alone because unchanged future log entries remain authoritative.
func (s *Service) PatchChannelSchedule(ctx context.Context, channelID int64, previousBlocks, nextBlocks []models.ProgrammingBlock) error {
	existing, err := s.store.ScheduledPrograms(ctx, channelID)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		return s.RegenerateChannel(ctx, channelID)
	}

	changed := changedProgrammingBlocks(previousBlocks, nextBlocks)
	if len(changed) == 0 {
		return nil
	}
	start := time.Now().UTC().Truncate(time.Hour)
	end := start.Add(scheduleWindow)
	windows := programmingPatchWindows(changed, start, end, existing)
	if len(windows) == 0 {
		return nil
	}

	channel, err := s.store.Channel(ctx, channelID)
	if err != nil {
		return err
	}
	assets, err := s.store.EnabledChannelAssets(ctx)
	if err != nil {
		return err
	}
	assets = assetsForChannel(channel, assets)
	candidate, _, err := s.buildBlockAwareSchedule(ctx, channel, start, end, models.ChannelScheduleState{}, false, models.ScheduledProgram{}, false, assets)
	if err != nil {
		return err
	}

	merged := make([]models.ScheduledProgram, 0, len(existing)+len(candidate))
	for _, program := range existing {
		if !programOverlapsPatchWindows(program, windows) {
			merged = append(merged, program)
		}
	}
	for _, program := range candidate {
		for _, window := range windows {
			if !program.EndTime.After(window.start) || !program.StartTime.Before(window.end) {
				continue
			}
			patched := program
			if patched.StartTime.Before(window.start) {
				patched.StartTime = window.start
			}
			if patched.EndTime.After(window.end) {
				patched.EndTime = window.end
			}
			if patched.EndTime.After(patched.StartTime) {
				merged = append(merged, patched)
			}
			break
		}
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].StartTime.Before(merged[j].StartTime) })
	return s.store.ReplaceScheduledPrograms(ctx, channelID, merged)
}

func changedProgrammingBlocks(previous, next []models.ProgrammingBlock) []models.ProgrammingBlock {
	previousByID := map[int64]models.ProgrammingBlock{}
	for _, block := range previous {
		if block.ID > 0 {
			previousByID[block.ID] = block
		}
	}
	changed := []models.ProgrammingBlock{}
	seen := map[int64]bool{}
	for _, block := range next {
		old, exists := previousByID[block.ID]
		if block.ID <= 0 || !exists {
			changed = append(changed, block)
			continue
		}
		seen[block.ID] = true
		if !reflect.DeepEqual(canonicalProgrammingBlock(old), canonicalProgrammingBlock(block)) {
			changed = append(changed, old, block)
		}
	}
	for id, block := range previousByID {
		if !seen[id] {
			changed = append(changed, block)
		}
	}
	return changed
}

func canonicalProgrammingBlock(block models.ProgrammingBlock) models.ProgrammingBlock {
	block.ID, block.ChannelID = 0, 0
	block.Name = block.Title
	for index := range block.MediaItems {
		block.MediaItems[index].ID = 0
		block.MediaItems[index].BlockID = 0
	}
	return block
}

func programmingPatchWindows(blocks []models.ProgrammingBlock, start, end time.Time, existing []models.ScheduledProgram) []programmingInterval {
	windows := programmingIntervals(blocks, start, end)
	for index := range windows {
		for _, program := range existing {
			if program.EndTime.After(windows[index].start) && program.StartTime.Before(windows[index].end) {
				if program.StartTime.Before(windows[index].start) {
					windows[index].start = program.StartTime
				}
				if program.EndTime.After(windows[index].end) {
					windows[index].end = program.EndTime
				}
			}
		}
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].start.Before(windows[j].start) })
	merged := []programmingInterval{}
	for _, window := range windows {
		if len(merged) == 0 || window.start.After(merged[len(merged)-1].end) {
			merged = append(merged, programmingInterval{start: window.start, end: window.end})
			continue
		}
		if window.end.After(merged[len(merged)-1].end) {
			merged[len(merged)-1].end = window.end
		}
	}
	return merged
}

func programOverlapsPatchWindows(program models.ScheduledProgram, windows []programmingInterval) bool {
	for _, window := range windows {
		if program.EndTime.After(window.start) && program.StartTime.Before(window.end) {
			return true
		}
	}
	return false
}

func (s *Service) buildBlockAwareSchedule(ctx context.Context, channel models.Channel, start time.Time, end time.Time, state models.ChannelScheduleState, hasState bool, latest models.ScheduledProgram, hasLatest bool, assets []models.ChannelAsset) ([]models.ScheduledProgram, models.ChannelScheduleState, error) {
	general, err := s.generalMediaPools(ctx, channel)
	if err != nil {
		return nil, models.ChannelScheduleState{}, err
	}
	segmentState := segmentScheduleState{
		movieIndex:     nextMovieIndex(channel, general.movies, state, hasState, latest, hasLatest),
		showIndex:      nextTVShowIndex(channel, general.shows, state, hasState, latest, hasLatest),
		episodeIndexes: cloneEpisodeIndexes(state.EpisodeIndexes),
	}
	if hasState {
		switch channel.ContentType {
		case "tv":
			segmentState.showIndex = state.NextMediaIndex
		case "mixed":
			segmentState.movieIndex = state.EpisodeIndexes[mixedMovieIndexStateKey]
			segmentState.showIndex = state.EpisodeIndexes[mixedShowIndexStateKey]
		default:
			segmentState.movieIndex = state.NextMediaIndex
		}
	} else {
		segmentState.movieIndex = initialScheduleOffset(channel, len(general.movies))
		segmentState.showIndex = initialScheduleOffset(channel, len(general.shows))
	}

	intervals := programmingIntervals(channel.ProgrammingBlocks, start, end)
	// Look past the window too, so content near its end cannot run into a
	// fixed-time block that starts just after it.
	upcoming := programmingIntervals(channel.ProgrammingBlocks, start, end.Add(timelineDeadlineLookahead))
	programs := []models.ScheduledProgram{}
	current := start
	for _, interval := range intervals {
		deadline := nextTimelineStart(upcoming, interval)
		if interval.end.After(end) {
			interval.end = end
		}
		if interval.start.Before(start) {
			interval.start = start
		}
		if !interval.start.After(current) && interval.end.After(current) {
			interval.start = current
		}
		if interval.start.After(current) {
			current = interval.start
		}
		if interval.end.After(current) {
			blockPools, err := s.programmingBlockPools(ctx, interval.block, general)
			if err != nil {
				return nil, models.ChannelScheduleState{}, err
			}
			blockPools = orderProgrammingBlockPools(channel.ID, interval.block, interval.start, blockPools)
			blockPrograms, nextState := buildScheduleSegment(channel.ID, segmentContentType(channel.ContentType, blockPools), blockPools, current, interval.end, deadline, segmentState, assets)
			programs = append(programs, blockPrograms...)
			segmentState = nextState
			current = interval.end
		}
	}
	return programs, models.ChannelScheduleState{
		ChannelID:           channel.ID,
		NextMediaIndex:      segmentState.movieIndex,
		LastMediaID:         segmentState.lastMediaID,
		EpisodeIndexes:      segmentState.episodeIndexes,
		LastScheduleEndTime: segmentState.lastEnd,
	}, nil
}

// nextTimelineStart returns when the next timeline (fixed-time) block after
// interval begins, or the zero time if none starts within the lookahead.
// Timeline blocks must start on time, so nothing scheduled before one may run
// past its start; all-day programming has no such guarantee.
func nextTimelineStart(upcoming []programmingInterval, interval programmingInterval) time.Time {
	for _, candidate := range upcoming {
		if candidate.block.Placement == models.ProgrammingBlockPlacementAllDay {
			continue
		}
		if candidate.start.After(interval.start) {
			return candidate.start
		}
	}
	return time.Time{}
}

func orderProgrammingBlockPools(channelID int64, block models.ProgrammingBlock, intervalStart time.Time, pools mediaPools) mediaPools {
	if block.PlaybackMode != "shuffle" {
		return pools
	}
	ordered := mediaPools{
		movies: append([]models.MovieRecord(nil), pools.movies...),
		shows:  append([]models.TVShowRecord(nil), pools.shows...),
	}
	// Seed by channel, block, and local broadcast day. This makes a published
	// schedule reproducible while ensuring independent channels and days do not
	// all walk the same database ordering.
	broadcastDay := intervalStart.In(time.Local)
	broadcastDay = time.Date(broadcastDay.Year(), broadcastDay.Month(), broadcastDay.Day(), 0, 0, 0, 0, time.Local)
	movieRNG := rand.New(rand.NewSource(scheduleSeed(channelID, block.ID*2+1, broadcastDay)))
	showRNG := rand.New(rand.NewSource(scheduleSeed(channelID, block.ID*2+2, broadcastDay)))
	movieRNG.Shuffle(len(ordered.movies), func(i, j int) {
		ordered.movies[i], ordered.movies[j] = ordered.movies[j], ordered.movies[i]
	})
	showRNG.Shuffle(len(ordered.shows), func(i, j int) {
		ordered.shows[i], ordered.shows[j] = ordered.shows[j], ordered.shows[i]
	})
	return ordered
}

func (s *Service) generalMediaPools(ctx context.Context, channel models.Channel) (mediaPools, error) {
	// Legacy rules are only available to the synthetic all-day migration block.
	// New and edited channels always have empty channel rules.
	movies, err := s.store.MatchingMovies(ctx, channel.Rules)
	if err != nil {
		return mediaPools{}, err
	}
	shows, err := s.store.MatchingTVShows(ctx, channel.Rules)
	if err != nil {
		return mediaPools{}, err
	}
	return mediaPools{movies: playableMovies(movies), shows: playableTVShows(shows)}, nil
}

func (s *Service) programmingBlockPools(ctx context.Context, block models.ProgrammingBlock, fallback mediaPools) (mediaPools, error) {
	if block.CollectionID > 0 {
		collection, err := s.store.Collection(ctx, block.CollectionID)
		if err != nil {
			return mediaPools{}, err
		}
		movies, err := s.store.MatchingMovies(ctx, collection.Rules)
		if err != nil {
			return mediaPools{}, err
		}
		shows, err := s.store.MatchingTVShows(ctx, collection.Rules)
		if err != nil {
			return mediaPools{}, err
		}
		for _, pinned := range collection.PinnedMedia {
			switch pinned.MediaType {
			case "movie":
				movie, err := s.store.MovieByID(ctx, pinned.MediaID)
				if err == nil {
					movies = append(movies, movie)
				}
			case "tv":
				show, err := s.store.TVShowByID(ctx, pinned.MediaID)
				if err == nil {
					shows = append(shows, show)
				}
			}
		}
		excludedMovies := make(map[int64]struct{})
		excludedShows := make(map[int64]struct{})
		for _, excluded := range collection.ExcludedMedia {
			switch excluded.MediaType {
			case "movie":
				excludedMovies[excluded.MediaID] = struct{}{}
			case "tv":
				excludedShows[excluded.MediaID] = struct{}{}
			}
		}
		includedMovies := movies[:0]
		for _, movie := range movies {
			if _, excluded := excludedMovies[movie.ID]; !excluded {
				includedMovies = append(includedMovies, movie)
			}
		}
		includedShows := shows[:0]
		for _, show := range shows {
			if _, excluded := excludedShows[show.ID]; !excluded {
				includedShows = append(includedShows, show)
			}
		}
		return mediaPools{movies: playableMovies(uniqueMovies(includedMovies)), shows: playableTVShows(uniqueShows(includedShows))}, nil
	}
	if len(block.MediaItems) == 0 {
		return fallback, nil
	}
	pools := mediaPools{}
	for _, item := range block.MediaItems {
		switch item.MediaType {
		case "movie":
			movie, err := s.store.MovieByID(ctx, item.MediaID)
			if err != nil {
				return mediaPools{}, err
			}
			pools.movies = append(pools.movies, movie)
		case "tv":
			show, err := s.store.TVShowByID(ctx, item.MediaID)
			if err != nil {
				return mediaPools{}, err
			}
			pools.shows = append(pools.shows, show)
		}
	}
	pools.movies = playableMovies(pools.movies)
	pools.shows = playableTVShows(pools.shows)
	return pools, nil
}

func uniqueMovies(items []models.MovieRecord) []models.MovieRecord {
	seen := map[int64]bool{}
	result := make([]models.MovieRecord, 0, len(items))
	for _, item := range items {
		if !seen[item.ID] {
			seen[item.ID] = true
			result = append(result, item)
		}
	}
	return result
}

func uniqueShows(items []models.TVShowRecord) []models.TVShowRecord {
	seen := map[int64]bool{}
	result := make([]models.TVShowRecord, 0, len(items))
	for _, item := range items {
		if !seen[item.ID] {
			seen[item.ID] = true
			result = append(result, item)
		}
	}
	return result
}

func buildScheduleSegment(channelID int64, contentType string, pools mediaPools, start time.Time, end time.Time, deadline time.Time, state segmentScheduleState, assets []models.ChannelAsset) ([]models.ScheduledProgram, segmentScheduleState) {
	if !start.Before(end) {
		return []models.ScheduledProgram{}, state
	}
	switch contentType {
	case "tv":
		programs, nextShowIndex, lastMediaID, lastEnd, episodeIndexes := buildTVScheduleWithin(channelID, pools.shows, start, end, deadline, state.showIndex, state.episodeIndexes, assets)
		state.showIndex = nextShowIndex
		state.lastMediaID = lastMediaID
		state.lastEnd = lastEnd
		state.episodeIndexes = episodeIndexes
		return programs, state
	case "mixed":
		programs, nextMovieIndex, nextShowIndex, lastMediaID, lastEnd, episodeIndexes := buildMixedScheduleWithin(channelID, pools.movies, pools.shows, start, end, deadline, state.movieIndex, state.showIndex, state.episodeIndexes, assets)
		state.movieIndex = nextMovieIndex
		state.showIndex = nextShowIndex
		state.lastMediaID = lastMediaID
		state.lastEnd = lastEnd
		state.episodeIndexes = episodeIndexes
		return programs, state
	default:
		programs, nextMovieIndex, lastMediaID, lastEnd := buildMovieScheduleWithin(channelID, pools.movies, start, end, deadline, state.movieIndex, assets)
		state.movieIndex = nextMovieIndex
		state.lastMediaID = lastMediaID
		state.lastEnd = lastEnd
		return programs, state
	}
}

func segmentContentType(fallback string, pools mediaPools) string {
	if len(pools.movies) > 0 && len(pools.shows) > 0 {
		return "mixed"
	}
	if len(pools.shows) > 0 {
		return "tv"
	}
	if len(pools.movies) > 0 {
		return "movie"
	}
	return fallback
}

func programmingIntervals(blocks []models.ProgrammingBlock, start time.Time, end time.Time) []programmingInterval {
	return programmingIntervalsInLocation(blocks, start, end, time.Local)
}

func programmingIntervalsInLocation(blocks []models.ProgrammingBlock, start time.Time, end time.Time, location *time.Location) []programmingInterval {
	candidates := []programmingInterval{}
	localStart := start.In(location)
	localEnd := end.In(location)
	for _, block := range blocks {
		if !block.Enabled {
			continue
		}
		var startClock, endClock time.Time
		if block.Placement != models.ProgrammingBlockPlacementAllDay {
			var err error
			startClock, err = time.Parse("15:04", block.StartTime)
			if err != nil {
				continue
			}
			endClock, err = time.Parse("15:04", block.EndTime)
			if err != nil {
				continue
			}
		}
		firstDay := time.Date(localStart.Year(), localStart.Month(), localStart.Day()-1, 0, 0, 0, 0, location)
		lastDay := time.Date(localEnd.Year(), localEnd.Month(), localEnd.Day()+1, 0, 0, 0, 0, location)
		for day := firstDay; day.Before(lastDay); day = day.AddDate(0, 0, 1) {
			if !blockRunsOnDay(block, day) {
				continue
			}
			intervalStart := day
			intervalEnd := day.AddDate(0, 0, 1)
			if block.Placement != models.ProgrammingBlockPlacementAllDay {
				intervalStart = time.Date(day.Year(), day.Month(), day.Day(), startClock.Hour(), startClock.Minute(), 0, 0, location)
				intervalEnd = time.Date(day.Year(), day.Month(), day.Day(), endClock.Hour(), endClock.Minute(), 0, 0, location)
				if !intervalEnd.After(intervalStart) {
					intervalEnd = intervalEnd.AddDate(0, 0, 1)
				}
			}
			if intervalEnd.After(start) && intervalStart.Before(end) {
				candidates = append(candidates, programmingInterval{block: block, start: intervalStart.UTC(), end: intervalEnd.UTC()})
			}
		}
	}
	boundaries := []time.Time{start, end}
	for _, candidate := range candidates {
		if candidate.start.After(start) && candidate.start.Before(end) {
			boundaries = append(boundaries, candidate.start)
		}
		if candidate.end.After(start) && candidate.end.Before(end) {
			boundaries = append(boundaries, candidate.end)
		}
	}
	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].Before(boundaries[j]) })
	intervals := []programmingInterval{}
	for index := 0; index+1 < len(boundaries); index++ {
		segmentStart, segmentEnd := boundaries[index], boundaries[index+1]
		if !segmentStart.Before(segmentEnd) {
			continue
		}
		active := []programmingInterval{}
		for _, candidate := range candidates {
			if !candidate.start.After(segmentStart) && candidate.end.After(segmentStart) {
				active = append(active, candidate)
			}
		}
		if len(active) == 0 {
			continue
		}
		sort.SliceStable(active, func(i, j int) bool {
			iTimeline := active[i].block.Placement != models.ProgrammingBlockPlacementAllDay
			jTimeline := active[j].block.Placement != models.ProgrammingBlockPlacementAllDay
			if iTimeline != jTimeline {
				return iTimeline
			}
			return active[i].block.Priority > active[j].block.Priority
		})
		chosen := active[0]
		if len(intervals) > 0 && intervals[len(intervals)-1].block.ID == chosen.block.ID && intervals[len(intervals)-1].end.Equal(segmentStart) {
			intervals[len(intervals)-1].end = segmentEnd
		} else {
			intervals = append(intervals, programmingInterval{block: chosen.block, start: segmentStart, end: segmentEnd})
		}
	}
	return intervals
}

func nonOverlappingIntervals(intervals []programmingInterval) []programmingInterval {
	filtered := []programmingInterval{}
	var cursor time.Time
	for _, interval := range intervals {
		if interval.end.IsZero() || !interval.end.After(interval.start) {
			continue
		}
		if !cursor.IsZero() && interval.start.Before(cursor) {
			if !interval.end.After(cursor) {
				continue
			}
			interval.start = cursor
		}
		filtered = append(filtered, interval)
		cursor = interval.end
	}
	return filtered
}

func blockRunsOnDay(block models.ProgrammingBlock, day time.Time) bool {
	date := day.Format("2006-01-02")
	for _, excluded := range block.ExcludedDates {
		if excluded == date {
			return false
		}
	}
	if block.ScheduleMode == "once" {
		return block.ExactDate == date
	}
	weekday := strings.ToLower(day.Weekday().String()[:3])
	for _, configured := range block.Days {
		if strings.EqualFold(configured, weekday) {
			return true
		}
	}
	return false
}

func firstPlayableEpisodePath(show models.TVShowRecord) string {
	paths := playableEpisodePaths(show)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func playableEpisodePaths(show models.TVShowRecord) []string {
	paths := []string{}
	missingScannedPaths := []string{}
	seen := map[string]struct{}{}
	for _, path := range show.EpisodePaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if resolvedPath := mediapath.Resolve(path); regularFileExists(resolvedPath) {
			path = resolvedPath
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
			continue
		}
		missingScannedPaths = append(missingScannedPaths, path)
	}

	if !show.RestrictToEpisodePaths {
		_ = filepath.WalkDir(show.Path, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if !videoFile(path) || !regularFileExists(path) {
				return nil
			}
			if _, ok := seen[path]; ok {
				return nil
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
			return nil
		})
	}
	for _, path := range missingScannedPaths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func regularFileExists(path string) bool {
	return mediapath.RegularFileExists(path)
}

func videoFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mkv", ".avi", ".mov":
		return true
	default:
		return false
	}
}

func buildTVSchedule(channelID int64, shows []models.TVShowRecord, start time.Time, duration time.Duration) []models.ScheduledProgram {
	programs, _, _, _, _ := buildTVScheduleUntil(channelID, shows, start, start.Add(duration), 0, nil, nil)
	return programs
}

func buildTVScheduleUntil(channelID int64, shows []models.TVShowRecord, start time.Time, end time.Time, startIndex int, startEpisodeIndexes map[int64]int, assets []models.ChannelAsset) ([]models.ScheduledProgram, int, int64, time.Time, map[int64]int) {
	return buildTVScheduleWithin(channelID, shows, start, end, time.Time{}, startIndex, startEpisodeIndexes, assets)
}

// buildTVScheduleWithin fills start..end with episodes. When deadline is set,
// no episode may run past it: a show whose next episode is too long is passed
// over for one that fits, and time nothing fits is left to channel filler.
func buildTVScheduleWithin(channelID int64, shows []models.TVShowRecord, start time.Time, end time.Time, deadline time.Time, startIndex int, startEpisodeIndexes map[int64]int, assets []models.ChannelAsset) ([]models.ScheduledProgram, int, int64, time.Time, map[int64]int) {
	if len(shows) == 0 {
		return []models.ScheduledProgram{}, 0, 0, time.Time{}, map[int64]int{}
	}

	programs := []models.ScheduledProgram{}
	current := alignTimeUp(start, scheduleAlignment)
	showIndex := normalizeIndex(startIndex, len(shows))
	episodeIndexes := cloneEpisodeIndexes(startEpisodeIndexes)
	var lastMediaID int64
	var lastEnd time.Time
	for current.Before(end) {
		choice, nextInLine, ok := episodeThatFits(shows, showIndex, episodeIndexes, current, deadline)
		if nextInLine < 0 {
			break
		}
		if !ok {
			programs = append(programs, fillAssetGap(channelID, current, end, assets, scheduleSeed(channelID, 0, current))...)
			lastEnd = end
			break
		}
		show := shows[choice.showIndex]
		episodeIndexes[show.ID] = (choice.episodeIndex + 1) % choice.episodeCount
		showIndex = rotationAfter(choice.showIndex, nextInLine, len(shows))
		slotEnd := slotEndBefore(current, choice.runtime, deadline)
		mediaEnd := current.Add(choice.runtime)
		if mediaEnd.After(slotEnd) {
			mediaEnd = slotEnd
		}
		programs = append(programs, models.ScheduledProgram{
			ChannelID:   channelID,
			MediaID:     show.ID,
			MediaType:   "tv",
			EpisodePath: choice.path,
			StartTime:   current,
			EndTime:     mediaEnd,
		})
		programs = append(programs, fillAssetGap(channelID, mediaEnd, slotEnd, assets, scheduleSeed(channelID, show.ID, current))...)
		lastMediaID = show.ID
		lastEnd = slotEnd
		current = slotEnd
	}
	return programs, showIndex, lastMediaID, lastEnd, episodeIndexes
}

func buildMixedScheduleUntil(channelID int64, movies []models.MovieRecord, shows []models.TVShowRecord, start time.Time, end time.Time, startMovieIndex int, startShowIndex int, startEpisodeIndexes map[int64]int, assets []models.ChannelAsset) ([]models.ScheduledProgram, int, int, int64, time.Time, map[int64]int) {
	return buildMixedScheduleWithin(channelID, movies, shows, start, end, time.Time{}, startMovieIndex, startShowIndex, startEpisodeIndexes, assets)
}

// buildMixedScheduleWithin alternates movies and episodes. When deadline is
// set and the type due next does not fit before it, the other type is tried,
// then channel filler.
func buildMixedScheduleWithin(channelID int64, movies []models.MovieRecord, shows []models.TVShowRecord, start time.Time, end time.Time, deadline time.Time, startMovieIndex int, startShowIndex int, startEpisodeIndexes map[int64]int, assets []models.ChannelAsset) ([]models.ScheduledProgram, int, int, int64, time.Time, map[int64]int) {
	if len(movies) == 0 {
		programs, nextShowIndex, lastMediaID, lastEnd, episodeIndexes := buildTVScheduleWithin(channelID, shows, start, end, deadline, startShowIndex, startEpisodeIndexes, assets)
		return programs, 0, nextShowIndex, lastMediaID, lastEnd, episodeIndexes
	}
	if len(shows) == 0 {
		programs, nextMovieIndex, lastMediaID, lastEnd := buildMovieScheduleWithin(channelID, movies, start, end, deadline, startMovieIndex, assets)
		return programs, nextMovieIndex, 0, lastMediaID, lastEnd, cloneEpisodeIndexes(startEpisodeIndexes)
	}

	programs := []models.ScheduledProgram{}
	current := alignTimeUp(start, scheduleAlignment)
	movieIndex := normalizeIndex(startMovieIndex, len(movies))
	showIndex := normalizeIndex(startShowIndex, len(shows))
	episodeIndexes := cloneEpisodeIndexes(startEpisodeIndexes)
	var lastMediaID int64
	var lastEnd time.Time
	useMovie := true
	for current.Before(end) {
		moviePick, movieFits := movieThatFits(movies, movieIndex, current, deadline)
		episode, nextShowInLine, episodeFits := episodeThatFits(shows, showIndex, episodeIndexes, current, deadline)
		playMovie := movieFits && (useMovie || !episodeFits)
		if !playMovie && !episodeFits {
			programs = append(programs, fillAssetGap(channelID, current, end, assets, scheduleSeed(channelID, 0, current))...)
			lastEnd = end
			break
		}

		if playMovie {
			movie := movies[moviePick]
			runtime := movieRuntime(movie)
			slotEnd := slotEndBefore(current, runtime, deadline)
			mediaEnd := current.Add(runtime)
			if mediaEnd.After(slotEnd) {
				mediaEnd = slotEnd
			}
			programs = append(programs, models.ScheduledProgram{
				ChannelID: channelID,
				MediaID:   movie.ID,
				MediaType: "movie",
				StartTime: current,
				EndTime:   mediaEnd,
			})
			programs = append(programs, fillAssetGap(channelID, mediaEnd, slotEnd, assets, scheduleSeed(channelID, movie.ID, current))...)
			lastMediaID = movie.ID
			lastEnd = slotEnd
			movieIndex = rotationAfter(moviePick, movieIndex, len(movies))
			current = slotEnd
			useMovie = false
			continue
		}

		show := shows[episode.showIndex]
		episodeIndexes[show.ID] = (episode.episodeIndex + 1) % episode.episodeCount
		showIndex = rotationAfter(episode.showIndex, nextShowInLine, len(shows))
		slotEnd := slotEndBefore(current, episode.runtime, deadline)
		mediaEnd := current.Add(episode.runtime)
		if mediaEnd.After(slotEnd) {
			mediaEnd = slotEnd
		}
		programs = append(programs, models.ScheduledProgram{
			ChannelID:   channelID,
			MediaID:     show.ID,
			MediaType:   "tv",
			EpisodePath: episode.path,
			StartTime:   current,
			EndTime:     mediaEnd,
		})
		programs = append(programs, fillAssetGap(channelID, mediaEnd, slotEnd, assets, scheduleSeed(channelID, show.ID, current))...)
		lastMediaID = show.ID
		lastEnd = slotEnd
		current = slotEnd
		useMovie = true
	}
	return programs, movieIndex, showIndex, lastMediaID, lastEnd, episodeIndexes
}

func tvEpisodeRuntime(show models.TVShowRecord, episodePath string) time.Duration {
	for _, episode := range show.EpisodeMetadata {
		if sameMediaPath(episode.Path, episodePath) {
			if episode.MediaMetadata.DurationSeconds > 0 {
				return time.Duration(int(episode.MediaMetadata.DurationSeconds)) * time.Second
			}
			if episode.RuntimeMinutes <= 0 {
				continue
			}
			return time.Duration(episode.RuntimeMinutes) * time.Minute
		}
	}
	return fallbackTVEpisodeRuntime
}

func movieRuntime(movie models.MovieRecord) time.Duration {
	if movie.MediaMetadata.DurationSeconds > 0 {
		return time.Duration(int(movie.MediaMetadata.DurationSeconds)) * time.Second
	}
	if movie.RuntimeMinutes > 0 {
		return time.Duration(movie.RuntimeMinutes) * time.Minute
	}
	return 120 * time.Minute
}

// fitsBefore reports whether media starting at start ends by deadline. A zero
// deadline means no fixed-time block is coming up, so anything fits.
func fitsBefore(start time.Time, runtime time.Duration, deadline time.Time) bool {
	return deadline.IsZero() || !start.Add(runtime).After(deadline)
}

// slotEndBefore is alignedSlotEnd, but never past deadline, so the filler
// after a programme stops where the fixed-time block begins.
func slotEndBefore(start time.Time, runtime time.Duration, deadline time.Time) time.Time {
	slotEnd := alignedSlotEnd(start, runtime)
	if !deadline.IsZero() && slotEnd.After(deadline) {
		return deadline
	}
	return slotEnd
}

// rotationAfter advances a rotation past the item that aired. When a later
// item aired in place of one that was too long, the skipped item stays next
// in line so it airs at the next opportunity.
func rotationAfter(aired int, nextInLine int, length int) int {
	if aired != nextInLine {
		return nextInLine
	}
	return (aired + 1) % length
}

// movieThatFits returns the movie next in rotation if it ends by deadline,
// otherwise the first later one in the rotation that does.
func movieThatFits(movies []models.MovieRecord, index int, start time.Time, deadline time.Time) (int, bool) {
	for offset := 0; offset < len(movies); offset++ {
		candidate := (index + offset) % len(movies)
		if fitsBefore(start, movieRuntime(movies[candidate]), deadline) {
			return candidate, true
		}
	}
	return 0, false
}

type episodeChoice struct {
	showIndex    int
	episodeIndex int
	episodeCount int
	path         string
	runtime      time.Duration
}

// episodeThatFits walks the show rotation from showIndex and returns the first
// show whose next episode ends by deadline. nextInLine is the first show with
// playable episodes (the one due to air), or -1 if no show has any.
func episodeThatFits(shows []models.TVShowRecord, showIndex int, episodeIndexes map[int64]int, start time.Time, deadline time.Time) (choice episodeChoice, nextInLine int, ok bool) {
	nextInLine = -1
	for offset := 0; offset < len(shows); offset++ {
		candidate := (showIndex + offset) % len(shows)
		show := shows[candidate]
		episodes := playableEpisodePaths(show)
		if len(episodes) == 0 {
			continue
		}
		if nextInLine < 0 {
			nextInLine = candidate
		}
		episodeIndex := normalizeIndex(episodeIndexes[show.ID], len(episodes))
		runtime := tvEpisodeRuntime(show, episodes[episodeIndex])
		if fitsBefore(start, runtime, deadline) {
			return episodeChoice{
				showIndex:    candidate,
				episodeIndex: episodeIndex,
				episodeCount: len(episodes),
				path:         episodes[episodeIndex],
				runtime:      runtime,
			}, nextInLine, true
		}
	}
	return episodeChoice{}, nextInLine, false
}

func alignedSlotEnd(start time.Time, runtime time.Duration) time.Time {
	if runtime <= 0 {
		runtime = scheduleAlignment
	}
	return alignTimeUp(start.Add(runtime), scheduleAlignment)
}

func alignTimeUp(value time.Time, interval time.Duration) time.Time {
	if interval <= 0 {
		return value
	}
	truncated := value.Truncate(interval)
	if truncated.Equal(value) {
		return value
	}
	return truncated.Add(interval)
}

func fillAssetGap(channelID int64, start time.Time, end time.Time, assets []models.ChannelAsset, seed int64) []models.ScheduledProgram {
	if len(assets) == 0 || !start.Before(end) {
		return nil
	}
	rng := rand.New(rand.NewSource(seed))
	programs := []models.ScheduledProgram{}
	current := start
	var previousID int64
	for current.Before(end) {
		remaining := end.Sub(current)
		candidates := fittingAssets(assets, remaining, previousID)
		if len(candidates) == 0 && previousID != 0 {
			candidates = fittingAssets(assets, remaining, 0)
		}
		if len(candidates) == 0 {
			// End the slot with a truncated asset rather than leaving dead air when
			// no asset duration exactly fits the final few seconds.
			asset, ok := shortestAsset(assets)
			if ok {
				programs = append(programs, models.ScheduledProgram{
					ChannelID: channelID,
					MediaID:   asset.ID,
					MediaType: "asset",
					StartTime: current,
					EndTime:   end,
					Asset:     asset,
				})
			}
			break
		}
		asset := candidates[rng.Intn(len(candidates))]
		if remaining <= 90*time.Second {
			asset = bestFitAsset(candidates, remaining)
		}
		duration := assetDuration(asset)
		next := current.Add(duration)
		if next.After(end) {
			break
		}
		programs = append(programs, models.ScheduledProgram{
			ChannelID: channelID,
			MediaID:   asset.ID,
			MediaType: "asset",
			StartTime: current,
			EndTime:   next,
			Asset:     asset,
		})
		current = next
		previousID = asset.ID
	}
	return programs
}

func shortestAsset(assets []models.ChannelAsset) (models.ChannelAsset, bool) {
	var shortest models.ChannelAsset
	for _, asset := range assets {
		if assetDuration(asset) <= 0 {
			continue
		}
		if shortest.ID == 0 || assetDuration(asset) < assetDuration(shortest) {
			shortest = asset
		}
	}
	return shortest, shortest.ID != 0
}

func fittingAssets(assets []models.ChannelAsset, remaining time.Duration, previousID int64) []models.ChannelAsset {
	candidates := []models.ChannelAsset{}
	for _, asset := range assets {
		duration := assetDuration(asset)
		if duration <= 0 || duration > remaining {
			continue
		}
		if previousID != 0 && asset.ID == previousID && len(assets) > 1 {
			continue
		}
		candidates = append(candidates, asset)
	}
	return candidates
}

func bestFitAsset(assets []models.ChannelAsset, remaining time.Duration) models.ChannelAsset {
	best := assets[0]
	bestRemainder := remaining - assetDuration(best)
	for _, asset := range assets[1:] {
		remainder := remaining - assetDuration(asset)
		if remainder >= 0 && remainder < bestRemainder {
			best = asset
			bestRemainder = remainder
		}
	}
	return best
}

func assetDuration(asset models.ChannelAsset) time.Duration {
	if asset.DurationSeconds <= 0 {
		return 0
	}
	return time.Duration(asset.DurationSeconds) * time.Second
}

func scheduleSeed(channelID int64, mediaID int64, start time.Time) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(strconv.FormatInt(channelID, 10)))
	_, _ = hash.Write([]byte(":"))
	_, _ = hash.Write([]byte(strconv.FormatInt(mediaID, 10)))
	_, _ = hash.Write([]byte(":"))
	_, _ = hash.Write([]byte(start.UTC().Format(time.RFC3339)))
	return int64(hash.Sum64())
}

func sameMediaPath(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	return mediapath.Resolve(left) == mediapath.Resolve(right)
}

func (s *Service) run(ctx context.Context) {
	if err := s.GenerateAll(ctx); err != nil {
		log.Printf("scheduler initial generation: %v", err)
	}

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case job := <-s.queue:
			var err error
			if job.regenerate {
				err = s.RegenerateChannel(ctx, job.channelID)
			} else {
				err = s.GenerateChannel(ctx, job.channelID)
			}
			if err != nil {
				log.Printf("scheduler channel %d: %v", job.channelID, err)
			}
		case <-ticker.C:
			if err := s.GenerateAll(ctx); err != nil {
				log.Printf("scheduler periodic generation: %v", err)
			}
		}
	}
}

func buildSchedule(channelID int64, movies []models.MovieRecord, start time.Time, duration time.Duration) []models.ScheduledProgram {
	programs, _, _, _ := buildMovieScheduleUntil(channelID, movies, start, start.Add(duration), 0, nil)
	return programs
}

func buildMovieScheduleUntil(channelID int64, movies []models.MovieRecord, start time.Time, end time.Time, startIndex int, assets []models.ChannelAsset) ([]models.ScheduledProgram, int, int64, time.Time) {
	return buildMovieScheduleWithin(channelID, movies, start, end, time.Time{}, startIndex, assets)
}

// buildMovieScheduleWithin fills start..end with movies. When deadline is set,
// no movie may run past it: a movie that is too long is passed over for one
// that fits, and time nothing fits is left to channel filler.
func buildMovieScheduleWithin(channelID int64, movies []models.MovieRecord, start time.Time, end time.Time, deadline time.Time, startIndex int, assets []models.ChannelAsset) ([]models.ScheduledProgram, int, int64, time.Time) {
	if len(movies) == 0 {
		return []models.ScheduledProgram{}, 0, 0, time.Time{}
	}

	programs := []models.ScheduledProgram{}
	current := alignTimeUp(start, scheduleAlignment)
	index := normalizeIndex(startIndex, len(movies))
	var lastMediaID int64
	var lastEnd time.Time
	for current.Before(end) {
		pick, ok := movieThatFits(movies, index, current, deadline)
		if !ok {
			programs = append(programs, fillAssetGap(channelID, current, end, assets, scheduleSeed(channelID, 0, current))...)
			lastEnd = end
			break
		}
		movie := movies[pick]
		runtime := movieRuntime(movie)
		slotEnd := slotEndBefore(current, runtime, deadline)
		mediaEnd := current.Add(runtime)
		if mediaEnd.After(slotEnd) {
			mediaEnd = slotEnd
		}
		programs = append(programs, models.ScheduledProgram{
			ChannelID: channelID,
			MediaID:   movie.ID,
			StartTime: current,
			EndTime:   mediaEnd,
		})
		programs = append(programs, fillAssetGap(channelID, mediaEnd, slotEnd, assets, scheduleSeed(channelID, movie.ID, current))...)
		lastMediaID = movie.ID
		lastEnd = slotEnd
		index = rotationAfter(pick, index, len(movies))
		current = slotEnd
	}
	return programs, index, lastMediaID, lastEnd
}

func nextMovieIndex(channel models.Channel, movies []models.MovieRecord, state models.ChannelScheduleState, hasState bool, latest models.ScheduledProgram, hasLatest bool) int {
	if len(movies) == 0 {
		return 0
	}
	if hasState && hasLatest && latest.EndTime.After(state.LastScheduleEndTime) {
		if index := movieIndexAfterID(movies, latest.MediaID); index >= 0 {
			return index
		}
	}
	if hasState {
		if index := movieIndexAfterID(movies, state.LastMediaID); index >= 0 {
			return index
		}
		return normalizeIndex(state.NextMediaIndex, len(movies))
	}
	if hasLatest {
		if index := movieIndexAfterID(movies, latest.MediaID); index >= 0 {
			return index
		}
	}
	return initialScheduleOffset(channel, len(movies))
}

func nextTVShowIndex(channel models.Channel, shows []models.TVShowRecord, state models.ChannelScheduleState, hasState bool, latest models.ScheduledProgram, hasLatest bool) int {
	if len(shows) == 0 {
		return 0
	}
	if hasState && hasLatest && latest.EndTime.After(state.LastScheduleEndTime) {
		if index := tvShowIndexAfterID(shows, latest.MediaID); index >= 0 {
			return index
		}
	}
	if hasState {
		if index := tvShowIndexAfterID(shows, state.LastMediaID); index >= 0 {
			return index
		}
		return normalizeIndex(state.NextMediaIndex, len(shows))
	}
	if hasLatest {
		if index := tvShowIndexAfterID(shows, latest.MediaID); index >= 0 {
			return index
		}
	}
	return initialScheduleOffset(channel, len(shows))
}

func mixedStartIndexes(channel models.Channel, movies []models.MovieRecord, shows []models.TVShowRecord, state models.ChannelScheduleState, hasState bool) (int, int) {
	if !hasState {
		return initialScheduleOffset(channel, len(movies)), initialScheduleOffset(channel, len(shows))
	}
	return normalizeIndex(state.EpisodeIndexes[mixedMovieIndexStateKey], len(movies)), normalizeIndex(state.EpisodeIndexes[mixedShowIndexStateKey], len(shows))
}

func movieIndexAfterID(movies []models.MovieRecord, mediaID int64) int {
	if mediaID == 0 {
		return -1
	}
	for index, movie := range movies {
		if movie.ID == mediaID {
			return (index + 1) % len(movies)
		}
	}
	return -1
}

func tvShowIndexAfterID(shows []models.TVShowRecord, mediaID int64) int {
	if mediaID == 0 {
		return -1
	}
	for index, show := range shows {
		if show.ID == mediaID {
			return (index + 1) % len(shows)
		}
	}
	return -1
}

func initialScheduleOffset(channel models.Channel, length int) int {
	if length <= 1 {
		return 0
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(channel.Name))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(time.Unix(channel.ID, 0).UTC().Format(time.RFC3339Nano)))
	index := int(hash.Sum64() % uint64(length))
	if index == 0 {
		return 1
	}
	return index
}

func cloneEpisodeIndexes(indexes map[int64]int) map[int64]int {
	clone := map[int64]int{}
	for showID, index := range indexes {
		clone[showID] = index
	}
	return clone
}

func normalizeIndex(index int, length int) int {
	if length == 0 {
		return 0
	}
	index %= length
	if index < 0 {
		index += length
	}
	return index
}
