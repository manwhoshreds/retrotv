package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"retrotv/internal/models"
)

const tmdbBaseURL = "https://api.themoviedb.org/3"
const tmdbImageBaseURL = "https://image.tmdb.org/t/p"

type MetadataDownloader struct {
	Token        string
	baseURL      string
	imageBaseURL string
	client       *http.Client
}

type tmdbSearchResponse struct {
	Results []tmdbSearchMovie `json:"results"`
}

type tmdbTVSearchResponse struct {
	Results []tmdbSearchTVShow `json:"results"`
}

type tmdbSearchMovie struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	ReleaseDate   string  `json:"release_date"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
}

type tmdbSearchTVShow struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
	FirstAirDate string `json:"first_air_date"`
}

type tmdbMovieDetails struct {
	ID                  int               `json:"id"`
	Title               string            `json:"title"`
	OriginalTitle       string            `json:"original_title"`
	ReleaseDate         string            `json:"release_date"`
	Overview            string            `json:"overview"`
	PosterPath          string            `json:"poster_path"`
	BackdropPath        string            `json:"backdrop_path"`
	Runtime             int               `json:"runtime"`
	Genres              []tmdbGenre       `json:"genres"`
	VoteAverage         float64           `json:"vote_average"`
	ReleaseDates        tmdbReleaseDates  `json:"release_dates"`
	Companies           []tmdbCompany     `json:"production_companies"`
	ProductionCountries []tmdbCountry     `json:"production_countries"`
	OriginalLanguage    string            `json:"original_language"`
	SpokenLanguages     []tmdbLanguage    `json:"spoken_languages"`
	Credits             tmdbCredits       `json:"credits"`
	Keywords            tmdbMovieKeywords `json:"keywords"`
}

type tmdbTVDetails struct {
	ID                  int                `json:"id"`
	Name                string             `json:"name"`
	OriginalName        string             `json:"original_name"`
	FirstAirDate        string             `json:"first_air_date"`
	Overview            string             `json:"overview"`
	PosterPath          string             `json:"poster_path"`
	BackdropPath        string             `json:"backdrop_path"`
	Genres              []tmdbGenre        `json:"genres"`
	VoteAverage         float64            `json:"vote_average"`
	NumberSeasons       int                `json:"number_of_seasons"`
	NumberEpisodes      int                `json:"number_of_episodes"`
	Networks            []tmdbNetwork      `json:"networks"`
	ProductionCompanies []tmdbCompany      `json:"production_companies"`
	CreatedBy           []tmdbPerson       `json:"created_by"`
	OriginCountry       []string           `json:"origin_country"`
	OriginalLanguage    string             `json:"original_language"`
	SpokenLanguages     []tmdbLanguage     `json:"spoken_languages"`
	Status              string             `json:"status"`
	ContentRatings      tmdbContentRatings `json:"content_ratings"`
	Credits             tmdbCredits        `json:"credits"`
	Keywords            tmdbKeywords       `json:"keywords"`
}

type tmdbTVEpisodeDetails struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
	AirDate       string `json:"air_date"`
	StillPath     string `json:"still_path"`
	Runtime       int    `json:"runtime"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
}

type tmdbNetwork struct {
	Name string `json:"name"`
}

type tmdbCompany struct {
	Name string `json:"name"`
}

type tmdbCountry struct {
	ISO31661 string `json:"iso_3166_1"`
	Name     string `json:"name"`
}

type tmdbLanguage struct {
	EnglishName string `json:"english_name"`
	Name        string `json:"name"`
	ISO6391     string `json:"iso_639_1"`
}

type tmdbPerson struct {
	Name       string `json:"name"`
	Character  string `json:"character"`
	Job        string `json:"job"`
	Department string `json:"department"`
}

type tmdbCredits struct {
	Cast []tmdbPerson `json:"cast"`
	Crew []tmdbPerson `json:"crew"`
}

type tmdbKeywords struct {
	Results []tmdbKeyword `json:"results"`
}

type tmdbMovieKeywords struct {
	Keywords []tmdbKeyword `json:"keywords"`
}

type tmdbKeyword struct {
	Name string `json:"name"`
}

type tmdbContentRatings struct {
	Results []tmdbContentRating `json:"results"`
}

