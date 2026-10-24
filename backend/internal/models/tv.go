package models

import "time"

type TVShowFile struct {
	Path         string
	Title        string
	Year         int
	TMDBID       string
	EpisodeCount int
	EpisodePaths []string
	// EpisodeFingerprints is populated during the directory walk so later scan
	// phases do not need to stat every episode again.
	EpisodeFingerprints map[string]MediaMetadata
}

type TVShow struct {
	TMDBID          int      `json:"tmdb_id"`
	MetadataVersion int      `json:"metadata_version"`
	Name            string   `json:"name"`
	OriginalName    string   `json:"original_name"`
	Year            int      `json:"year"`
	FirstAirDate    string   `json:"first_air_date"`
	Overview        string   `json:"overview"`
	PosterPath      string   `json:"poster_path"`
	BackdropPath    string   `json:"backdrop_path"`
	Genres          []string `json:"genres"`
	Rating          string   `json:"rating"`
	Seasons         int      `json:"seasons"`
	Episodes        int      `json:"episodes"`
	Network         string   `json:"network"`
	Studio          string   `json:"studio"`
	Actors          []string `json:"actors"`
	Directors       []string `json:"directors"`
	Writers         []string `json:"writers"`
	Countries       []string `json:"countries"`
	Languages       []string `json:"languages"`
	SeriesStatus    string   `json:"series_status"`
	Tags            []string `json:"tags"`
}

type TVShowRecord struct {
	TVShow
	ID                int64       `json:"id"`
	Path              string      `json:"path"`
	LocalTitle        string      `json:"local_title"`
	LocalYear         int         `json:"local_year"`
	LocalEpisodeCount int         `json:"local_episode_count"`
	EpisodePaths      []string    `json:"episode_paths"`
	EpisodeMetadata   []TVEpisode `json:"episode_metadata"`
	PosterLocalPath   string      `json:"poster_local_path"`
	BackdropLocalPath string      `json:"backdrop_local_path"`
	LastSeenAt        time.Time   `json:"last_seen_at"`
	MissingSince      *time.Time  `json:"missing_since,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	// RestrictToEpisodePaths stops the scheduler from adding unscanned files
	// found in Path, for rule matches that narrowed EpisodePaths.
	RestrictToEpisodePaths bool `json:"-"`
}

type TVEpisode struct {
	Path               string        `json:"path"`
	Season             int           `json:"season"`
	Episode            int           `json:"episode"`
	Name               string        `json:"name"`
	Overview           string        `json:"overview"`
	AirDate            string        `json:"air_date"`
	StillPath          string        `json:"still_path"`
	ThumbnailLocalPath string        `json:"thumbnail_local_path"`
	RuntimeMinutes     int           `json:"runtime_minutes"`
	MediaMetadata      MediaMetadata `json:"media_metadata,omitempty"`
}
