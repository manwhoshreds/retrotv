package metadata

import "testing"

func TestParseMovieFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		title    string
		year     int
		tmdbID   string
		ok       bool
	}{
		{
			name:     "kodi movie filename",
			filename: "Blade Runner 2049 (2017).mkv",
			title:    "Blade Runner 2049",
			year:     2017,
			ok:       true,
		},
		{
			name:     "title containing dot",
			filename: "Mr. Nobody (2009).mkv",
			title:    "Mr. Nobody",
			year:     2009,
			ok:       true,
		},
		{
			name:     "identifier with hyphen form",
			filename: "Alien (1979) [tmdb-348].mp4",
			title:    "Alien",
			year:     1979,
			tmdbID:   "348",
			ok:       true,
		},
		{
			name:     "split file suffix",
			filename: "The Lord of the Rings The Fellowship of the Ring (2001)A.mkv",
			title:    "The Lord of the Rings The Fellowship of the Ring",
			year:     2001,
			ok:       true,
		},
		{
			name:     "title containing parentheses",
			filename: "Halloween (Extended Cut) (1978).avi",
			title:    "Halloween (Extended Cut)",
			year:     1978,
			ok:       true,
		},
		{
			name:     "missing year",
			filename: "Die Hard.mkv",
			ok:       false,
		},
		{
			name:     "underscored kodi folder name",
			filename: "16_Blocks_(2006).mkv",
			title:    "16 Blocks",
			year:     2006,
			ok:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, ok := ParseMovieFilename(test.filename)
			if ok != test.ok {
				t.Fatalf("ok = %v, want %v", ok, test.ok)
			}
			if !ok {
				return
			}
			if parsed.Title != test.title {
				t.Fatalf("Title = %q, want %q", parsed.Title, test.title)
			}
			if parsed.Year != test.year {
				t.Fatalf("Year = %d, want %d", parsed.Year, test.year)
			}
			if parsed.TMDBID != test.tmdbID {
				t.Fatalf("TMDBID = %q, want %q", parsed.TMDBID, test.tmdbID)
			}
		})
	}
}

func TestParseMovieName(t *testing.T) {
	parsed, ok := ParseMovieName("Blade Runner (2017) {tmdb=335984}")
	if !ok {
		t.Fatal("ParseMovieName returned false")
	}
	if parsed.Title != "Blade Runner" {
		t.Fatalf("Title = %q, want Blade Runner", parsed.Title)
	}
	if parsed.Year != 2017 {
		t.Fatalf("Year = %d, want 2017", parsed.Year)
	}
	if parsed.TMDBID != "335984" {
		t.Fatalf("TMDBID = %q, want 335984", parsed.TMDBID)
	}
}