type tmdbContentRating struct {
	ISO31661 string `json:"iso_3166_1"`
	Rating   string `json:"rating"`
}

type tmdbGenre struct {
	Name string `json:"name"`
}

type tmdbReleaseDates struct {
	Results []tmdbReleaseDateCountry `json:"results"`
}

type tmdbReleaseDateCountry struct {
	ISO31661     string            `json:"iso_3166_1"`
	ReleaseDates []tmdbReleaseDate `json:"release_dates"`
}

type tmdbReleaseDate struct {
	Certification string `json:"certification"`
}

func NewMetadataDownloader(token string) *MetadataDownloader {
	return &MetadataDownloader{
		Token:        token,
		baseURL:      tmdbBaseURL,
		imageBaseURL: tmdbImageBaseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (d *MetadataDownloader) GetMovie(ctx context.Context, title string, year int) (models.Movie, error) {
	if strings.TrimSpace(title) == "" {
		return models.Movie{}, fmt.Errorf("movie title is required")
	}
	if year <= 0 {
		return models.Movie{}, fmt.Errorf("movie year is required")
	}

	result, err := d.searchMovie(ctx, title, year)
	if err != nil {
		return models.Movie{}, err
	}

	details, err := d.movieDetails(ctx, result.ID)
	if err != nil {
		return models.Movie{}, err
	}

	return movieFromTMDBDetails(details), nil
}

func (d *MetadataDownloader) SearchMovies(ctx context.Context, title string, year int) ([]models.MetadataMatch, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("movie title is required")
	}
	params := url.Values{}
	params.Set("query", title)
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	var searchResponse tmdbSearchResponse
	if err := d.get(ctx, "/search/movie", params, &searchResponse); err != nil {
		return nil, err
	}
	matches := make([]models.MetadataMatch, 0, len(searchResponse.Results))
	for _, result := range searchResponse.Results {
		matches = append(matches, models.MetadataMatch{
			TMDBID:        result.ID,
			Title:         result.Title,
			OriginalTitle: result.OriginalTitle,
			Year:          releaseYear(result.ReleaseDate),
			ReleaseDate:   result.ReleaseDate,
			Overview:      result.Overview,
			PosterPath:    result.PosterPath,
			BackdropPath:  result.BackdropPath,
			VoteAverage:   result.VoteAverage,
		})
	}
	return matches, nil
}

func (d *MetadataDownloader) GetMovieByID(ctx context.Context, tmdbID int) (models.Movie, error) {
	if tmdbID <= 0 {
		return models.Movie{}, fmt.Errorf("tmdb id is required")
	}

	details, err := d.movieDetails(ctx, tmdbID)
	if err != nil {
		return models.Movie{}, err
	}

	return movieFromTMDBDetails(details), nil
}

func (d *MetadataDownloader) GetTVShow(ctx context.Context, title string, year int) (models.TVShow, error) {
	if strings.TrimSpace(title) == "" {
		return models.TVShow{}, fmt.Errorf("TV show title is required")
	}
	if year <= 0 {
		return models.TVShow{}, fmt.Errorf("TV show year is required")
	}

	result, err := d.searchTVShow(ctx, title, year)
	if err != nil {
		return models.TVShow{}, err
	}
	return d.GetTVShowByID(ctx, result.ID)
}

func (d *MetadataDownloader) SearchTVShows(ctx context.Context, title string, year int) ([]models.MetadataMatch, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("TV show title is required")
	}
	params := url.Values{}
	params.Set("query", title)
	if year > 0 {
		params.Set("first_air_date_year", strconv.Itoa(year))
	}

	var searchResponse tmdbTVSearchResponse
	if err := d.get(ctx, "/search/tv", params, &searchResponse); err != nil {
		return nil, err
	}
	matches := make([]models.MetadataMatch, 0, len(searchResponse.Results))
	for _, result := range searchResponse.Results {
		matches = append(matches, models.MetadataMatch{
			TMDBID:        result.ID,
			Title:         result.Name,
			OriginalTitle: result.OriginalName,
			Year:          releaseYear(result.FirstAirDate),
			ReleaseDate:   result.FirstAirDate,
		})
	}
	return matches, nil
}

