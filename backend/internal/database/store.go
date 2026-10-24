package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"retrotv/internal/models"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if path == "" {
		path = filepath.Join("data", "retrotv.sqlite")
	}

	params := url.Values{}
	params.Add("_pragma", "busy_timeout(5000)")
	params.Add("_pragma", "foreign_keys(ON)")
	params.Add("_pragma", "journal_mode(WAL)")
	db, err := sql.Open("sqlite", path+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.Migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS movies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path TEXT NOT NULL UNIQUE,
			filename TEXT NOT NULL,
			local_title TEXT NOT NULL,
			local_year INTEGER NOT NULL,
			tmdb_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			original_title TEXT NOT NULL,
			year INTEGER NOT NULL,
			release_date TEXT NOT NULL,
			overview TEXT NOT NULL,
			poster_path TEXT NOT NULL,
			backdrop_path TEXT NOT NULL,
			poster_local_path TEXT NOT NULL,
			backdrop_local_path TEXT NOT NULL,
			runtime_minutes INTEGER NOT NULL,
			genres_json TEXT NOT NULL,
			vote_average REAL NOT NULL,
			rating TEXT NOT NULL DEFAULT '',
			studio TEXT NOT NULL DEFAULT '',
			actors_json TEXT NOT NULL DEFAULT '[]',
			directors_json TEXT NOT NULL DEFAULT '[]',
			writers_json TEXT NOT NULL DEFAULT '[]',
			countries_json TEXT NOT NULL DEFAULT '[]',
			languages_json TEXT NOT NULL DEFAULT '[]',
			tags_json TEXT NOT NULL DEFAULT '[]',
			media_metadata_json TEXT NOT NULL DEFAULT '{}',
			last_seen_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			missing_since TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);

		CREATE INDEX IF NOT EXISTS idx_movies_title ON movies(title);
		CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);

		CREATE TABLE IF NOT EXISTS tv_shows (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			local_title TEXT NOT NULL,
			local_year INTEGER NOT NULL,
			local_episode_count INTEGER NOT NULL,
			episode_paths_json TEXT NOT NULL DEFAULT '[]',
			episode_metadata_json TEXT NOT NULL DEFAULT '[]',
			tmdb_id INTEGER NOT NULL,
			metadata_version INTEGER NOT NULL DEFAULT 1,
			name TEXT NOT NULL,
			original_name TEXT NOT NULL,
			year INTEGER NOT NULL,
			first_air_date TEXT NOT NULL,
			overview TEXT NOT NULL,
			poster_path TEXT NOT NULL,
			backdrop_path TEXT NOT NULL,
			poster_local_path TEXT NOT NULL,
			backdrop_local_path TEXT NOT NULL,
			genres_json TEXT NOT NULL,
			rating TEXT NOT NULL,
			seasons INTEGER NOT NULL,
			episodes INTEGER NOT NULL,
			network TEXT NOT NULL,
			studio TEXT NOT NULL DEFAULT '',
			actors_json TEXT NOT NULL DEFAULT '[]',
			directors_json TEXT NOT NULL DEFAULT '[]',
			writers_json TEXT NOT NULL DEFAULT '[]',
			countries_json TEXT NOT NULL DEFAULT '[]',
			languages_json TEXT NOT NULL DEFAULT '[]',
			series_status TEXT NOT NULL DEFAULT '',
			tags_json TEXT NOT NULL DEFAULT '[]',
			last_seen_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			missing_since TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);

		CREATE INDEX IF NOT EXISTS idx_tv_shows_name ON tv_shows(name);
		CREATE INDEX IF NOT EXISTS idx_tv_shows_tmdb_id ON tv_shows(tmdb_id);

		CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			logo_path TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK (content_type IN ('movie', 'tv', 'mixed')),
			sort_order INTEGER NOT NULL DEFAULT 0,
			airing INTEGER NOT NULL DEFAULT 1,
			channel_bug_enabled INTEGER NOT NULL DEFAULT 0,
			channel_bug_asset_path TEXT NOT NULL DEFAULT '',
			channel_bug_position TEXT NOT NULL DEFAULT '',
			channel_bug_width INTEGER NOT NULL DEFAULT 0,
			channel_bug_opacity REAL NOT NULL DEFAULT 0,
			channel_bug_margin_x INTEGER NOT NULL DEFAULT 0,
			channel_bug_margin_y INTEGER NOT NULL DEFAULT 0,
			rating_overlay_enabled INTEGER NOT NULL DEFAULT 0,
			rating_overlay_position TEXT NOT NULL DEFAULT '',
			rating_overlay_duration_seconds INTEGER NOT NULL DEFAULT 0,
			now_playing_enabled INTEGER NOT NULL DEFAULT 0,
			now_playing_position TEXT NOT NULL DEFAULT '',
			now_playing_duration_seconds INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);

		CREATE TABLE IF NOT EXISTS channel_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			media_type TEXT NOT NULL DEFAULT '' CHECK (media_type IN ('', 'movie', 'tv')),
			field TEXT NOT NULL,
			operator TEXT NOT NULL,
			value TEXT NOT NULL,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS collections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			artwork_path TEXT NOT NULL DEFAULT '',
			artwork_mode TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);

		CREATE TABLE IF NOT EXISTS collection_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			collection_id INTEGER NOT NULL,
			media_type TEXT NOT NULL DEFAULT '' CHECK (media_type IN ('', 'movie', 'tv')),
			field TEXT NOT NULL,
			operator TEXT NOT NULL,
			value TEXT NOT NULL,
			FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS collection_media (
			collection_id INTEGER NOT NULL,
			media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv')),
			media_id INTEGER NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (collection_id, media_type, media_id),
			FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS collection_excluded_media (
			collection_id INTEGER NOT NULL,
			media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv')),
			media_id INTEGER NOT NULL,
			PRIMARY KEY (collection_id, media_type, media_id),
			FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS collection_artwork_media (
			collection_id INTEGER NOT NULL,
			media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv')),
			media_id INTEGER NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (collection_id, media_type, media_id),
			FOREIGN KEY (collection_id) REFERENCES collections(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS channel_programming_blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			placement TEXT NOT NULL DEFAULT 'timeline',
			collection_id INTEGER NOT NULL DEFAULT 0,
			days_json TEXT NOT NULL DEFAULT '[]',
			content_source TEXT NOT NULL DEFAULT 'rules',
			rule_set TEXT NOT NULL DEFAULT '',
			schedule_mode TEXT NOT NULL DEFAULT 'weekly',
			exact_date TEXT NOT NULL DEFAULT '',
			excluded_dates_json TEXT NOT NULL DEFAULT '[]',
			playback_mode TEXT NOT NULL DEFAULT 'shuffle',
			priority INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS channel_programming_block_media (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			block_id INTEGER NOT NULL,
			media_type TEXT NOT NULL CHECK (media_type IN ('movie', 'tv')),
			media_id INTEGER NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (block_id) REFERENCES channel_programming_blocks(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_channel_rules_channel_id ON channel_rules(channel_id);

		CREATE TABLE IF NOT EXISTS channel_assets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path TEXT NOT NULL UNIQUE,
			channel_id INTEGER NOT NULL DEFAULT 0,
			filename TEXT NOT NULL,
			asset_type TEXT NOT NULL CHECK (asset_type IN ('bumper', 'commercial', 'promo', 'station_id')),
			tag TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL DEFAULT 'global',
			duration_seconds INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			media_metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);

		CREATE INDEX IF NOT EXISTS idx_channel_assets_type ON channel_assets(asset_type);
		CREATE INDEX IF NOT EXISTS idx_channel_assets_enabled ON channel_assets(enabled);

		CREATE TABLE IF NOT EXISTS scheduled_programs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL DEFAULT 'movie' CHECK (media_type IN ('movie', 'tv', 'asset')),
			episode_path TEXT NOT NULL DEFAULT '',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_scheduled_programs_channel_time ON scheduled_programs(channel_id, start_time);

		CREATE TABLE IF NOT EXISTS channel_schedule_state (
			channel_id INTEGER PRIMARY KEY,
			next_media_index INTEGER NOT NULL DEFAULT 0,
			last_media_id INTEGER NOT NULL DEFAULT 0,
			episode_indexes_json TEXT NOT NULL DEFAULT '{}',
			last_schedule_end_time TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "rating", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "studio", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "actors_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "directors_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "writers_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "countries_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "languages_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "tags_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "media_metadata_json", "TEXT NOT NULL DEFAULT '{}'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "last_seen_at", "TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "movies", "missing_since", "TEXT"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE movies SET last_seen_at = updated_at WHERE last_seen_at = '1970-01-01T00:00:00Z'`); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "episode_paths_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "episode_metadata_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "metadata_version", "INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "studio", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "actors_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "directors_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "writers_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "countries_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "languages_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "series_status", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "tags_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "last_seen_at", "TEXT NOT NULL DEFAULT '1970-01-01T00:00:00Z'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "tv_shows", "missing_since", "TEXT"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE tv_shows SET last_seen_at = updated_at WHERE last_seen_at = '1970-01-01T00:00:00Z'`); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "sort_order", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "airing", "INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "channel_bug_enabled", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "channel_bug_asset_path", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "channel_bug_position", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "channel_bug_width", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "channel_bug_opacity", "REAL NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "channel_bug_margin_x", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "channel_bug_margin_y", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "rating_overlay_enabled", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "rating_overlay_position", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "rating_overlay_duration_seconds", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "now_playing_enabled", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "now_playing_position", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "now_playing_duration_seconds", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "scheduled_programs", "episode_path", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_schedule_state", "episode_indexes_json", "TEXT NOT NULL DEFAULT '{}'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_rules", "media_type", "TEXT NOT NULL DEFAULT '' CHECK (media_type IN ('', 'movie', 'tv'))"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_programming_blocks", "placement", "TEXT NOT NULL DEFAULT 'timeline'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_programming_blocks", "collection_id", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_programming_blocks", "schedule_mode", "TEXT NOT NULL DEFAULT 'weekly'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_programming_blocks", "exact_date", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_programming_blocks", "excluded_dates_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_programming_blocks", "playback_mode", "TEXT NOT NULL DEFAULT 'shuffle'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channels", "description", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "collections", "artwork_path", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "collections", "artwork_mode", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "channel_assets", "channel_id", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_channel_assets_channel_id ON channel_assets(channel_id)`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE channels SET sort_order = id WHERE sort_order = 0`); err != nil {
		return err
	}
	if err := s.ensureScheduledProgramMediaType(ctx); err != nil {
		return err
	}
	return s.ensureScheduledProgramAssetType(ctx)
}

func (s *Store) ensureScheduledProgramMediaType(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(ctx, "PRAGMA table_info(scheduled_programs)")
	if err != nil {
		return err
	}
	hasMediaType := false
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == "media_type" {
			hasMediaType = true
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if hasMediaType {
		return s.recoverScheduledProgramsOld(ctx, conn)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")

	_, err = conn.ExecContext(ctx, `
		ALTER TABLE scheduled_programs RENAME TO scheduled_programs_old;
		CREATE TABLE scheduled_programs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL DEFAULT 'movie' CHECK (media_type IN ('movie', 'tv', 'asset')),
			episode_path TEXT NOT NULL DEFAULT '',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		);
		INSERT INTO scheduled_programs (id, channel_id, media_id, media_type, episode_path, start_time, end_time)
			SELECT id, channel_id, media_id, 'movie', '', start_time, end_time FROM scheduled_programs_old;
		DROP TABLE scheduled_programs_old;
		CREATE INDEX IF NOT EXISTS idx_scheduled_programs_channel_time ON scheduled_programs(channel_id, start_time);
	`)
	return err
}

func (s *Store) ensureScheduledProgramAssetType(ctx context.Context) error {
	var sqlText string
	err := s.db.QueryRowContext(ctx, `
		SELECT sql
		FROM sqlite_master
		WHERE type = 'table' AND name = 'scheduled_programs'
	`).Scan(&sqlText)
	if err != nil {
		return err
	}
	if strings.Contains(sqlText, "'asset'") {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer s.db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")

	_, err = s.db.ExecContext(ctx, `
		ALTER TABLE scheduled_programs RENAME TO scheduled_programs_no_asset;
		CREATE TABLE scheduled_programs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL,
			media_id INTEGER NOT NULL,
			media_type TEXT NOT NULL DEFAULT 'movie' CHECK (media_type IN ('movie', 'tv', 'asset')),
			episode_path TEXT NOT NULL DEFAULT '',
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		);
		INSERT INTO scheduled_programs (id, channel_id, media_id, media_type, episode_path, start_time, end_time)
			SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time FROM scheduled_programs_no_asset;
		DROP TABLE scheduled_programs_no_asset;
		CREATE INDEX IF NOT EXISTS idx_scheduled_programs_channel_time ON scheduled_programs(channel_id, start_time);
	`)
	return err
}

func (s *Store) recoverScheduledProgramsOld(ctx context.Context, conn *sql.Conn) error {
	var name string
	err := conn.QueryRowContext(ctx, `
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name = 'scheduled_programs_old'
	`).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")

	_, err = conn.ExecContext(ctx, `
		INSERT OR IGNORE INTO scheduled_programs (id, channel_id, media_id, media_type, episode_path, start_time, end_time)
			SELECT id, channel_id, media_id, 'movie', '', start_time, end_time FROM scheduled_programs_old;
		DROP TABLE scheduled_programs_old;
		CREATE INDEX IF NOT EXISTS idx_scheduled_programs_channel_time ON scheduled_programs(channel_id, start_time);
	`)
	return err
}

func (s *Store) addColumnIfMissing(ctx context.Context, table string, column string, definition string) error {
	rows, err := s.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func (s *Store) UpsertMovie(ctx context.Context, record models.MovieRecord) (models.MovieRecord, error) {
	genresJSON, err := json.Marshal(record.Genres)
	if err != nil {
		return models.MovieRecord{}, err
	}
	actorsJSON, err := json.Marshal(record.Actors)
	if err != nil {
		return models.MovieRecord{}, err
	}
	directorsJSON, err := json.Marshal(record.Directors)
	if err != nil {
		return models.MovieRecord{}, err
	}
	writersJSON, err := json.Marshal(record.Writers)
	if err != nil {
		return models.MovieRecord{}, err
	}
	countriesJSON, err := json.Marshal(record.Countries)
	if err != nil {
		return models.MovieRecord{}, err
	}
	languagesJSON, err := json.Marshal(record.Languages)
	if err != nil {
		return models.MovieRecord{}, err
	}
	tagsJSON, err := json.Marshal(record.Tags)
	if err != nil {
		return models.MovieRecord{}, err
	}
	mediaMetadataJSON, err := json.Marshal(record.MediaMetadata)
	if err != nil {
		return models.MovieRecord{}, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO movies (
			file_path,
			filename,
			local_title,
			local_year,
			tmdb_id,
			title,
			original_title,
			year,
			release_date,
			overview,
			poster_path,
			backdrop_path,
			poster_local_path,
			backdrop_local_path,
			runtime_minutes,
			genres_json,
			vote_average,
			rating,
			studio,
			actors_json,
			directors_json,
			writers_json,
			countries_json,
			languages_json,
			tags_json,
			media_metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			filename = excluded.filename,
			local_title = excluded.local_title,
			local_year = excluded.local_year,
			tmdb_id = excluded.tmdb_id,
			title = excluded.title,
			original_title = excluded.original_title,
			year = excluded.year,
			release_date = excluded.release_date,
			overview = excluded.overview,
			poster_path = excluded.poster_path,
			backdrop_path = excluded.backdrop_path,
			poster_local_path = excluded.poster_local_path,
			backdrop_local_path = excluded.backdrop_local_path,
			runtime_minutes = excluded.runtime_minutes,
			genres_json = excluded.genres_json,
			vote_average = excluded.vote_average,
			rating = excluded.rating,
			studio = excluded.studio,
			actors_json = excluded.actors_json,
			directors_json = excluded.directors_json,
			writers_json = excluded.writers_json,
			countries_json = excluded.countries_json,
			languages_json = excluded.languages_json,
			tags_json = excluded.tags_json,
			media_metadata_json = excluded.media_metadata_json,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	`, record.FilePath, record.Filename, record.LocalTitle, record.LocalYear, record.TMDBID, record.Title,
		record.OriginalTitle, record.Year, record.ReleaseDate, record.Overview, record.PosterPath,
		record.BackdropPath, record.PosterLocalPath, record.BackdropLocalPath, record.RuntimeMinutes,
		string(genresJSON), record.VoteAverage, record.Rating, record.Studio,
		string(actorsJSON), string(directorsJSON), string(writersJSON), string(countriesJSON),
		string(languagesJSON), string(tagsJSON), string(mediaMetadataJSON))
	if err != nil {
		return models.MovieRecord{}, err
	}

	return s.MovieByFilePath(ctx, record.FilePath)
}

func (s *Store) Movies(ctx context.Context) ([]models.MovieRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, file_path, filename, local_title, local_year, tmdb_id, title, original_title,
			year, release_date, overview, poster_path, backdrop_path, poster_local_path,
			backdrop_local_path, runtime_minutes, genres_json, vote_average, rating, studio,
			actors_json, directors_json, writers_json, countries_json, languages_json, tags_json,
			media_metadata_json, last_seen_at, missing_since, created_at, updated_at
		FROM movies
		ORDER BY title COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movies := []models.MovieRecord{}
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, err
		}
		movies = append(movies, movie)
	}

	return movies, rows.Err()
}

func (s *Store) MovieByFilePath(ctx context.Context, filePath string) (models.MovieRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, file_path, filename, local_title, local_year, tmdb_id, title, original_title,
			year, release_date, overview, poster_path, backdrop_path, poster_local_path,
			backdrop_local_path, runtime_minutes, genres_json, vote_average, rating, studio,
			actors_json, directors_json, writers_json, countries_json, languages_json, tags_json,
			media_metadata_json, last_seen_at, missing_since, created_at, updated_at
		FROM movies
		WHERE file_path = ?
	`, filePath)

	return scanMovie(row)
}

func (s *Store) MovieByID(ctx context.Context, id int64) (models.MovieRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, file_path, filename, local_title, local_year, tmdb_id, title, original_title,
			year, release_date, overview, poster_path, backdrop_path, poster_local_path,
			backdrop_local_path, runtime_minutes, genres_json, vote_average, rating, studio,
			actors_json, directors_json, writers_json, countries_json, languages_json, tags_json,
			media_metadata_json, last_seen_at, missing_since, created_at, updated_at
		FROM movies
		WHERE id = ?
	`, id)

	return scanMovie(row)
}

// RelocateMovie changes only the location-specific fields of an existing
// record. Keeping the row ID preserves collection and scheduling references.
func (s *Store) RelocateMovie(ctx context.Context, id int64, filePath, filename, localTitle string, localYear int, mediaMetadata models.MediaMetadata) (models.MovieRecord, error) {
	metadataJSON, err := json.Marshal(mediaMetadata)
	if err != nil {
		return models.MovieRecord{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE movies SET
			file_path = ?, filename = ?, local_title = ?, local_year = ?,
			media_metadata_json = ?, missing_since = NULL,
			last_seen_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?
	`, filePath, filename, localTitle, localYear, string(metadataJSON), id)
	if err != nil {
		return models.MovieRecord{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return models.MovieRecord{}, err
	}
	if rows == 0 {
		return models.MovieRecord{}, sql.ErrNoRows
	}
	return s.MovieByID(ctx, id)
}

// MergeMovieRecords transfers references from a stale row to the active row,
// then removes the stale database row. It never touches media files.
func (s *Store) MergeMovieRecords(ctx context.Context, staleID, activeID int64) error {
	return s.mergeMediaRecords(ctx, "movie", "movies", staleID, activeID, nil)
}

func (s *Store) DeleteMovie(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM scheduled_programs WHERE media_type = 'movie' AND media_id = ?`, id); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM movies WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ReconcileMoviePresence updates database presence state for roots that were
// scanned successfully. It never changes or removes anything on disk.
func (s *Store) ReconcileMoviePresence(ctx context.Context, roots []string, discoveredPaths []string, observedAt time.Time) error {
	if len(roots) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(discoveredPaths))
	for _, path := range discoveredPaths {
		seen[filepath.Clean(path)] = struct{}{}
	}
	movies, err := s.Movies(ctx)
	if err != nil {
		return err
	}
	timestamp := observedAt.UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, movie := range movies {
		if !pathWithinRoots(movie.FilePath, roots) {
			continue
		}
		if _, ok := seen[filepath.Clean(movie.FilePath)]; ok {
			if _, err := tx.ExecContext(ctx, `UPDATE movies SET last_seen_at = ?, missing_since = NULL WHERE id = ?`, timestamp, movie.ID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE movies SET missing_since = COALESCE(missing_since, ?) WHERE id = ?`, timestamp, movie.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// MarkMoviesOutsideRootsMissing retires records from sources that are no
// longer configured. It is intended for settings reconciliation, where roots
// is the complete authoritative source list.
func (s *Store) MarkMoviesOutsideRootsMissing(ctx context.Context, roots []string, observedAt time.Time) error {
	movies, err := s.Movies(ctx)
	if err != nil {
		return err
	}
	timestamp := observedAt.UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, movie := range movies {
		if pathWithinRoots(movie.FilePath, roots) {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE movies SET missing_since = COALESCE(missing_since, ?) WHERE id = ?`, timestamp, movie.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) UpsertTVShow(ctx context.Context, record models.TVShowRecord) (models.TVShowRecord, error) {
	genresJSON, err := json.Marshal(record.Genres)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	episodePathsJSON, err := json.Marshal(record.EpisodePaths)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	episodeMetadataJSON, err := marshalTVEpisodeMetadata(record.EpisodeMetadata)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	actorsJSON, err := json.Marshal(record.Actors)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	directorsJSON, err := json.Marshal(record.Directors)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	writersJSON, err := json.Marshal(record.Writers)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	countriesJSON, err := json.Marshal(record.Countries)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	languagesJSON, err := json.Marshal(record.Languages)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	tagsJSON, err := json.Marshal(record.Tags)
	if err != nil {
		return models.TVShowRecord{}, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO tv_shows (
			path,
			local_title,
			local_year,
			local_episode_count,
			episode_paths_json,
			episode_metadata_json,
			tmdb_id,
			metadata_version,
			name,
			original_name,
			year,
			first_air_date,
			overview,
			poster_path,
			backdrop_path,
			poster_local_path,
			backdrop_local_path,
			genres_json,
			rating,
			seasons,
			episodes,
			network,
			studio,
			actors_json,
			directors_json,
			writers_json,
			countries_json,
			languages_json,
			series_status,
			tags_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			local_title = excluded.local_title,
			local_year = excluded.local_year,
			local_episode_count = excluded.local_episode_count,
			episode_paths_json = excluded.episode_paths_json,
			episode_metadata_json = excluded.episode_metadata_json,
			tmdb_id = excluded.tmdb_id,
			metadata_version = excluded.metadata_version,
			name = excluded.name,
			original_name = excluded.original_name,
			year = excluded.year,
			first_air_date = excluded.first_air_date,
			overview = excluded.overview,
			poster_path = excluded.poster_path,
			backdrop_path = excluded.backdrop_path,
			poster_local_path = excluded.poster_local_path,
			backdrop_local_path = excluded.backdrop_local_path,
			genres_json = excluded.genres_json,
			rating = excluded.rating,
			seasons = excluded.seasons,
			episodes = excluded.episodes,
			network = excluded.network,
			studio = excluded.studio,
			actors_json = excluded.actors_json,
			directors_json = excluded.directors_json,
			writers_json = excluded.writers_json,
			countries_json = excluded.countries_json,
			languages_json = excluded.languages_json,
			series_status = excluded.series_status,
			tags_json = excluded.tags_json,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	`, record.Path, record.LocalTitle, record.LocalYear, record.LocalEpisodeCount, string(episodePathsJSON), string(episodeMetadataJSON), record.TMDBID, record.MetadataVersion,
		record.Name, record.OriginalName, record.Year, record.FirstAirDate, record.Overview,
		record.PosterPath, record.BackdropPath, record.PosterLocalPath, record.BackdropLocalPath,
		string(genresJSON), record.Rating, record.Seasons, record.Episodes, record.Network,
		record.Studio, string(actorsJSON), string(directorsJSON), string(writersJSON), string(countriesJSON),
		string(languagesJSON), record.SeriesStatus, string(tagsJSON))
	if err != nil {
		return models.TVShowRecord{}, err
	}

	return s.TVShowByPath(ctx, record.Path)
}

func (s *Store) TVShows(ctx context.Context) ([]models.TVShowRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, path, local_title, local_year, local_episode_count, episode_paths_json, episode_metadata_json, tmdb_id, metadata_version, name, original_name,
			year, first_air_date, overview, poster_path, backdrop_path, poster_local_path,
			backdrop_local_path, genres_json, rating, seasons, episodes, network, studio,
			actors_json, directors_json, writers_json, countries_json, languages_json, series_status,
			tags_json, last_seen_at, missing_since, created_at, updated_at
		FROM tv_shows
		ORDER BY name COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shows := []models.TVShowRecord{}
	for rows.Next() {
		show, err := scanTVShow(rows)
		if err != nil {
			return nil, err
		}
		shows = append(shows, show)
	}
	return shows, rows.Err()
}

func (s *Store) TVShowByPath(ctx context.Context, path string) (models.TVShowRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, path, local_title, local_year, local_episode_count, episode_paths_json, episode_metadata_json, tmdb_id, metadata_version, name, original_name,
			year, first_air_date, overview, poster_path, backdrop_path, poster_local_path,
			backdrop_local_path, genres_json, rating, seasons, episodes, network, studio,
			actors_json, directors_json, writers_json, countries_json, languages_json, series_status,
			tags_json, last_seen_at, missing_since, created_at, updated_at
		FROM tv_shows
		WHERE path = ?
	`, path)
	return scanTVShow(row)
}

func (s *Store) TVShowByID(ctx context.Context, id int64) (models.TVShowRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, path, local_title, local_year, local_episode_count, episode_paths_json, episode_metadata_json, tmdb_id, metadata_version, name, original_name,
			year, first_air_date, overview, poster_path, backdrop_path, poster_local_path,
			backdrop_local_path, genres_json, rating, seasons, episodes, network, studio,
			actors_json, directors_json, writers_json, countries_json, languages_json, series_status,
			tags_json, last_seen_at, missing_since, created_at, updated_at
		FROM tv_shows
		WHERE id = ?
	`, id)
	return scanTVShow(row)
}

func (s *Store) DeleteTVShow(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM scheduled_programs WHERE media_type = 'tv' AND media_id = ?`, id); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM tv_shows WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RelocateTVShow updates a show's folder and episode locations in place so
// references to the show's database ID remain valid after a source move.
func (s *Store) RelocateTVShow(ctx context.Context, id int64, path, localTitle string, localYear int, episodePaths []string, episodes []models.TVEpisode) (models.TVShowRecord, error) {
	episodePathsJSON, err := json.Marshal(episodePaths)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	episodesJSON, err := marshalTVEpisodeMetadata(episodes)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE tv_shows SET
			path = ?, local_title = ?, local_year = ?, local_episode_count = ?,
			episode_paths_json = ?, episode_metadata_json = ?, missing_since = NULL,
			last_seen_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?
	`, path, localTitle, localYear, len(episodePaths), string(episodePathsJSON), string(episodesJSON), id)
	if err != nil {
		return models.TVShowRecord{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return models.TVShowRecord{}, err
	}
	if rows == 0 {
		return models.TVShowRecord{}, sql.ErrNoRows
	}
	return s.TVShowByID(ctx, id)
}

// MergeTVShowRecords consolidates duplicate show rows and rewrites scheduled
// episode paths using the supplied old-to-new path map.
func (s *Store) MergeTVShowRecords(ctx context.Context, staleID, activeID int64, episodePaths map[string]string) error {
	return s.mergeMediaRecords(ctx, "tv", "tv_shows", staleID, activeID, episodePaths)
}

func (s *Store) mergeMediaRecords(ctx context.Context, mediaType, table string, staleID, activeID int64, episodePaths map[string]string) error {
	if staleID == activeID {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, referenceTable := range []string{"collection_media", "collection_excluded_media", "collection_artwork_media"} {
		columns := "collection_id, media_type, media_id"
		selectColumns := "collection_id, media_type, ?"
		if referenceTable != "collection_excluded_media" {
			columns += ", sort_order"
			selectColumns += ", sort_order"
		}
		query := fmt.Sprintf(`INSERT OR IGNORE INTO %s (%s) SELECT %s FROM %s WHERE media_type = ? AND media_id = ?`, referenceTable, columns, selectColumns, referenceTable)
		if _, err := tx.ExecContext(ctx, query, activeID, mediaType, staleID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE media_type = ? AND media_id = ?`, referenceTable), mediaType, staleID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE channel_programming_block_media SET media_id = ? WHERE media_type = ? AND media_id = ?`, activeID, mediaType, staleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE scheduled_programs SET media_id = ? WHERE media_type = ? AND media_id = ?`, activeID, mediaType, staleID); err != nil {
		return err
	}
	for oldPath, newPath := range episodePaths {
		if _, err := tx.ExecContext(ctx, `UPDATE scheduled_programs SET episode_path = ? WHERE media_type = 'tv' AND media_id = ? AND episode_path = ?`, newPath, activeID, oldPath); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE channel_schedule_state SET last_media_id = ?
		WHERE last_media_id = ? AND channel_id IN (SELECT id FROM channels WHERE content_type = ?)
	`, activeID, staleID, mediaType); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, table), staleID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// ReconcileTVShowPresence updates database presence state for roots that were
// scanned successfully. It never changes or removes anything on disk.
func (s *Store) ReconcileTVShowPresence(ctx context.Context, roots []string, discoveredPaths []string, observedAt time.Time) error {
	if len(roots) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(discoveredPaths))
	for _, path := range discoveredPaths {
		seen[filepath.Clean(path)] = struct{}{}
	}
	shows, err := s.TVShows(ctx)
	if err != nil {
		return err
	}
	timestamp := observedAt.UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, show := range shows {
		if !pathWithinRoots(show.Path, roots) {
			continue
		}
		if _, ok := seen[filepath.Clean(show.Path)]; ok {
			if _, err := tx.ExecContext(ctx, `UPDATE tv_shows SET last_seen_at = ?, missing_since = NULL WHERE id = ?`, timestamp, show.ID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tv_shows SET missing_since = COALESCE(missing_since, ?) WHERE id = ?`, timestamp, show.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// MarkTVShowsOutsideRootsMissing is the TV equivalent of
// MarkMoviesOutsideRootsMissing.
func (s *Store) MarkTVShowsOutsideRootsMissing(ctx context.Context, roots []string, observedAt time.Time) error {
	shows, err := s.TVShows(ctx)
	if err != nil {
		return err
	}
	timestamp := observedAt.UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, show := range shows {
		if pathWithinRoots(show.Path, roots) {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tv_shows SET missing_since = COALESCE(missing_since, ?) WHERE id = ?`, timestamp, show.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func pathWithinRoots(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (s *Store) MetadataFacets(ctx context.Context) (models.MetadataFacets, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT title, year, release_date, runtime_minutes, genres_json, vote_average, rating, studio,
			actors_json, directors_json, writers_json, countries_json, languages_json, tags_json, created_at
		FROM movies
		WHERE missing_since IS NULL
	`)
	if err != nil {
		return models.MetadataFacets{}, err
	}
	defer rows.Close()

	titles := map[string]struct{}{}
	genres := map[string]struct{}{}
	years := map[int]struct{}{}
	runtimes := map[int]struct{}{}
	voteAverages := map[float64]struct{}{}
	ratings := map[string]struct{}{}
	studios := map[string]struct{}{}
	networks := map[string]struct{}{}
	actors := map[string]struct{}{}
	directors := map[string]struct{}{}
	writers := map[string]struct{}{}
	countries := map[string]struct{}{}
	languages := map[string]struct{}{}
	seriesStatuses := map[string]struct{}{}
	tags := map[string]struct{}{}
	releaseDates := map[string]struct{}{}
	firstAirDates := map[string]struct{}{}
	dateAddedDates := map[string]struct{}{}

	for rows.Next() {
		var title string
		var year int
		var releaseDate string
		var runtime int
		var genresJSON string
		var voteAverage float64
		var rating string
		var studio string
		var actorsJSON string
		var directorsJSON string
		var writersJSON string
		var countriesJSON string
		var languagesJSON string
		var tagsJSON string
		var createdAt string
		if err := rows.Scan(
			&title,
			&year,
			&releaseDate,
			&runtime,
			&genresJSON,
			&voteAverage,
			&rating,
			&studio,
			&actorsJSON,
			&directorsJSON,
			&writersJSON,
			&countriesJSON,
			&languagesJSON,
			&tagsJSON,
			&createdAt,
		); err != nil {
			return models.MetadataFacets{}, err
		}

		if title != "" {
			titles[title] = struct{}{}
		}
		if year > 0 {
			years[year] = struct{}{}
		}
		if runtime > 0 {
			runtimes[runtime] = struct{}{}
		}
		if voteAverage > 0 {
			voteAverages[voteAverage] = struct{}{}
		}
		if rating != "" {
			ratings[rating] = struct{}{}
		}
		if studio != "" {
			studios[studio] = struct{}{}
		}
		if releaseDate != "" {
			releaseDates[releaseDate] = struct{}{}
		}
		if addedDate := datePart(createdAt); addedDate != "" {
			dateAddedDates[addedDate] = struct{}{}
		}

		var movieGenres []string
		if err := json.Unmarshal([]byte(genresJSON), &movieGenres); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode genres: %w", err)
		}
		for _, genre := range movieGenres {
			if genre != "" {
				genres[genre] = struct{}{}
			}
		}
		if err := collectStringFacetJSON(actorsJSON, actors); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode movie actors: %w", err)
		}
		if err := collectStringFacetJSON(directorsJSON, directors); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode movie directors: %w", err)
		}
		if err := collectStringFacetJSON(writersJSON, writers); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode movie writers: %w", err)
		}
		if err := collectStringFacetJSON(countriesJSON, countries); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode movie countries: %w", err)
		}
		if err := collectStringFacetJSON(languagesJSON, languages); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode movie languages: %w", err)
		}
		if err := collectStringFacetJSON(tagsJSON, tags); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode movie tags: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return models.MetadataFacets{}, err
	}

	networkRows, err := s.db.QueryContext(ctx, `
		SELECT name, year, genres_json, rating, network, studio, actors_json, directors_json, writers_json,
			countries_json, languages_json, series_status, tags_json, first_air_date, created_at, episode_metadata_json
		FROM tv_shows
		WHERE missing_since IS NULL
	`)
	if err != nil {
		return models.MetadataFacets{}, err
	}
	defer networkRows.Close()
	for networkRows.Next() {
		var name string
		var year int
		var genresJSON string
		var rating string
		var network string
		var studio string
		var actorsJSON string
		var directorsJSON string
		var writersJSON string
		var countriesJSON string
		var languagesJSON string
		var seriesStatus string
		var tagsJSON string
		var firstAirDate string
		var createdAt string
		var episodeMetadataJSON string
		if err := networkRows.Scan(
			&name,
			&year,
			&genresJSON,
			&rating,
			&network,
			&studio,
			&actorsJSON,
			&directorsJSON,
			&writersJSON,
			&countriesJSON,
			&languagesJSON,
			&seriesStatus,
			&tagsJSON,
			&firstAirDate,
			&createdAt,
			&episodeMetadataJSON,
		); err != nil {
			return models.MetadataFacets{}, err
		}
		if name != "" {
			titles[name] = struct{}{}
		}
		if year > 0 {
			years[year] = struct{}{}
		}
		if rating != "" {
			ratings[rating] = struct{}{}
		}
		if network != "" {
			networks[network] = struct{}{}
		}
		if studio != "" {
			studios[studio] = struct{}{}
		}
		if seriesStatus != "" {
			seriesStatuses[seriesStatus] = struct{}{}
		}
		if firstAirDate != "" {
			firstAirDates[firstAirDate] = struct{}{}
		}
		if addedDate := datePart(createdAt); addedDate != "" {
			dateAddedDates[addedDate] = struct{}{}
		}
		var showGenres []string
		if err := json.Unmarshal([]byte(genresJSON), &showGenres); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV genres: %w", err)
		}
		for _, genre := range showGenres {
			if genre != "" {
				genres[genre] = struct{}{}
			}
		}
		var episodes []models.TVEpisode
		if err := json.Unmarshal([]byte(episodeMetadataJSON), &episodes); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV episode metadata: %w", err)
		}
		for _, episode := range episodes {
			if episode.RuntimeMinutes > 0 {
				runtimes[episode.RuntimeMinutes] = struct{}{}
			}
		}
		if err := collectStringFacetJSON(actorsJSON, actors); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV actors: %w", err)
		}
		if err := collectStringFacetJSON(directorsJSON, directors); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV directors: %w", err)
		}
		if err := collectStringFacetJSON(writersJSON, writers); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV writers: %w", err)
		}
		if err := collectStringFacetJSON(countriesJSON, countries); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV countries: %w", err)
		}
		if err := collectStringFacetJSON(languagesJSON, languages); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV languages: %w", err)
		}
		if err := collectStringFacetJSON(tagsJSON, tags); err != nil {
			return models.MetadataFacets{}, fmt.Errorf("decode TV tags: %w", err)
		}
	}
	if err := networkRows.Err(); err != nil {
		return models.MetadataFacets{}, err
	}

	return models.MetadataFacets{
		Titles:         sortedStrings(titles),
		Genres:         sortedStrings(genres),
		ReleaseYears:   expandedYearRange(years),
		VoteAverages:   sortedFloats(voteAverages),
		Runtimes:       sortedInts(runtimes),
		Ratings:        sortedStrings(ratings),
		Studios:        sortedStrings(studios),
		Networks:       sortedStrings(networks),
		Actors:         sortedStrings(actors),
		Directors:      sortedStrings(directors),
		Writers:        sortedStrings(writers),
		Countries:      sortedStrings(countries),
		Languages:      sortedStrings(languages),
		SeriesStatuses: sortedStrings(seriesStatuses),
		Tags:           sortedStrings(tags),
		ReleaseDates:   sortedStrings(releaseDates),
		FirstAirDates:  sortedStrings(firstAirDates),
		DateAddedDates: sortedStrings(dateAddedDates),
	}, nil
}

func collectStringFacetJSON(encoded string, values map[string]struct{}) error {
	var decoded []string
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		return err
	}
	for _, value := range decoded {
		if value != "" {
			values[value] = struct{}{}
		}
	}
	return nil
}

func (s *Store) SaveChannel(ctx context.Context, channel models.Channel) (models.Channel, error) {
	if channel.ContentType == "" {
		channel.ContentType = "movie"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Channel{}, err
	}
	defer tx.Rollback()

	if channel.ID == 0 {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO channels (
				name, description, logo_path, content_type, sort_order,
				channel_bug_enabled, channel_bug_asset_path, channel_bug_position, channel_bug_width,
				channel_bug_opacity, channel_bug_margin_x, channel_bug_margin_y,
				rating_overlay_enabled, rating_overlay_position, rating_overlay_duration_seconds,
				now_playing_enabled, now_playing_position, now_playing_duration_seconds
			)
			VALUES (?, ?, ?, ?, COALESCE((SELECT MAX(sort_order) FROM channels), 0) + 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			channel.Name, channel.Description, channel.LogoPath, channel.ContentType,
			channel.Presentation.ChannelBugEnabled,
			channel.Presentation.ChannelBugAssetPath,
			channel.Presentation.ChannelBugPosition,
			channel.Presentation.ChannelBugWidth,
			channel.Presentation.ChannelBugOpacity,
			channel.Presentation.ChannelBugMarginX,
			channel.Presentation.ChannelBugMarginY,
			channel.Presentation.RatingOverlayEnabled,
			channel.Presentation.RatingOverlayPosition,
			channel.Presentation.RatingOverlayDurationSeconds,
			channel.Presentation.NowPlayingEnabled,
			channel.Presentation.NowPlayingPosition,
			channel.Presentation.NowPlayingDurationSeconds,
		)
		if err != nil {
			return models.Channel{}, err
		}
		channel.ID, err = result.LastInsertId()
		if err != nil {
			return models.Channel{}, err
		}
	} else {
		result, err := tx.ExecContext(ctx, `
			UPDATE channels
			SET name = ?,
				description = ?,
				logo_path = ?,
				content_type = ?,
				channel_bug_enabled = ?,
				channel_bug_asset_path = ?,
				channel_bug_position = ?,
				channel_bug_width = ?,
				channel_bug_opacity = ?,
				channel_bug_margin_x = ?,
				channel_bug_margin_y = ?,
				rating_overlay_enabled = ?,
				rating_overlay_position = ?,
				rating_overlay_duration_seconds = ?,
				now_playing_enabled = ?,
				now_playing_position = ?,
				now_playing_duration_seconds = ?,
				updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
			WHERE id = ?
		`,
			channel.Name,
			channel.Description,
			channel.LogoPath,
			channel.ContentType,
			channel.Presentation.ChannelBugEnabled,
			channel.Presentation.ChannelBugAssetPath,
			channel.Presentation.ChannelBugPosition,
			channel.Presentation.ChannelBugWidth,
			channel.Presentation.ChannelBugOpacity,
			channel.Presentation.ChannelBugMarginX,
			channel.Presentation.ChannelBugMarginY,
			channel.Presentation.RatingOverlayEnabled,
			channel.Presentation.RatingOverlayPosition,
			channel.Presentation.RatingOverlayDurationSeconds,
			channel.Presentation.NowPlayingEnabled,
			channel.Presentation.NowPlayingPosition,
			channel.Presentation.NowPlayingDurationSeconds,
			channel.ID,
		)
		if err != nil {
			return models.Channel{}, err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return models.Channel{}, err
		}
		if rows == 0 {
			return models.Channel{}, sql.ErrNoRows
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM channel_rules WHERE channel_id = ?`, channel.ID); err != nil {
		return models.Channel{}, err
	}
	for _, rule := range channel.Rules {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO channel_rules (channel_id, media_type, field, operator, value)
			VALUES (?, ?, ?, ?, ?)
		`, channel.ID, rule.MediaType, rule.Field, rule.Operator, rule.Value); err != nil {
			return models.Channel{}, err
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM channel_programming_blocks WHERE channel_id = ?`, channel.ID); err != nil {
		return models.Channel{}, err
	}
	for _, block := range channel.ProgrammingBlocks {
		if block.Placement == "" {
			block.Placement = models.ProgrammingBlockPlacementTimeline
		}
		if block.ScheduleMode == "" {
			block.ScheduleMode = "weekly"
		}
		if block.PlaybackMode == "" {
			block.PlaybackMode = "shuffle"
		}
		daysJSON, err := json.Marshal(block.Days)
		if err != nil {
			return models.Channel{}, err
		}
		excludedDatesJSON, err := json.Marshal(block.ExcludedDates)
		if err != nil {
			return models.Channel{}, err
		}
		title := strings.TrimSpace(block.Title)
		if title == "" {
			title = strings.TrimSpace(block.Name)
		}
		if title == "" {
			continue
		}
		enabled := 1
		if !block.Enabled {
			enabled = 0
		}
		result, err := tx.ExecContext(ctx, `
			INSERT INTO channel_programming_blocks (
				channel_id, title, start_time, end_time, placement, collection_id, days_json, content_source, rule_set, schedule_mode, exact_date, excluded_dates_json, playback_mode, priority, enabled
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, channel.ID, title, block.StartTime, block.EndTime, block.Placement, block.CollectionID, string(daysJSON), block.ContentSource, block.RuleSet, block.ScheduleMode, block.ExactDate, string(excludedDatesJSON), block.PlaybackMode, block.Priority, enabled)
		if err != nil {
			return models.Channel{}, err
		}
		blockID, err := result.LastInsertId()
		if err != nil {
			return models.Channel{}, err
		}
		for index, item := range block.MediaItems {
			if item.MediaType != "movie" && item.MediaType != "tv" {
				continue
			}
			if item.MediaID <= 0 {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO channel_programming_block_media (block_id, media_type, media_id, sort_order)
				VALUES (?, ?, ?, ?)
			`, blockID, item.MediaType, item.MediaID, index+1); err != nil {
				return models.Channel{}, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return models.Channel{}, err
	}
	return s.Channel(ctx, channel.ID)
}

func (s *Store) Channels(ctx context.Context) ([]models.Channel, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, logo_path, content_type, sort_order, airing,
			channel_bug_enabled, channel_bug_asset_path, channel_bug_position, channel_bug_width,
			channel_bug_opacity, channel_bug_margin_x, channel_bug_margin_y,
			rating_overlay_enabled, rating_overlay_position, rating_overlay_duration_seconds,
			now_playing_enabled, now_playing_position, now_playing_duration_seconds,
			created_at, updated_at
		FROM channels
		ORDER BY sort_order, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := []models.Channel{}
	for rows.Next() {
		channel, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		rules, err := s.ChannelRules(ctx, channel.ID)
		if err != nil {
			return nil, err
		}
		channel.Rules = rules
		blocks, err := s.ProgrammingBlocks(ctx, channel.ID)
		if err != nil {
			return nil, err
		}
		channel.ProgrammingBlocks = blocks
		applyLegacyProgrammingFallback(&channel)
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func (s *Store) Channel(ctx context.Context, id int64) (models.Channel, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, logo_path, content_type, sort_order, airing,
			channel_bug_enabled, channel_bug_asset_path, channel_bug_position, channel_bug_width,
			channel_bug_opacity, channel_bug_margin_x, channel_bug_margin_y,
			rating_overlay_enabled, rating_overlay_position, rating_overlay_duration_seconds,
			now_playing_enabled, now_playing_position, now_playing_duration_seconds,
			created_at, updated_at
		FROM channels
		WHERE id = ?
	`, id)
	channel, err := scanChannel(row)
	if err != nil {
		return models.Channel{}, err
	}
	channel.Rules, err = s.ChannelRules(ctx, channel.ID)
	if err != nil {
		return models.Channel{}, err
	}
	channel.ProgrammingBlocks, err = s.ProgrammingBlocks(ctx, channel.ID)
	if err != nil {
		return models.Channel{}, err
	}
	applyLegacyProgrammingFallback(&channel)
	return channel, nil
}

// applyLegacyProgrammingFallback keeps pre-refactor databases playable while
// routing their old channel rules through the programming-block scheduler.
// Any programming save clears the legacy rules and replaces this synthetic block.
func applyLegacyProgrammingFallback(channel *models.Channel) {
	if len(channel.ProgrammingBlocks) != 0 || len(channel.Rules) == 0 {
		return
	}
	channel.ProgrammingBlocks = []models.ProgrammingBlock{{
		ID: -channel.ID, ChannelID: channel.ID, Title: "Migrated Default Programming", Name: "Migrated Default Programming",
		Placement:     models.ProgrammingBlockPlacementAllDay,
		Days:          []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
		ContentSource: "rules", RuleSet: "Migrated channel rules", Enabled: true,
	}}
}

func (s *Store) ProgrammingBlocks(ctx context.Context, channelID int64) ([]models.ProgrammingBlock, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, title, start_time, end_time, placement, collection_id, days_json, content_source, rule_set, schedule_mode, exact_date, excluded_dates_json, playback_mode, priority, enabled
		FROM channel_programming_blocks
		WHERE channel_id = ?
		ORDER BY priority DESC, start_time, id
	`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := []models.ProgrammingBlock{}
	for rows.Next() {
		var block models.ProgrammingBlock
		var daysJSON string
		var excludedDatesJSON string
		var enabled int
		if err := rows.Scan(&block.ID, &block.ChannelID, &block.Title, &block.StartTime, &block.EndTime, &block.Placement, &block.CollectionID, &daysJSON, &block.ContentSource, &block.RuleSet, &block.ScheduleMode, &block.ExactDate, &excludedDatesJSON, &block.PlaybackMode, &block.Priority, &enabled); err != nil {
			return nil, err
		}
		block.Name = block.Title
		block.Enabled = enabled != 0
		if err := json.Unmarshal([]byte(daysJSON), &block.Days); err != nil {
			return nil, fmt.Errorf("decode programming block days: %w", err)
		}
		if err := json.Unmarshal([]byte(excludedDatesJSON), &block.ExcludedDates); err != nil {
			return nil, fmt.Errorf("decode programming block excluded dates: %w", err)
		}
		media, err := s.ProgrammingBlockMedia(ctx, block.ID)
		if err != nil {
			return nil, err
		}
		block.MediaItems = media
		blocks = append(blocks, block)
	}
	return blocks, rows.Err()
}

func (s *Store) ProgrammingBlockMedia(ctx context.Context, blockID int64) ([]models.ProgrammingBlockMedia, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, block_id, media_type, media_id, sort_order
		FROM channel_programming_block_media
		WHERE block_id = ?
		ORDER BY sort_order, id
	`, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.ProgrammingBlockMedia{}
	for rows.Next() {
		var item models.ProgrammingBlockMedia
		if err := rows.Scan(&item.ID, &item.BlockID, &item.MediaType, &item.MediaID, &item.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ReorderChannels(ctx context.Context, channelIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for index, id := range channelIDs {
		result, err := tx.ExecContext(ctx, `
			UPDATE channels
			SET sort_order = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
			WHERE id = ?
		`, index+1, id)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return sql.ErrNoRows
		}
	}

	return tx.Commit()
}

func (s *Store) SetChannelAiring(ctx context.Context, channelID int64, airing bool) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE channels
		SET airing = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?
	`, airing, channelID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteChannel(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM channels WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ChannelRules(ctx context.Context, channelID int64) ([]models.ChannelRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, media_type, field, operator, value
		FROM channel_rules
		WHERE channel_id = ?
		ORDER BY id
	`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := []models.ChannelRule{}
	for rows.Next() {
		var rule models.ChannelRule
		if err := rows.Scan(&rule.ID, &rule.ChannelID, &rule.MediaType, &rule.Field, &rule.Operator, &rule.Value); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (s *Store) MatchingMovies(ctx context.Context, rules []models.ChannelRule) ([]models.MovieRecord, error) {
	movies, err := s.Movies(ctx)
	if err != nil {
		return nil, err
	}
	if len(rulesForMediaType(rules, "movie")) == 0 {
		return []models.MovieRecord{}, nil
	}
	present := movies[:0]
	for _, movie := range movies {
		if movie.MissingSince == nil {
			present = append(present, movie)
		}
	}
	movies = present

	matches := []models.MovieRecord{}
	for _, movie := range movies {
		if movieMatchesRules(movie, rules) {
			matches = append(matches, movie)
		}
	}
	if len(matches) == 0 {
		for _, movie := range movies {
			if movieMatchesAnyIncludeRule(movie, rules) {
				matches = append(matches, movie)
			}
		}
	}
	return filterMoviesByQuality(matches, rules), nil
}

func (s *Store) MatchingTVShows(ctx context.Context, rules []models.ChannelRule) ([]models.TVShowRecord, error) {
	shows, err := s.TVShows(ctx)
	if err != nil {
		return nil, err
	}
	if len(rulesForMediaType(rules, "tv")) == 0 {
		return []models.TVShowRecord{}, nil
	}
	present := shows[:0]
	for _, show := range shows {
		if show.MissingSince == nil {
			present = append(present, show)
		}
	}
	shows = present
	if allRulesAreTVPoolIncludes(rules) {
		matches := []models.TVShowRecord{}
		for _, show := range shows {
			if tvShowMatchesAnyIncludeRule(show, rules) {
				matches = append(matches, show)
			}
		}
		return filterTVShowsByQuality(matches, rules), nil
	}

	matches := []models.TVShowRecord{}
	for _, show := range shows {
		if tvShowMatchesRules(show, rules) {
			matches = append(matches, show)
		}
	}
	if len(matches) == 0 {
		for _, show := range shows {
			if tvShowMatchesAnyIncludeRule(show, rules) {
				matches = append(matches, show)
			}
		}
	}
	return filterTVShowsByQuality(matches, rules), nil
}

func (s *Store) ReplaceScheduledPrograms(ctx context.Context, channelID int64, programs []models.ScheduledProgram) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM scheduled_programs WHERE channel_id = ?`, channelID); err != nil {
		return err
	}
	for _, program := range programs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO scheduled_programs (channel_id, media_id, media_type, episode_path, start_time, end_time)
			VALUES (?, ?, ?, ?, ?, ?)
		`, channelID, program.MediaID, scheduledMediaType(program.MediaType), program.EpisodePath, formatDBTime(program.StartTime), formatDBTime(program.EndTime)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) AppendScheduledPrograms(ctx context.Context, channelID int64, programs []models.ScheduledProgram) error {
	if len(programs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, program := range programs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO scheduled_programs (channel_id, media_id, media_type, episode_path, start_time, end_time)
			VALUES (?, ?, ?, ?, ?, ?)
		`, channelID, program.MediaID, scheduledMediaType(program.MediaType), program.EpisodePath, formatDBTime(program.StartTime), formatDBTime(program.EndTime)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) UpsertChannelAsset(ctx context.Context, asset models.ChannelAsset) (models.ChannelAsset, error) {
	metadataJSON, err := json.Marshal(asset.MediaMetadata)
	if err != nil {
		return models.ChannelAsset{}, err
	}
	enabled := 1
	if !asset.Enabled {
		enabled = 0
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO channel_assets (file_path, channel_id, filename, asset_type, tag, scope, duration_seconds, enabled, media_metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			channel_id = excluded.channel_id,
			filename = excluded.filename,
			asset_type = excluded.asset_type,
			tag = excluded.tag,
			scope = excluded.scope,
			duration_seconds = excluded.duration_seconds,
			media_metadata_json = excluded.media_metadata_json,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	`, asset.FilePath, asset.ChannelID, asset.Filename, asset.AssetType, asset.Tag, asset.Scope, asset.DurationSeconds, enabled, string(metadataJSON))
	if err != nil {
		return models.ChannelAsset{}, err
	}
	return s.ChannelAssetByPath(ctx, asset.FilePath)
}

func (s *Store) ChannelAssetByPath(ctx context.Context, path string) (models.ChannelAsset, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, channel_id, file_path, filename, asset_type, tag, scope, duration_seconds, enabled, media_metadata_json, created_at, updated_at
		FROM channel_assets
		WHERE file_path = ?
	`, path)
	return scanChannelAsset(row)
}

func (s *Store) ChannelAsset(ctx context.Context, id int64) (models.ChannelAsset, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, channel_id, file_path, filename, asset_type, tag, scope, duration_seconds, enabled, media_metadata_json, created_at, updated_at
		FROM channel_assets
		WHERE id = ?
	`, id)
	return scanChannelAsset(row)
}

func (s *Store) ChannelAssets(ctx context.Context) ([]models.ChannelAsset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, file_path, filename, asset_type, tag, scope, duration_seconds, enabled, media_metadata_json, created_at, updated_at
		FROM channel_assets
		ORDER BY asset_type, tag, filename
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChannelAssets(rows)
}

func (s *Store) EnabledChannelAssets(ctx context.Context) ([]models.ChannelAsset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, file_path, filename, asset_type, tag, scope, duration_seconds, enabled, media_metadata_json, created_at, updated_at
		FROM channel_assets
		WHERE enabled = 1
		ORDER BY asset_type, tag, filename
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChannelAssets(rows)
}

func (s *Store) PruneScheduledProgramsBefore(ctx context.Context, channelID int64, cutoff time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM scheduled_programs
		WHERE channel_id = ? AND end_time <= ?
	`, channelID, formatDBTime(cutoff))
	return err
}

func (s *Store) LatestScheduledProgram(ctx context.Context, channelID int64) (models.ScheduledProgram, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time
		FROM scheduled_programs
		WHERE channel_id = ?
		ORDER BY end_time DESC
		LIMIT 1
	`, channelID)

	program, err := scanScheduledProgramRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ScheduledProgram{}, false, nil
	}
	if err != nil {
		return models.ScheduledProgram{}, false, err
	}
	return program, true, nil
}

func (s *Store) ChannelScheduleState(ctx context.Context, channelID int64) (models.ChannelScheduleState, bool, error) {
	var state models.ChannelScheduleState
	var episodeIndexesJSON string
	var lastEnd string
	var updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT channel_id, next_media_index, last_media_id, episode_indexes_json, last_schedule_end_time, updated_at
		FROM channel_schedule_state
		WHERE channel_id = ?
	`, channelID).Scan(&state.ChannelID, &state.NextMediaIndex, &state.LastMediaID, &episodeIndexesJSON, &lastEnd, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ChannelScheduleState{}, false, nil
	}
	if err != nil {
		return models.ChannelScheduleState{}, false, err
	}
	if episodeIndexesJSON != "" {
		if err := json.Unmarshal([]byte(episodeIndexesJSON), &state.EpisodeIndexes); err != nil {
			return models.ChannelScheduleState{}, false, fmt.Errorf("decode schedule state episode indexes: %w", err)
		}
	}
	if lastEnd != "" {
		state.LastScheduleEndTime, err = time.Parse(time.RFC3339, lastEnd)
		if err != nil {
			return models.ChannelScheduleState{}, false, fmt.Errorf("decode schedule state last_schedule_end_time: %w", err)
		}
	}
	state.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.ChannelScheduleState{}, false, fmt.Errorf("decode schedule state updated_at: %w", err)
	}
	return state, true, nil
}

func (s *Store) SaveChannelScheduleState(ctx context.Context, state models.ChannelScheduleState) error {
	if state.EpisodeIndexes == nil {
		state.EpisodeIndexes = map[int64]int{}
	}
	episodeIndexesJSON, err := json.Marshal(state.EpisodeIndexes)
	if err != nil {
		return fmt.Errorf("encode schedule state episode indexes: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO channel_schedule_state (channel_id, next_media_index, last_media_id, episode_indexes_json, last_schedule_end_time, updated_at)
		VALUES (?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT(channel_id) DO UPDATE SET
			next_media_index = excluded.next_media_index,
			last_media_id = excluded.last_media_id,
			episode_indexes_json = excluded.episode_indexes_json,
			last_schedule_end_time = excluded.last_schedule_end_time,
			updated_at = excluded.updated_at
	`, state.ChannelID, state.NextMediaIndex, state.LastMediaID, string(episodeIndexesJSON), formatDBTime(state.LastScheduleEndTime))
	return err
}

func scheduledMediaType(value string) string {
	switch value {
	case "tv":
		return "tv"
	case "asset":
		return "asset"
	default:
		return "movie"
	}
}

func (s *Store) ScheduledPrograms(ctx context.Context, channelID int64) ([]models.ScheduledProgram, error) {
	return s.scheduledProgramsQuery(ctx, `
		SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time
		FROM scheduled_programs
		WHERE channel_id = ?
		ORDER BY start_time
	`, channelID)
}

func (s *Store) ScheduledProgramsWindow(ctx context.Context, channelID int64, start time.Time, end time.Time) ([]models.ScheduledProgram, error) {
	return s.scheduledProgramsQuery(ctx, `
		SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time
		FROM scheduled_programs
		WHERE channel_id = ? AND end_time > ? AND start_time < ?
		ORDER BY start_time
	`, channelID, formatDBTime(start), formatDBTime(end))
}

// GuideScheduledProgramsWindow loads a guide window for multiple channels in
// one schedule query. Assets remain lightweight timing rows, while repeated
// movie and TV records are hydrated only once per request.
func (s *Store) GuideScheduledProgramsWindow(ctx context.Context, channelIDs []int64, start time.Time, end time.Time) (map[int64][]models.ScheduledProgram, error) {
	programsByChannel := make(map[int64][]models.ScheduledProgram, len(channelIDs))
	if len(channelIDs) == 0 {
		return programsByChannel, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(channelIDs)), ",")
	args := make([]any, 0, len(channelIDs)+2)
	for _, channelID := range channelIDs {
		args = append(args, channelID)
	}
	args = append(args, formatDBTime(start), formatDBTime(end))

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time
		FROM scheduled_programs
		WHERE channel_id IN (`+placeholders+`) AND end_time > ? AND start_time < ?
		ORDER BY channel_id, start_time
	`, args...)
	if err != nil {
		return nil, err
	}

	programs := []models.ScheduledProgram{}
	for rows.Next() {
		program, scanErr := scanScheduledProgramRow(rows)
		if scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		programs = append(programs, program)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	movies := map[int64]models.MovieRecord{}
	shows := map[int64]models.TVShowRecord{}
	for index := range programs {
		program := &programs[index]
		switch program.MediaType {
		case "movie":
			movie, ok := movies[program.MediaID]
			if !ok {
				movie, err = s.MovieByID(ctx, program.MediaID)
				if err != nil {
					return nil, err
				}
				movies[program.MediaID] = movie
			}
			program.Media = movie
		case "tv":
			show, ok := shows[program.MediaID]
			if !ok {
				show, err = s.TVShowByID(ctx, program.MediaID)
				if err != nil {
					return nil, err
				}
				shows[program.MediaID] = show
			}
			program.TVMedia = show
		}
		programsByChannel[program.ChannelID] = append(programsByChannel[program.ChannelID], *program)
	}
	return programsByChannel, nil
}

func (s *Store) CurrentAndNextScheduledPrograms(ctx context.Context, channelID int64, now time.Time) ([]models.ScheduledProgram, error) {
	return s.scheduledProgramsQuery(ctx, `
		SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time
		FROM scheduled_programs
		WHERE channel_id = ? AND end_time > ?
		ORDER BY start_time
		LIMIT 2
	`, channelID, formatDBTime(now))
}

func (s *Store) ViewerSchedule(ctx context.Context, channelID int64, now time.Time) ([]models.ScheduledProgram, error) {
	return s.scheduledProgramsQuery(ctx, `
		WITH anchored AS (
			SELECT COALESCE(MAX(start_time), ?) AS start_time
			FROM scheduled_programs
			WHERE channel_id = ? AND media_type != 'asset' AND start_time <= ?
		), ranked AS (
			SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time,
				SUM(CASE WHEN media_type != 'asset' THEN 1 ELSE 0 END)
					OVER (ORDER BY start_time, id) AS viewer_program_number
			FROM scheduled_programs
			WHERE channel_id = ? AND start_time >= (SELECT start_time FROM anchored)
		)
		SELECT id, channel_id, media_id, media_type, episode_path, start_time, end_time
		FROM ranked
		WHERE viewer_program_number <= 32
		ORDER BY start_time, id
	`, formatDBTime(now), channelID, formatDBTime(now), channelID)
}

func (s *Store) scheduledProgramsQuery(ctx context.Context, query string, args ...any) ([]models.ScheduledProgram, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	programs := []models.ScheduledProgram{}
	for rows.Next() {
		program, err := scanScheduledProgramRow(rows)
		if err != nil {
			return nil, err
		}
		if program.MediaType == "asset" {
			program.Asset, err = s.ChannelAsset(ctx, program.MediaID)
		} else if program.MediaType == "tv" {
			program.TVMedia, err = s.TVShowByID(ctx, program.MediaID)
		} else {
			program.Media, err = s.MovieByID(ctx, program.MediaID)
		}
		if err != nil {
			return nil, err
		}
		programs = append(programs, program)
	}
	return programs, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMovie(row scanner) (models.MovieRecord, error) {
	var record models.MovieRecord
	var genresJSON string
	var actorsJSON string
	var directorsJSON string
	var writersJSON string
	var countriesJSON string
	var languagesJSON string
	var tagsJSON string
	var mediaMetadataJSON string
	var lastSeenAt string
	var missingSince sql.NullString
	var createdAt string
	var updatedAt string

	err := row.Scan(
		&record.ID,
		&record.FilePath,
		&record.Filename,
		&record.LocalTitle,
		&record.LocalYear,
		&record.TMDBID,
		&record.Title,
		&record.OriginalTitle,
		&record.Year,
		&record.ReleaseDate,
		&record.Overview,
		&record.PosterPath,
		&record.BackdropPath,
		&record.PosterLocalPath,
		&record.BackdropLocalPath,
		&record.RuntimeMinutes,
		&genresJSON,
		&record.VoteAverage,
		&record.Rating,
		&record.Studio,
		&actorsJSON,
		&directorsJSON,
		&writersJSON,
		&countriesJSON,
		&languagesJSON,
		&tagsJSON,
		&mediaMetadataJSON,
		&lastSeenAt,
		&missingSince,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return models.MovieRecord{}, err
	}

	if err := json.Unmarshal([]byte(genresJSON), &record.Genres); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode genres: %w", err)
	}
	if err := json.Unmarshal([]byte(actorsJSON), &record.Actors); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode movie actors: %w", err)
	}
	if err := json.Unmarshal([]byte(directorsJSON), &record.Directors); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode movie directors: %w", err)
	}
	if err := json.Unmarshal([]byte(writersJSON), &record.Writers); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode movie writers: %w", err)
	}
	if err := json.Unmarshal([]byte(countriesJSON), &record.Countries); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode movie countries: %w", err)
	}
	if err := json.Unmarshal([]byte(languagesJSON), &record.Languages); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode movie languages: %w", err)
	}
	if err := json.Unmarshal([]byte(tagsJSON), &record.Tags); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode movie tags: %w", err)
	}
	if err := json.Unmarshal([]byte(mediaMetadataJSON), &record.MediaMetadata); err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode media metadata: %w", err)
	}
	record.LastSeenAt, err = time.Parse(time.RFC3339, lastSeenAt)
	if err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode movie last_seen_at: %w", err)
	}
	if missingSince.Valid {
		parsed, parseErr := time.Parse(time.RFC3339, missingSince.String)
		if parseErr != nil {
			return models.MovieRecord{}, fmt.Errorf("decode movie missing_since: %w", parseErr)
		}
		record.MissingSince = &parsed
	}
	record.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode created_at: %w", err)
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.MovieRecord{}, fmt.Errorf("decode updated_at: %w", err)
	}

	return record, nil
}

func scanTVShow(row scanner) (models.TVShowRecord, error) {
	var record models.TVShowRecord
	var genresJSON string
	var episodePathsJSON string
	var episodeMetadataJSON string
	var actorsJSON string
	var directorsJSON string
	var writersJSON string
	var countriesJSON string
	var languagesJSON string
	var tagsJSON string
	var lastSeenAt string
	var missingSince sql.NullString
	var createdAt string
	var updatedAt string

	err := row.Scan(
		&record.ID,
		&record.Path,
		&record.LocalTitle,
		&record.LocalYear,
		&record.LocalEpisodeCount,
		&episodePathsJSON,
		&episodeMetadataJSON,
		&record.TMDBID,
		&record.MetadataVersion,
		&record.Name,
		&record.OriginalName,
		&record.Year,
		&record.FirstAirDate,
		&record.Overview,
		&record.PosterPath,
		&record.BackdropPath,
		&record.PosterLocalPath,
		&record.BackdropLocalPath,
		&genresJSON,
		&record.Rating,
		&record.Seasons,
		&record.Episodes,
		&record.Network,
		&record.Studio,
		&actorsJSON,
		&directorsJSON,
		&writersJSON,
		&countriesJSON,
		&languagesJSON,
		&record.SeriesStatus,
		&tagsJSON,
		&lastSeenAt,
		&missingSince,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return models.TVShowRecord{}, err
	}

	if err := json.Unmarshal([]byte(genresJSON), &record.Genres); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV genres: %w", err)
	}
	if err := json.Unmarshal([]byte(episodePathsJSON), &record.EpisodePaths); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV episode paths: %w", err)
	}
	if err := unmarshalTVEpisodeMetadata(episodeMetadataJSON, &record.EpisodeMetadata); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV episode metadata: %w", err)
	}
	if err := json.Unmarshal([]byte(actorsJSON), &record.Actors); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV actors: %w", err)
	}
	if err := json.Unmarshal([]byte(directorsJSON), &record.Directors); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV directors: %w", err)
	}
	if err := json.Unmarshal([]byte(writersJSON), &record.Writers); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV writers: %w", err)
	}
	if err := json.Unmarshal([]byte(countriesJSON), &record.Countries); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV countries: %w", err)
	}
	if err := json.Unmarshal([]byte(languagesJSON), &record.Languages); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV languages: %w", err)
	}
	if err := json.Unmarshal([]byte(tagsJSON), &record.Tags); err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV tags: %w", err)
	}
	record.LastSeenAt, err = time.Parse(time.RFC3339, lastSeenAt)
	if err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV last_seen_at: %w", err)
	}
	if missingSince.Valid {
		parsed, parseErr := time.Parse(time.RFC3339, missingSince.String)
		if parseErr != nil {
			return models.TVShowRecord{}, fmt.Errorf("decode TV missing_since: %w", parseErr)
		}
		record.MissingSince = &parsed
	}
	record.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV created_at: %w", err)
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.TVShowRecord{}, fmt.Errorf("decode TV updated_at: %w", err)
	}

	return record, nil
}

