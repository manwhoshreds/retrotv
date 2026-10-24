package mediapath

import (
	"os"
	"path/filepath"
	"strings"
)

func Resolve(path string) string {
	if regularFileExists(path) {
		return path
	}
	for _, candidate := range siblingMovieRootPaths(path) {
		if regularFileExists(candidate) {
			return candidate
		}
	}
	return path
}

func RegularFileExists(path string) bool {
	return regularFileExists(Resolve(path))
}

func regularFileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func siblingMovieRootPaths(path string) []string {
	normalized := filepath.ToSlash(filepath.Clean(path))
	switch {
	case strings.HasPrefix(normalized, "data/movies/"):
		return []string{filepath.FromSlash("data/movies2/" + strings.TrimPrefix(normalized, "data/movies/"))}
	case strings.HasPrefix(normalized, "data/movies2/"):
		return []string{filepath.FromSlash("data/movies/" + strings.TrimPrefix(normalized, "data/movies2/"))}
	default:
		return nil
	}
}