func (d *MetadataDownloader) GetTVShowByID(ctx context.Context, tmdbID int) (models.TVShow, error) {
	if tmdbID <= 0 {
		return models.TVShow{}, fmt.Errorf("tmdb id is required")
	}

	details, err := d.tvDetails(ctx, tmdbID)
	if err != nil {
		return models.TVShow{}, err
	}
	return tvShowFromTMDBDetails(details), nil
}

func (d *MetadataDownloader) DownloadArtwork(ctx context.Context, movie models.Movie, artworkDir string) (posterPath string, backdropPath string, err error) {
	if artworkDir == "" {
		artworkDir = filepath.Join("data", "artwork")
	}
	movieArtworkDir := filepath.Join(artworkDir, "movies", fmt.Sprintf("%d", movie.TMDBID))
	if err := os.MkdirAll(movieArtworkDir, 0o755); err != nil {
		return "", "", err
	}

	if movie.PosterPath != "" {
		posterPath, err = d.downloadImage(ctx, movie.PosterPath, "w500", filepath.Join(movieArtworkDir, "poster"+imageExtension(movie.PosterPath)))
		if err != nil {
			return "", "", err
		}
	}

	if movie.BackdropPath != "" {
		backdropPath, err = d.downloadImage(ctx, movie.BackdropPath, "original", filepath.Join(movieArtworkDir, "backdrop"+imageExtension(movie.BackdropPath)))
		if err != nil {
			return "", "", err
		}
	}

	return posterPath, backdropPath, nil
}

func (d *MetadataDownloader) DownloadTVArtwork(ctx context.Context, show models.TVShow, artworkDir string) (posterPath string, backdropPath string, err error) {
	if artworkDir == "" {
		artworkDir = filepath.Join("data", "artwork")
	}
	showArtworkDir := filepath.Join(artworkDir, "tvshows", fmt.Sprintf("%d", show.TMDBID))
	if err := os.MkdirAll(showArtworkDir, 0o755); err != nil {
		return "", "", err
	}
	if show.PosterPath != "" {
		posterPath, err = d.downloadImage(ctx, show.PosterPath, "w500", filepath.Join(showArtworkDir, "poster"+imageExtension(show.PosterPath)))
		if err != nil {
			return "", "", err
		}
	}
	if show.BackdropPath != "" {
		backdropPath, err = d.downloadImage(ctx, show.BackdropPath, "original", filepath.Join(showArtworkDir, "backdrop"+imageExtension(show.BackdropPath)))
		if err != nil {
			return "", "", err
		}
	}
	return posterPath, backdropPath, nil
}

func (d *MetadataDownloader) DownloadTVEpisodeThumbnails(ctx context.Context, show models.TVShow, episodePaths []string, imageRoot string) ([]models.TVEpisode, []string) {
	return d.DownloadTVEpisodeThumbnailsWithCache(ctx, show, episodePaths, imageRoot, nil)
}

func (d *MetadataDownloader) DownloadTVEpisodeThumbnailsWithCache(ctx context.Context, show models.TVShow, episodePaths []string, imageRoot string, cachedEpisodes []models.TVEpisode) ([]models.TVEpisode, []string) {
	if imageRoot == "" {
		imageRoot = filepath.Join("data", "artwork", "tvshows")
	}

	episodes := []models.TVEpisode{}
	errors := []string{}
	seen := map[string]struct{}{}
	cachedByPath := map[string]models.TVEpisode{}
	for _, episode := range cachedEpisodes {
		if cachedTVEpisodeComplete(episode) {
			cachedByPath[episode.Path] = episode
		}
	}
	for _, episodePath := range episodePaths {
		if episodePath == "" {
			continue
		}
		if _, ok := seen[episodePath]; ok {
			continue
		}
		seen[episodePath] = struct{}{}

		if cached, ok := cachedByPath[episodePath]; ok {
			episodes = append(episodes, cached)
			continue
		}

		parsed, ok := ParseTVEpisodeFilename(episodePath)
		if !ok {
			errors = append(errors, episodePath+": could not parse TV episode season/episode")
			continue
		}

		episode := models.TVEpisode{
			Path:    episodePath,
			Season:  parsed.Season,
			Episode: parsed.Episode,
		}
		details, err := d.tvEpisodeDetails(ctx, show.TMDBID, parsed.Season, parsed.Episode)
		if err != nil {
			errors = append(errors, episodePath+": "+err.Error())
			episodes = append(episodes, episode)
			continue
		}

		episode.Name = details.Name
		episode.Overview = details.Overview
		episode.AirDate = details.AirDate
		episode.StillPath = details.StillPath
		episode.RuntimeMinutes = details.Runtime
		if episode.Season == 0 && details.SeasonNumber > 0 {
			episode.Season = details.SeasonNumber
		}
		if episode.Episode == 0 && details.EpisodeNumber > 0 {
			episode.Episode = details.EpisodeNumber
		}

		if details.StillPath != "" {
			destPath := tvEpisodeThumbnailPath(imageRoot, show.TMDBID, episode.Season, episode.Episode, details.StillPath)
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				errors = append(errors, episodePath+": "+err.Error())
				episodes = append(episodes, episode)
				continue
			}
			thumbnailPath, err := d.downloadImage(ctx, details.StillPath, "w300", destPath)
			if err != nil {
				errors = append(errors, episodePath+": "+err.Error())
			} else {
				episode.ThumbnailLocalPath = thumbnailPath
			}
		}
		episodes = append(episodes, episode)
	}
	return episodes, errors
}

