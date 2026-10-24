package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"retrotv/internal/models"
)

func TestGetMovieSearchesAndFetchesDetails(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization header = %q", r.Header.Get("Authorization"))
		}

		switch r.URL.Path {
		case "/search/movie":
			if r.URL.Query().Get("query") != "Die Hard" {
				t.Fatalf("query = %q, want Die Hard", r.URL.Query().Get("query"))
			}
			if r.URL.Query().Get("year") != "1988" {
				t.Fatalf("year = %q, want 1988", r.URL.Query().Get("year"))
			}
			return jsonResponse(t, tmdbSearchResponse{
				Results: []tmdbSearchMovie{
					{
						ID:          562,
						Title:       "Die Hard",
						ReleaseDate: "1988-07-15",
					},
				},
			})
		case "/movie/562":
			if r.URL.Query().Get("append_to_response") != "release_dates,credits,keywords" {
				t.Fatalf("append_to_response = %q", r.URL.Query().Get("append_to_response"))
			}
			return jsonResponse(t, tmdbMovieDetails{
				ID:            562,
				Title:         "Die Hard",
				OriginalTitle: "Die Hard",
				ReleaseDate:   "1988-07-15",
				Overview:      "NYPD cop John McClane tries to save hostages.",
				PosterPath:    "/poster.jpg",
				BackdropPath:  "/backdrop.jpg",
				Runtime:       132,
				Genres: []tmdbGenre{
					{Name: "Action"},
					{Name: "Thriller"},
				},
				Companies: []tmdbCompany{
					{Name: "20th Century Fox"},
				},
				ProductionCountries: []tmdbCountry{{ISO31661: "US", Name: "United States of America"}},
				SpokenLanguages:     []tmdbLanguage{{EnglishName: "English", ISO6391: "en"}},
				Credits: tmdbCredits{
					Cast: []tmdbPerson{{Name: "Bruce Willis"}},
					Crew: []tmdbPerson{
						{Name: "John McTiernan", Job: "Director"},
						{Name: "Jeb Stuart", Job: "Screenplay"},
					},
				},
				Keywords:    tmdbMovieKeywords{Keywords: []tmdbKeyword{{Name: "christmas"}}},
				VoteAverage: 7.8,
				ReleaseDates: tmdbReleaseDates{
					Results: []tmdbReleaseDateCountry{
						{
							ISO31661: "US",
							ReleaseDates: []tmdbReleaseDate{
								{Certification: "R"},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		return nil, nil
	})

	downloader := NewMetadataDownloader("test-token")
	downloader.baseURL = "https://tmdb.test"
	downloader.client = &http.Client{Transport: transport}

	movie, err := downloader.GetMovie(context.Background(), "Die Hard", 1988)
	if err != nil {
		t.Fatal(err)
	}

	if movie.TMDBID != 562 {
		t.Fatalf("TMDBID = %d, want 562", movie.TMDBID)
	}
	if movie.Title != "Die Hard" {
		t.Fatalf("Title = %q, want Die Hard", movie.Title)
	}
	if movie.Year != 1988 {
		t.Fatalf("Year = %d, want 1988", movie.Year)
	}
	if movie.RuntimeMinutes != 132 {
		t.Fatalf("RuntimeMinutes = %d, want 132", movie.RuntimeMinutes)
	}
	if len(movie.Genres) != 2 || movie.Genres[0] != "Action" || movie.Genres[1] != "Thriller" {
		t.Fatalf("Genres = %#v, want Action and Thriller", movie.Genres)
	}
	if movie.Rating != "R" {
		t.Fatalf("Rating = %q, want R", movie.Rating)
	}
	if movie.Studio != "20th Century Fox" {
		t.Fatalf("Studio = %q, want 20th Century Fox", movie.Studio)
	}
	if !reflect.DeepEqual(movie.Actors, []string{"Bruce Willis"}) ||
		!reflect.DeepEqual(movie.Directors, []string{"John McTiernan"}) ||
		!reflect.DeepEqual(movie.Writers, []string{"Jeb Stuart"}) ||
		!reflect.DeepEqual(movie.Countries, []string{"US"}) ||
		!reflect.DeepEqual(movie.Languages, []string{"English"}) ||
		!reflect.DeepEqual(movie.Tags, []string{"christmas"}) {
		t.Fatalf("movie metadata = %#v", movie)
	}
}

func TestGetMovieRequiresToken(t *testing.T) {
	downloader := NewMetadataDownloader("")

	_, err := downloader.GetMovie(context.Background(), "Die Hard", 1988)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetTVShowStoresExtendedMetadata(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization header = %q", r.Header.Get("Authorization"))
		}

		switch r.URL.Path {
		case "/search/tv":
			if r.URL.Query().Get("query") != "Columbo" {
				t.Fatalf("query = %q, want Columbo", r.URL.Query().Get("query"))
			}
			if r.URL.Query().Get("first_air_date_year") != "1971" {
				t.Fatalf("first_air_date_year = %q, want 1971", r.URL.Query().Get("first_air_date_year"))
			}
			return jsonResponse(t, tmdbTVSearchResponse{
				Results: []tmdbSearchTVShow{{ID: 873, Name: "Columbo", FirstAirDate: "1971-09-15"}},
			})
		case "/tv/873":
			if r.URL.Query().Get("append_to_response") != "content_ratings,credits,keywords" {
				t.Fatalf("append_to_response = %q", r.URL.Query().Get("append_to_response"))
			}
			return jsonResponse(t, tmdbTVDetails{
				ID:            873,
				Name:          "Columbo",
				OriginalName:  "Columbo",
				FirstAirDate:  "1971-09-15",
				Overview:      "A detective solves murders.",
				PosterPath:    "/poster.jpg",
				BackdropPath:  "/backdrop.jpg",
				Genres:        []tmdbGenre{{Name: "Mystery"}},
				NumberSeasons: 10,
				Networks:      []tmdbNetwork{{Name: "NBC"}},
				ProductionCompanies: []tmdbCompany{
					{Name: "Universal Television"},
				},
				CreatedBy:        []tmdbPerson{{Name: "Richard Levinson"}},
				OriginCountry:    []string{"US"},
				OriginalLanguage: "en",
				SpokenLanguages:  []tmdbLanguage{{EnglishName: "English", ISO6391: "en"}},
				Status:           "Ended",
				Credits: tmdbCredits{
					Cast: []tmdbPerson{{Name: "Peter Falk"}},
					Crew: []tmdbPerson{
						{Name: "Steven Spielberg", Job: "Director"},
						{Name: "William Link", Job: "Writer"},
					},
				},
				Keywords: tmdbKeywords{Results: []tmdbKeyword{{Name: "detective"}}},
				ContentRatings: tmdbContentRatings{
					Results: []tmdbContentRating{{ISO31661: "US", Rating: "TV-PG"}},
				},
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		return nil, nil
	})

	downloader := NewMetadataDownloader("test-token")
	downloader.baseURL = "https://tmdb.test"
	downloader.client = &http.Client{Transport: transport}

	show, err := downloader.GetTVShow(context.Background(), "Columbo", 1971)
	if err != nil {
		t.Fatal(err)
	}
	if show.Studio != "Universal Television" ||
		!reflect.DeepEqual(show.Actors, []string{"Peter Falk"}) ||
		!reflect.DeepEqual(show.Directors, []string{"Steven Spielberg"}) ||
		!reflect.DeepEqual(show.Writers, []string{"William Link", "Richard Levinson"}) ||
		!reflect.DeepEqual(show.Countries, []string{"US"}) ||
		!reflect.DeepEqual(show.Languages, []string{"English"}) ||
		show.SeriesStatus != "Ended" ||
		!reflect.DeepEqual(show.Tags, []string{"detective"}) {
		t.Fatalf("show metadata = %#v", show)
	}
}

func TestDownloadArtworkKeepsExistingFiles(t *testing.T) {
	requests := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("new-image")),
			Header:     make(http.Header),
		}, nil
	})

	artworkDir := t.TempDir()
	posterPath := filepath.Join(artworkDir, "movies", "562", "poster.jpg")
	if err := os.MkdirAll(filepath.Dir(posterPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(posterPath, []byte("cached-image"), 0o644); err != nil {
		t.Fatal(err)
	}

	downloader := NewMetadataDownloader("test-token")
	downloader.client = &http.Client{Transport: transport}
	poster, backdrop, err := downloader.DownloadArtwork(context.Background(), models.Movie{
		TMDBID:     562,
		PosterPath: "/poster.jpg",
	}, artworkDir)
	if err != nil {
		t.Fatal(err)
	}
	if poster != posterPath {
		t.Fatalf("poster = %q, want %q", poster, posterPath)
	}
	if backdrop != "" {
		t.Fatalf("backdrop = %q, want empty", backdrop)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
	content, err := os.ReadFile(posterPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "cached-image" {
		t.Fatalf("poster content = %q, want cached-image", content)
	}
}

func TestDownloadTVEpisodeThumbnailsUsesCleanDirectoryStructure(t *testing.T) {
	requests := []string{}
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.String())
		if r.Header.Get("Authorization") != "Bearer test-token" && r.URL.Host == "tmdb.test" {
			t.Fatalf("Authorization header = %q", r.Header.Get("Authorization"))
		}

		switch r.URL.String() {
		case "https://tmdb.test/tv/777/season/4/episode/1":
			return jsonResponse(t, tmdbTVEpisodeDetails{
				ID:            9001,
				Name:          "North by North Quahog",
				Overview:      "Brian and Stewie have a problem.",
				AirDate:       "2005-05-01",
				StillPath:     "/stills/family-guy-s04e01.jpg",
				Runtime:       22,
				SeasonNumber:  4,
				EpisodeNumber: 1,
			})
		case "https://images.test/w300/stills/family-guy-s04e01.jpg":
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString("episode-still")),
			}, nil
		default:
			t.Fatalf("unexpected request %q", r.URL.String())
		}

		return nil, nil
	})

	imageRoot := filepath.Join(t.TempDir(), "data", "artwork", "tvshows")
	downloader := NewMetadataDownloader("test-token")
	downloader.baseURL = "https://tmdb.test"
	downloader.imageBaseURL = "https://images.test"
	downloader.client = &http.Client{Transport: transport}

	episodes, errors := downloader.DownloadTVEpisodeThumbnails(context.Background(), models.TVShow{
		TMDBID: 777,
		Name:   "Family Guy",
	}, []string{"/tv/Family Guy/Season 04/Family.Guy.S04E01.North.by.North.Quahog.mkv"}, imageRoot)
	if len(errors) > 0 {
		t.Fatalf("errors = %#v", errors)
	}
	if len(episodes) != 1 {
		t.Fatalf("len(episodes) = %d, want 1", len(episodes))
	}

	wantPath := filepath.Join(imageRoot, "777", "seasons", "04", "episodes", "01.jpg")
	if episodes[0].ThumbnailLocalPath != wantPath {
		t.Fatalf("ThumbnailLocalPath = %q, want %q", episodes[0].ThumbnailLocalPath, wantPath)
	}
	if episodes[0].Name != "North by North Quahog" || episodes[0].RuntimeMinutes != 22 {
		t.Fatalf("episode metadata = %#v", episodes[0])
	}
	content, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "episode-still" {
		t.Fatalf("thumbnail content = %q, want episode-still", content)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %#v, want metadata and image requests", requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(t *testing.T, value any) (*http.Response, error) {
	t.Helper()

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(value); err != nil {
		t.Fatal(err)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(&body),
	}, nil
}