type storedTVEpisode struct {
	Path               string               `json:"path"`
	Season             int                  `json:"season"`
	Episode            int                  `json:"episode"`
	Name               string               `json:"name"`
	Overview           string               `json:"overview"`
	AirDate            string               `json:"air_date"`
	StillPath          string               `json:"still_path"`
	ThumbnailLocalPath string               `json:"thumbnail_local_path"`
	RuntimeMinutes     int                  `json:"runtime_minutes"`
	MediaMetadata      models.MediaMetadata `json:"media_metadata,omitempty"`
}

func marshalTVEpisodeMetadata(episodes []models.TVEpisode) (string, error) {
	stored := make([]storedTVEpisode, 0, len(episodes))
	for _, episode := range episodes {
		stored = append(stored, storedTVEpisode{
			Path:               episode.Path,
			Season:             episode.Season,
			Episode:            episode.Episode,
			Name:               episode.Name,
			Overview:           episode.Overview,
			AirDate:            episode.AirDate,
			StillPath:          episode.StillPath,
			ThumbnailLocalPath: episode.ThumbnailLocalPath,
			RuntimeMinutes:     episode.RuntimeMinutes,
			MediaMetadata:      episode.MediaMetadata,
		})
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalTVEpisodeMetadata(data string, episodes *[]models.TVEpisode) error {
	var stored []storedTVEpisode
	if err := json.Unmarshal([]byte(data), &stored); err != nil {
		return err
	}
	result := make([]models.TVEpisode, 0, len(stored))
	for _, episode := range stored {
		result = append(result, models.TVEpisode{
			Path:               episode.Path,
			Season:             episode.Season,
			Episode:            episode.Episode,
			Name:               episode.Name,
			Overview:           episode.Overview,
			AirDate:            episode.AirDate,
			StillPath:          episode.StillPath,
			ThumbnailLocalPath: episode.ThumbnailLocalPath,
			RuntimeMinutes:     episode.RuntimeMinutes,
			MediaMetadata:      episode.MediaMetadata,
		})
	}
	*episodes = result
	return nil
}

func scanChannel(row scanner) (models.Channel, error) {
	var channel models.Channel
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&channel.ID,
		&channel.Name,
		&channel.Description,
		&channel.LogoPath,
		&channel.ContentType,
		&channel.SortOrder,
		&channel.Airing,
		&channel.Presentation.ChannelBugEnabled,
		&channel.Presentation.ChannelBugAssetPath,
		&channel.Presentation.ChannelBugPosition,
		&channel.Presentation.ChannelBugWidth,
		&channel.Presentation.ChannelBugOpacity,
		&channel.Presentation.ChannelBugMarginX,
		&channel.Presentation.ChannelBugMarginY,
		&channel.Presentation.RatingOverlayEnabled,
		&channel.Presentation.RatingOverlayPosition,
		&channel.Presentation.RatingOverlayDurationSeconds,
		&channel.Presentation.NowPlayingEnabled,
		&channel.Presentation.NowPlayingPosition,
		&channel.Presentation.NowPlayingDurationSeconds,
		&createdAt,
		&updatedAt,
	); err != nil {
		return models.Channel{}, err
	}

	var err error
	channel.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.Channel{}, fmt.Errorf("decode channel created_at: %w", err)
	}
	channel.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.Channel{}, fmt.Errorf("decode channel updated_at: %w", err)
	}
	return channel, nil
}