func cachedTVEpisodeComplete(episode models.TVEpisode) bool {
	if episode.Path == "" || episode.Season < 0 || episode.Episode <= 0 || episode.Name == "" {
		return false
	}
	if episode.StillPath == "" {
		return true
	}
	return existingFile(episode.ThumbnailLocalPath)
}

func (d *MetadataDownloader) searchMovie(ctx context.Context, title string, year int) (tmdbSearchMovie, error) {
	params := url.Values{}
	params.Set("query", title)
	params.Set("year", strconv.Itoa(year))

	var searchResponse tmdbSearchResponse
	if err := d.get(ctx, "/search/movie", params, &searchResponse); err != nil {
		return tmdbSearchMovie{}, err
	}
	if len(searchResponse.Results) == 0 {
		return tmdbSearchMovie{}, fmt.Errorf("no TMDB movie match for %q (%d)", title, year)
	}

	for _, result := range searchResponse.Results {
		if releaseYear(result.ReleaseDate) == year {
			return result, nil
		}
	}

	return searchResponse.Results[0], nil
}

func (d *MetadataDownloader) searchTVShow(ctx context.Context, title string, year int) (tmdbSearchTVShow, error) {
	params := url.Values{}
	params.Set("query", title)
	params.Set("first_air_date_year", strconv.Itoa(year))

	var searchResponse tmdbTVSearchResponse
	if err := d.get(ctx, "/search/tv", params, &searchResponse); err != nil {
		return tmdbSearchTVShow{}, err
	}
	if len(searchResponse.Results) == 0 {
		return tmdbSearchTVShow{}, fmt.Errorf("no TMDB TV show match for %q (%d)", title, year)
	}
	for _, result := range searchResponse.Results {
		if releaseYear(result.FirstAirDate) == year {
			return result, nil
		}
	}
	return searchResponse.Results[0], nil
}

func (d *MetadataDownloader) movieDetails(ctx context.Context, tmdbID int) (tmdbMovieDetails, error) {
	var details tmdbMovieDetails
	params := url.Values{}
	params.Set("append_to_response", "release_dates,credits,keywords")
	if err := d.get(ctx, fmt.Sprintf("/movie/%d", tmdbID), params, &details); err != nil {
		return tmdbMovieDetails{}, err
	}

	return details, nil
}

func (d *MetadataDownloader) tvDetails(ctx context.Context, tmdbID int) (tmdbTVDetails, error) {
	var details tmdbTVDetails
	params := url.Values{}
	params.Set("append_to_response", "content_ratings,credits,keywords")
	if err := d.get(ctx, fmt.Sprintf("/tv/%d", tmdbID), params, &details); err != nil {
		return tmdbTVDetails{}, err
	}
	return details, nil
}

func (d *MetadataDownloader) tvEpisodeDetails(ctx context.Context, tmdbID int, season int, episode int) (tmdbTVEpisodeDetails, error) {
	var details tmdbTVEpisodeDetails
	if tmdbID <= 0 {
		return details, fmt.Errorf("tmdb id is required")
	}
	if season < 0 || episode <= 0 {
		return details, fmt.Errorf("TV episode season/episode is required")
	}
	if err := d.get(ctx, fmt.Sprintf("/tv/%d/season/%d/episode/%d", tmdbID, season, episode), url.Values{}, &details); err != nil {
		return tmdbTVEpisodeDetails{}, err
	}
	return details, nil
}

