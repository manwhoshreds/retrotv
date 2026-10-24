package metadata

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

import "retrotv/internal/models"

var videoExtensions = map[string]bool{
	".mp4": true,
	".mkv": true,
	".avi": true,
	".mov": true,
	".m4v": true,
}

func ScanMovies(root string) ([]models.MovieFile, error) {
	return ScanMoviesWithIndex(root, nil, nil)
}

func ScanMoviesWithProgress(root string, progress func(path string, discovered int)) ([]models.MovieFile, error) {
	return ScanMoviesWithIndex(root, nil, progress)
}

// ScanMoviesWithIndex avoids filesystem stat calls for paths already present in
// the scan index. This keeps directory discovery fast on network and FUSE
// mounts, where stat can be dramatically slower than directory enumeration.
func ScanMoviesWithIndex(root string, indexed map[string]models.MediaMetadata, progress func(path string, discovered int)) ([]models.MovieFile, error) {
	var movies []models.MovieFile

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		if videoExtensions[ext] {
			fingerprint, indexedPath := indexed[path]
			if !indexedPath {
				info, infoErr := d.Info()
				if infoErr != nil {
					return infoErr
				}
				fingerprint = models.MediaMetadata{FileSize: info.Size(), ModTimeUnix: info.ModTime().Unix()}
			}
			parsed, ok := ParseMovieName(filepath.Base(filepath.Dir(path)))
			if !ok {
				parsed, _ = ParseMovieFilename(path)
				if parsed.Title == "" {
					parsed = ParseLocalMediaName(filepath.Base(path))
				}
			}

			movie := models.MovieFile{
				Path:          path,
				Filename:      filepath.Base(path),
				Title:         parsed.Title,
				Year:          parsed.Year,
				TMDBID:        parsed.TMDBID,
				MediaMetadata: fingerprint,
			}
			movies = append(movies, movie)
			if progress != nil {
				progress(path, len(movies))
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println("walk error:", err)
		return nil, err
	}

	fmt.Println("video count:", len(movies))
	return movies, nil
}

func ScanTVShows(root string) ([]models.TVShowFile, error) {
	return ScanTVShowsWithIndex(root, nil, nil)
}

func ScanTVShowsWithProgress(root string, progress func(path string, discovered int)) ([]models.TVShowFile, error) {
	return ScanTVShowsWithIndex(root, nil, progress)
}

func ScanTVShowsWithIndex(root string, indexed map[string]models.MediaMetadata, progress func(path string, discovered int)) ([]models.TVShowFile, error) {
	shows := map[string]models.TVShowFile{}
	discovered := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !videoExtensions[ext] {
			return nil
		}
		if _, ok := ParseTVEpisodeFilename(path); !ok {
			return nil
		}
		fingerprint, indexedPath := indexed[path]
		if !indexedPath {
			info, infoErr := d.Info()
			if infoErr != nil {
				return infoErr
			}
			fingerprint = models.MediaMetadata{FileSize: info.Size(), ModTimeUnix: info.ModTime().Unix()}
		}

		showPath, ok := tvShowDirectory(root, path)
		if !ok {
			return nil
		}
		parsed := ParseLocalMediaName(filepath.Base(showPath))

		show := shows[showPath]
		if show.Path == "" {
			show = models.TVShowFile{
				Path:                showPath,
				Title:               parsed.Title,
				Year:                parsed.Year,
				TMDBID:              parsed.TMDBID,
				EpisodeFingerprints: map[string]models.MediaMetadata{},
			}
		}
		show.EpisodeCount++
		show.EpisodePaths = append(show.EpisodePaths, path)
		show.EpisodeFingerprints[path] = fingerprint
		shows[showPath] = show
		discovered++
		if progress != nil {
			progress(path, discovered)
		}
		return nil
	})
	if err != nil {
		fmt.Println("walk error:", err)
		return nil, err
	}

	result := make([]models.TVShowFile, 0, len(shows))
	for _, show := range shows {
		result = append(result, show)
	}
	return result, nil
}

func ParseLocalMediaName(name string) ParsedMovieFilename {
	if parsed, ok := ParseMovieName(name); ok {
		return parsed
	}
	title := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.Trim(title, " .-_")
	title = strings.Join(strings.Fields(title), " ")
	return ParsedMovieFilename{Title: title}
}

func tvShowDirectory(root string, episodePath string) (string, bool) {
	rel, err := filepath.Rel(root, episodePath)
	if err != nil {
		return "", false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) < 2 {
		return "", false
	}
	return filepath.Join(root, parts[0]), true
}