func scanScheduledProgramRow(row scanner) (models.ScheduledProgram, error) {
	var program models.ScheduledProgram
	var startTime string
	var endTime string
	if err := row.Scan(
		&program.ID,
		&program.ChannelID,
		&program.MediaID,
		&program.MediaType,
		&program.EpisodePath,
		&startTime,
		&endTime,
	); err != nil {
		return models.ScheduledProgram{}, err
	}

	var err error
	program.StartTime, err = time.Parse(time.RFC3339, startTime)
	if err != nil {
		return models.ScheduledProgram{}, fmt.Errorf("decode scheduled start_time: %w", err)
	}
	program.EndTime, err = time.Parse(time.RFC3339, endTime)
	if err != nil {
		return models.ScheduledProgram{}, fmt.Errorf("decode scheduled end_time: %w", err)
	}
	return program, nil
}

func scanChannelAssets(rows *sql.Rows) ([]models.ChannelAsset, error) {
	assets := []models.ChannelAsset{}
	for rows.Next() {
		asset, err := scanChannelAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func scanChannelAsset(row scanner) (models.ChannelAsset, error) {
	var asset models.ChannelAsset
	var enabled int
	var mediaMetadataJSON string
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&asset.ID,
		&asset.ChannelID,
		&asset.FilePath,
		&asset.Filename,
		&asset.AssetType,
		&asset.Tag,
		&asset.Scope,
		&asset.DurationSeconds,
		&enabled,
		&mediaMetadataJSON,
		&createdAt,
		&updatedAt,
	); err != nil {
		return models.ChannelAsset{}, err
	}
	asset.Enabled = enabled != 0
	if err := json.Unmarshal([]byte(mediaMetadataJSON), &asset.MediaMetadata); err != nil {
		return models.ChannelAsset{}, fmt.Errorf("decode channel asset media metadata: %w", err)
	}
	var err error
	asset.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return models.ChannelAsset{}, fmt.Errorf("decode channel asset created_at: %w", err)
	}
	asset.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return models.ChannelAsset{}, fmt.Errorf("decode channel asset updated_at: %w", err)
	}
	return asset, nil
}