func (d *MetadataDownloader) get(ctx context.Context, path string, params url.Values, dest any) error {
	if strings.TrimSpace(d.Token) == "" {
		return fmt.Errorf("TMDB_READ_TOKEN is required")
	}

	requestURL, err := url.Parse(d.baseURL + path)
	if err != nil {
		return err
	}
	requestURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+d.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("TMDB request failed: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return err
	}

	return nil
}

func (d *MetadataDownloader) downloadImage(ctx context.Context, imagePath string, size string, destPath string) (string, error) {
	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		return destPath, nil
	}

	imagePath = strings.TrimPrefix(imagePath, "/")
	imageURL := fmt.Sprintf("%s/%s/%s", d.imageBaseURL, size, imagePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("TMDB image request failed: %s", resp.Status)
	}

	file, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}

	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}

	return destPath, nil
}

func movieFromTMDBDetails(details tmdbMovieDetails) models.Movie {
	genres := make([]string, 0, len(details.Genres))
	for _, genre := range details.Genres {
		if genre.Name != "" {
			genres = append(genres, genre.Name)
		}
	}
	studio := ""
	if len(details.Companies) > 0 {
		studio = details.Companies[0].Name
	}

	return models.Movie{
		TMDBID:         details.ID,
		Title:          details.Title,
		OriginalTitle:  details.OriginalTitle,
		Year:           releaseYear(details.ReleaseDate),
		ReleaseDate:    details.ReleaseDate,
		Overview:       details.Overview,
		PosterPath:     details.PosterPath,
		BackdropPath:   details.BackdropPath,
		RuntimeMinutes: details.Runtime,
		Genres:         genres,
		VoteAverage:    details.VoteAverage,
		Rating:         mpaaRating(details.ReleaseDates),
		Studio:         studio,
		Actors:         tvCreditNames(details.Credits.Cast, "", 12),
		Directors:      tvCrewNames(details.Credits.Crew, map[string]struct{}{"Director": {}}, 0),
		Writers:        tvCrewNames(details.Credits.Crew, map[string]struct{}{"Writer": {}, "Screenplay": {}, "Story": {}}, 0),
		Countries:      movieCountries(details.ProductionCountries),
		Languages:      movieLanguages(details),
		Tags:           movieKeywordNames(details.Keywords),
	}
}

func movieCountries(countries []tmdbCountry) []string {
	values := []string{}
	for _, country := range countries {
		if country.ISO31661 != "" {
			values = append(values, country.ISO31661)
			continue
		}
		if country.Name != "" {
			values = append(values, country.Name)
		}
	}
	return uniqueStrings(values)
}

func movieLanguages(details tmdbMovieDetails) []string {
	languages := []string{}
	for _, language := range details.SpokenLanguages {
		switch {
		case language.EnglishName != "":
			languages = append(languages, language.EnglishName)
		case language.Name != "":
			languages = append(languages, language.Name)
		case language.ISO6391 != "":
			languages = append(languages, language.ISO6391)
		}
	}
	if len(languages) == 0 && details.OriginalLanguage != "" {
		languages = append(languages, details.OriginalLanguage)
	}
	return uniqueStrings(languages)
}

func movieKeywordNames(keywords tmdbMovieKeywords) []string {
	names := make([]string, 0, len(keywords.Keywords))
	for _, keyword := range keywords.Keywords {
		if keyword.Name != "" {
			names = append(names, keyword.Name)
		}
	}
	return uniqueStrings(names)
}

