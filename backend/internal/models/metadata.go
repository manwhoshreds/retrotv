package models

type MetadataMatch struct {
	TMDBID        int     `json:"tmdb_id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title,omitempty"`
	Year          int     `json:"year,omitempty"`
	ReleaseDate   string  `json:"release_date,omitempty"`
	Overview      string  `json:"overview,omitempty"`
	PosterPath    string  `json:"poster_path,omitempty"`
	BackdropPath  string  `json:"backdrop_path,omitempty"`
	VoteAverage   float64 `json:"vote_average,omitempty"`
}

type MetadataFacets struct {
	Titles         []string  `json:"titles"`
	Genres         []string  `json:"genres"`
	ReleaseYears   []int     `json:"release_years"`
	VoteAverages   []float64 `json:"vote_averages"`
	Runtimes       []int     `json:"runtimes"`
	Ratings        []string  `json:"ratings"`
	Studios        []string  `json:"studios"`
	Networks       []string  `json:"networks"`
	Actors         []string  `json:"actors"`
	Directors      []string  `json:"directors"`
	Writers        []string  `json:"writers"`
	Countries      []string  `json:"countries"`
	Languages      []string  `json:"languages"`
	SeriesStatuses []string  `json:"series_statuses"`
	Tags           []string  `json:"tags"`
	ReleaseDates   []string  `json:"release_dates"`
	FirstAirDates  []string  `json:"first_air_dates"`
	DateAddedDates []string  `json:"date_added_dates"`
}