func sortedStrings(values map[string]struct{}) []string {
	sorted := make([]string, 0, len(values))
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)
	return sorted
}

func sortedInts(values map[int]struct{}) []int {
	sorted := make([]int, 0, len(values))
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Ints(sorted)
	return sorted
}

func expandedYearRange(values map[int]struct{}) []int {
	sorted := sortedInts(values)
	if len(sorted) == 0 {
		return sorted
	}

	start := sorted[0] - 1
	stop := sorted[len(sorted)-1] + 1
	years := make([]int, 0, stop-start+1)
	for year := start; year <= stop; year++ {
		years = append(years, year)
	}
	return years
}

func sortedFloats(values map[float64]struct{}) []float64 {
	sorted := make([]float64, 0, len(values))
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Float64s(sorted)
	return sorted
}

func movieMatchesRules(movie models.MovieRecord, rules []models.ChannelRule) bool {
	movieRules := rulesForMediaType(rules, "movie")
	if len(movieRules) == 0 {
		return false
	}
	if !titleRulesMatch(recordTitles(movie.Title, movie.LocalTitle), movieRules) {
		return false
	}
	for _, rule := range movieRules {
		switch rule.Field {
		case "title", "quality":
			continue
		case "genre":
			if !matchStringList(movie.Genres, rule.Operator, rule.Value) {
				return false
			}
		case "year":
			if !matchInt(movie.Year, rule.Operator, rule.Value) {
				return false
			}
		case "runtime":
			if !matchInt(movie.RuntimeMinutes, rule.Operator, rule.Value) {
				return false
			}
		case "rating":
			if !matchString(movie.Rating, rule.Operator, rule.Value) {
				return false
			}
		case "studio":
			if !matchString(movie.Studio, rule.Operator, rule.Value) {
				return false
			}
		case "actor":
			if !matchStringList(movie.Actors, rule.Operator, rule.Value) {
				return false
			}
		case "director":
			if !matchStringList(movie.Directors, rule.Operator, rule.Value) {
				return false
			}
		case "writer":
			if !matchStringList(movie.Writers, rule.Operator, rule.Value) {
				return false
			}
		case "country":
			if !matchStringList(movie.Countries, rule.Operator, rule.Value) {
				return false
			}
		case "language":
			if !matchStringList(movie.Languages, rule.Operator, rule.Value) {
				return false
			}
		case "release_date":
			if !matchDate(movie.ReleaseDate, rule.Operator, rule.Value) {
				return false
			}
		case "date_added":
			if !matchDate(datePart(movie.CreatedAt.Format(time.RFC3339)), rule.Operator, rule.Value) {
				return false
			}
		case "tag":
			if !matchStringList(movie.Tags, rule.Operator, rule.Value) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func tvShowMatchesRules(show models.TVShowRecord, rules []models.ChannelRule) bool {
	tvRules := rulesForMediaType(rules, "tv")
	if len(tvRules) == 0 {
		return false
	}
	if !titleRulesMatch(recordTitles(show.Name, show.LocalTitle), tvRules) {
		return false
	}
	for _, rule := range tvRules {
		switch rule.Field {
		case "title", "quality":
			continue
		case "genre":
			if !matchStringList(show.Genres, rule.Operator, rule.Value) {
				return false
			}
		case "year":
			if !matchInt(show.Year, rule.Operator, rule.Value) {
				return false
			}
		case "runtime":
			if !matchInt(tvShowRuntimeMinutes(show), rule.Operator, rule.Value) {
				return false
			}
		case "rating":
			if !matchString(show.Rating, rule.Operator, rule.Value) {
				return false
			}
		case "first_air_date":
			if !matchDate(show.FirstAirDate, rule.Operator, rule.Value) {
				return false
			}
		case "date_added":
			if !matchDate(datePart(show.CreatedAt.Format(time.RFC3339)), rule.Operator, rule.Value) {
				return false
			}
		case "network":
			if !matchString(show.Network, rule.Operator, rule.Value) {
				return false
			}
		case "studio":
			if !matchString(show.Studio, rule.Operator, rule.Value) {
				return false
			}
		case "actor":
			if !matchStringList(show.Actors, rule.Operator, rule.Value) {
				return false
			}
		case "director":
			if !matchStringList(show.Directors, rule.Operator, rule.Value) {
				return false
			}
		case "writer":
			if !matchStringList(show.Writers, rule.Operator, rule.Value) {
				return false
			}
		case "country":
			if !matchStringList(show.Countries, rule.Operator, rule.Value) {
				return false
			}
		case "language":
			if !matchStringList(show.Languages, rule.Operator, rule.Value) {
				return false
			}
		case "series_status":
			if !matchString(show.SeriesStatus, rule.Operator, rule.Value) {
				return false
			}
		case "tag":
			if !matchStringList(show.Tags, rule.Operator, rule.Value) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func movieMatchesAnyIncludeRule(movie models.MovieRecord, rules []models.ChannelRule) bool {
	for _, rule := range rulesForMediaType(rules, "movie") {
		if !ruleIsInclude(rule) {
			continue
		}
		switch rule.Field {
		case "title":
			if matchAnyString(recordTitles(movie.Title, movie.LocalTitle), rule.Operator, rule.Value) {
				return true
			}
		case "genre":
			if matchStringList(movie.Genres, rule.Operator, rule.Value) {
				return true
			}
		case "year":
			if matchInt(movie.Year, rule.Operator, rule.Value) {
				return true
			}
		case "runtime":
			if matchInt(movie.RuntimeMinutes, rule.Operator, rule.Value) {
				return true
			}
		case "rating":
			if matchString(movie.Rating, rule.Operator, rule.Value) {
				return true
			}
		case "studio":
			if matchString(movie.Studio, rule.Operator, rule.Value) {
				return true
			}
		case "actor":
			if matchStringList(movie.Actors, rule.Operator, rule.Value) {
				return true
			}
		case "director":
			if matchStringList(movie.Directors, rule.Operator, rule.Value) {
				return true
			}
		case "writer":
			if matchStringList(movie.Writers, rule.Operator, rule.Value) {
				return true
			}
		case "country":
			if matchStringList(movie.Countries, rule.Operator, rule.Value) {
				return true
			}
		case "language":
			if matchStringList(movie.Languages, rule.Operator, rule.Value) {
				return true
			}
		case "release_date":
			if matchDate(movie.ReleaseDate, rule.Operator, rule.Value) {
				return true
			}
		case "date_added":
			if matchDate(datePart(movie.CreatedAt.Format(time.RFC3339)), rule.Operator, rule.Value) {
				return true
			}
		case "tag":
			if matchStringList(movie.Tags, rule.Operator, rule.Value) {
				return true
			}
		}
	}
	return false
}

func tvShowRuntimeMinutes(show models.TVShowRecord) int {
	total := 0
	count := 0
	for _, episode := range show.EpisodeMetadata {
		if episode.RuntimeMinutes <= 0 {
			continue
		}
		total += episode.RuntimeMinutes
		count++
	}
	if count == 0 {
		return 0
	}
	return total / count
}

func tvShowMatchesAnyIncludeRule(show models.TVShowRecord, rules []models.ChannelRule) bool {
	for _, rule := range rulesForMediaType(rules, "tv") {
		if !ruleIsInclude(rule) {
			continue
		}
		switch rule.Field {
		case "title":
			if matchAnyString(recordTitles(show.Name, show.LocalTitle), rule.Operator, rule.Value) {
				return true
			}
		case "genre":
			if matchStringList(show.Genres, rule.Operator, rule.Value) {
				return true
			}
		case "year":
			if matchInt(show.Year, rule.Operator, rule.Value) {
				return true
			}
		case "runtime":
			if matchInt(tvShowRuntimeMinutes(show), rule.Operator, rule.Value) {
				return true
			}
		case "rating":
			if matchString(show.Rating, rule.Operator, rule.Value) {
				return true
			}
		case "first_air_date":
			if matchDate(show.FirstAirDate, rule.Operator, rule.Value) {
				return true
			}
		case "date_added":
			if matchDate(datePart(show.CreatedAt.Format(time.RFC3339)), rule.Operator, rule.Value) {
				return true
			}
		case "network":
			if matchString(show.Network, rule.Operator, rule.Value) {
				return true
			}
		case "studio":
			if matchString(show.Studio, rule.Operator, rule.Value) {
				return true
			}
		case "actor":
			if matchStringList(show.Actors, rule.Operator, rule.Value) {
				return true
			}
		case "director":
			if matchStringList(show.Directors, rule.Operator, rule.Value) {
				return true
			}
		case "writer":
			if matchStringList(show.Writers, rule.Operator, rule.Value) {
				return true
			}
		case "country":
			if matchStringList(show.Countries, rule.Operator, rule.Value) {
				return true
			}
		case "language":
			if matchStringList(show.Languages, rule.Operator, rule.Value) {
				return true
			}
		case "series_status":
			if matchString(show.SeriesStatus, rule.Operator, rule.Value) {
				return true
			}
		case "tag":
			if matchStringList(show.Tags, rule.Operator, rule.Value) {
				return true
			}
		}
	}
	return false
}

func ruleIsInclude(rule models.ChannelRule) bool {
	switch rule.Operator {
	case "is", "equals":
		return true
	default:
		return false
	}
}

func allRulesAreTVPoolIncludes(rules []models.ChannelRule) bool {
	poolRules := 0
	for _, rule := range rulesForMediaType(rules, "tv") {
		if rule.Field == "quality" {
			continue
		}
		if !ruleIsInclude(rule) || (rule.Field != "title" && rule.Field != "network") {
			return false
		}
		poolRules++
	}
	return poolRules > 0
}

// hdMinHeight matches models.ScheduledProgram.IsHD. Media that has not been
// probed has no height and counts as SD.
const hdMinHeight = 720

func qualityRulesFor(rules []models.ChannelRule, mediaType string) []models.ChannelRule {
	quality := []models.ChannelRule{}
	for _, rule := range rulesForMediaType(rules, mediaType) {
		if rule.Field == "quality" {
			quality = append(quality, rule)
		}
	}
	return quality
}

func qualityRulesMatch(height int, rules []models.ChannelRule) bool {
	quality := "sd"
	if height >= hdMinHeight {
		quality = "hd"
	}
	for _, rule := range rules {
		if !matchString(quality, rule.Operator, rule.Value) {
			return false
		}
	}
	return true
}

// filterMoviesByQuality applies quality rules as a hard constraint, including
// to movies that only matched through the any-include fallback.
func filterMoviesByQuality(movies []models.MovieRecord, rules []models.ChannelRule) []models.MovieRecord {
	quality := qualityRulesFor(rules, "movie")
	if len(quality) == 0 {
		return movies
	}
	filtered := []models.MovieRecord{}
	for _, movie := range movies {
		if qualityRulesMatch(movie.MediaMetadata.Height, quality) {
			filtered = append(filtered, movie)
		}
	}
	return filtered
}

// filterTVShowsByQuality narrows each show to the episodes that match the
// quality rules, since a show can mix SD and HD episodes. Shows with no
// matching episodes are dropped.
func filterTVShowsByQuality(shows []models.TVShowRecord, rules []models.ChannelRule) []models.TVShowRecord {
	quality := qualityRulesFor(rules, "tv")
	if len(quality) == 0 {
		return shows
	}
	filtered := []models.TVShowRecord{}
	for _, show := range shows {
		heights := make(map[string]int, len(show.EpisodeMetadata))
		for _, episode := range show.EpisodeMetadata {
			heights[episode.Path] = episode.MediaMetadata.Height
		}
		paths := []string{}
		for _, path := range show.EpisodePaths {
			if qualityRulesMatch(heights[path], quality) {
				paths = append(paths, path)
			}
		}
		if len(paths) == 0 {
			continue
		}
		show.EpisodePaths = paths
		show.RestrictToEpisodePaths = true
		filtered = append(filtered, show)
	}
	return filtered
}

func rulesForMediaType(rules []models.ChannelRule, mediaType string) []models.ChannelRule {
	filtered := make([]models.ChannelRule, 0, len(rules))
	for _, rule := range rules {
		if rule.MediaType == "" || rule.MediaType == mediaType {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func titleRulesMatch(titles []string, rules []models.ChannelRule) bool {
	hasIncludeRule := false
	includeMatched := false
	for _, rule := range rules {
		if rule.Field != "title" {
			continue
		}
		switch rule.Operator {
		case "is":
			hasIncludeRule = true
			if matchAnyString(titles, rule.Operator, rule.Value) {
				includeMatched = true
			}
		case "is_not":
			if matchAnyString(titles, "is", rule.Value) {
				return false
			}
		default:
			return false
		}
	}
	return !hasIncludeRule || includeMatched
}

func recordTitles(values ...string) []string {
	titles := make([]string, 0, len(values))
	seen := map[string]struct{}{}
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
		titles = append(titles, value)
	}
	return titles
}

func matchAnyString(values []string, operator string, expected string) bool {
	for _, value := range values {
		if matchString(value, operator, expected) {
			return true
		}
	}
	return false
}

func matchString(value string, operator string, expected string) bool {
	found := strings.EqualFold(value, expected)
	switch operator {
	case "is":
		return found
	case "is_not":
		return !found
	default:
		return false
	}
}

func matchStringList(values []string, operator string, expected string) bool {
	found := false
	for _, value := range values {
		if strings.EqualFold(value, expected) {
			found = true
			break
		}
	}

	switch operator {
	case "is":
		return found
	case "is_not":
		return !found
	default:
		return false
	}
}

func matchInt(value int, operator string, expected string) bool {
	expectedInt, err := strconv.Atoi(expected)
	if err != nil {
		return false
	}

	switch operator {
	case "equals":
		return value == expectedInt
	case "greater_than":
		return value > expectedInt
	case "less_than":
		return value < expectedInt
	default:
		return false
	}
}

func matchDate(value string, operator string, expected string) bool {
	if value == "" || expected == "" {
		return false
	}
	valueTime, err := time.Parse("2006-01-02", value)
	if err != nil {
		return false
	}
	expectedTime, err := time.Parse("2006-01-02", expected)
	if err != nil {
		return false
	}
	switch operator {
	case "equals":
		return valueTime.Equal(expectedTime)
	case "greater_than":
		return valueTime.After(expectedTime)
	case "less_than":
		return valueTime.Before(expectedTime)
	default:
		return false
	}
}

func datePart(value string) string {
	if len(value) >= len("2006-01-02") {
		return value[:len("2006-01-02")]
	}
	return ""
}

func formatDBTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