func tvShowFromTMDBDetails(details tmdbTVDetails) models.TVShow {
	genres := make([]string, 0, len(details.Genres))
	for _, genre := range details.Genres {
		if genre.Name != "" {
			genres = append(genres, genre.Name)
		}
	}
	network := ""
	if len(details.Networks) > 0 {
		network = details.Networks[0].Name
	}
	studio := ""
	if len(details.ProductionCompanies) > 0 {
		studio = details.ProductionCompanies[0].Name
	}

	return models.TVShow{
		TMDBID:       details.ID,
		Name:         details.Name,
		OriginalName: details.OriginalName,
		Year:         releaseYear(details.FirstAirDate),
		FirstAirDate: details.FirstAirDate,
		Overview:     details.Overview,
		PosterPath:   details.PosterPath,
		BackdropPath: details.BackdropPath,
		Genres:       genres,
		Rating:       tvContentRating(details.ContentRatings),
		Seasons:      details.NumberSeasons,
		Episodes:     details.NumberEpisodes,
		Network:      network,
		Studio:       studio,
		Actors:       tvCreditNames(details.Credits.Cast, "", 12),
		Directors:    tvCrewNames(details.Credits.Crew, map[string]struct{}{"Director": {}}, 0),
		Writers:      uniqueStrings(append(tvCrewNames(details.Credits.Crew, map[string]struct{}{"Writer": {}, "Screenplay": {}, "Teleplay": {}, "Story": {}}, 0), tvPersonNames(details.CreatedBy, 0)...)),
		Countries:    uniqueStrings(details.OriginCountry),
		Languages:    tvLanguages(details),
		SeriesStatus: details.Status,
		Tags:         tvKeywordNames(details.Keywords),
	}
}

func tvCreditNames(people []tmdbPerson, job string, limit int) []string {
	names := []string{}
	for _, person := range people {
		if person.Name == "" {
			continue
		}
		if job != "" && person.Job != job {
			continue
		}
		names = append(names, person.Name)
		if limit > 0 && len(names) >= limit {
			break
		}
	}
	return uniqueStrings(names)
}

func tvCrewNames(people []tmdbPerson, jobs map[string]struct{}, limit int) []string {
	names := []string{}
	for _, person := range people {
		if person.Name == "" {
			continue
		}
		if _, ok := jobs[person.Job]; !ok {
			continue
		}
		names = append(names, person.Name)
		if limit > 0 && len(names) >= limit {
			break
		}
	}
	return uniqueStrings(names)
}

func tvPersonNames(people []tmdbPerson, limit int) []string {
	names := []string{}
	for _, person := range people {
		if person.Name == "" {
			continue
		}
		names = append(names, person.Name)
		if limit > 0 && len(names) >= limit {
			break
		}
	}
	return uniqueStrings(names)
}

func tvLanguages(details tmdbTVDetails) []string {
	languages := []string{}
	for _, language := range details.SpokenLanguages {
		switch {
		case language.EnglishName != "":
			languages = append(languages, language.EnglishName)
		case language.Name != "":
			languages = append(languages, language.Name)
		case language.ISO6391 != "":
			languages = append(languages, language.ISO6391)
		}
	}
	if len(languages) == 0 && details.OriginalLanguage != "" {
		languages = append(languages, details.OriginalLanguage)
	}
	return uniqueStrings(languages)
}

func tvKeywordNames(keywords tmdbKeywords) []string {
	names := make([]string, 0, len(keywords.Results))
	for _, keyword := range keywords.Results {
		if keyword.Name != "" {
			names = append(names, keyword.Name)
		}
	}
	return uniqueStrings(names)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func tvContentRating(contentRatings tmdbContentRatings) string {
	for _, rating := range contentRatings.Results {
		if rating.ISO31661 == "US" && rating.Rating != "" {
			return rating.Rating
		}
	}
	return ""
}

func mpaaRating(releaseDates tmdbReleaseDates) string {
	for _, country := range releaseDates.Results {
		if country.ISO31661 != "US" {
			continue
		}
		for _, releaseDate := range country.ReleaseDates {
			if releaseDate.Certification != "" {
				return releaseDate.Certification
			}
		}
	}
	return ""
}

func releaseYear(releaseDate string) int {
	if len(releaseDate) < 4 {
		return 0
	}

	year, err := strconv.Atoi(releaseDate[:4])
	if err != nil {
		return 0
	}

	return year
}

func tvEpisodeThumbnailPath(root string, showTMDBID int, season int, episode int, stillPath string) string {
	return filepath.Join(root, fmt.Sprintf("%d", showTMDBID), "seasons", fmt.Sprintf("%02d", season), "episodes", fmt.Sprintf("%02d%s", episode, imageExtension(stillPath)))
}

func imageExtension(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return ".jpg"
	}
	return ext
}

func existingFile(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
