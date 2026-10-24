package metadata

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ParsedMovieFilename struct {
	Title  string
	Year   int
	TMDBID string
}

type ParsedTVEpisodeFilename struct {
	Season  int
	Episode int
}

var (
	kodiYearPattern       = regexp.MustCompile(`\((\d{4})\)`)
	kodiIdentifierPattern = regexp.MustCompile(`[\[{](?i:tmdb)(?:id)?[-=](\d+)[\]}]`)
	kodiEpisodePattern    = regexp.MustCompile(`(?i)s(\d{1,2})e(\d{1,3})`)
)

func ParseMovieFilename(filename string) (ParsedMovieFilename, bool) {
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return ParseMovieName(name)
}

func ParseTVShowName(name string) (ParsedMovieFilename, bool) {
	return ParseMovieName(name)
}

func ParseTVEpisodeFilename(filename string) (ParsedTVEpisodeFilename, bool) {
	name := filepath.Base(filename)
	match := kodiEpisodePattern.FindStringSubmatch(name)
	if len(match) != 3 {
		return ParsedTVEpisodeFilename{}, false
	}
	season, err := strconv.Atoi(match[1])
	if err != nil {
		return ParsedTVEpisodeFilename{}, false
	}
	episode, err := strconv.Atoi(match[2])
	if err != nil {
		return ParsedTVEpisodeFilename{}, false
	}
	return ParsedTVEpisodeFilename{Season: season, Episode: episode}, true
}

func ParseMovieName(name string) (ParsedMovieFilename, bool) {
	name = strings.TrimSpace(name)

	matches := kodiYearPattern.FindAllStringSubmatchIndex(name, -1)
	if len(matches) == 0 {
		return ParsedMovieFilename{}, false
	}

	yearMatch := matches[len(matches)-1]
	year, err := strconv.Atoi(name[yearMatch[2]:yearMatch[3]])
	if err != nil {
		return ParsedMovieFilename{}, false
	}

	title := strings.TrimSpace(name[:yearMatch[0]])
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.Trim(title, " .-_")
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return ParsedMovieFilename{}, false
	}

	tmdbID := ""
	if identifier := kodiIdentifierPattern.FindStringSubmatch(name[yearMatch[1]:]); len(identifier) == 2 {
		tmdbID = identifier[1]
	}

	return ParsedMovieFilename{
		Title:  title,
		Year:   year,
		TMDBID: tmdbID,
	}, true
}
