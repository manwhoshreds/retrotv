package models

import "time"

type MovieFile struct {
	Path          string
	Filename      string
	Title         string
	Year          int
	TMDBID        string
	MediaMetadata MediaMetadata
}

type Movie struct {
	TMDBID         int      `json:"tmdb_id"`
	Title          string   `json:"title"`
	OriginalTitle  string   `json:"original_title"`
	Year           int      `json:"year"`
	ReleaseDate    string   `json:"release_date"`
	Overview       string   `json:"overview"`
	PosterPath     string   `json:"poster_path"`
	BackdropPath   string   `json:"backdrop_path"`
	RuntimeMinutes int      `json:"runtime_minutes"`
	Genres         []string `json:"genres"`
	VoteAverage    float64  `json:"vote_average"`
	Rating         string   `json:"rating"`
	Studio         string   `json:"studio"`
	Actors         []string `json:"actors"`
	Directors      []string `json:"directors"`
	Writers        []string `json:"writers"`
	Countries      []string `json:"countries"`
	Languages      []string `json:"languages"`
	Tags           []string `json:"tags"`
}

type MovieRecord struct {
	Movie
	ID                int64         `json:"id"`
	FilePath          string        `json:"file_path"`
	Filename          string        `json:"filename"`
	LocalTitle        string        `json:"local_title"`
	LocalYear         int           `json:"local_year"`
	MediaMetadata     MediaMetadata `json:"media_metadata,omitempty"`
	PosterLocalPath   string        `json:"poster_local_path"`
	BackdropLocalPath string        `json:"backdrop_local_path"`
	LastSeenAt        time.Time     `json:"last_seen_at"`
	MissingSince      *time.Time    `json:"missing_since,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}
