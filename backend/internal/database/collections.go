package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"retrotv/internal/models"
)

func (s *Store) Collections(ctx context.Context) ([]models.Collection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, artwork_path, artwork_mode, created_at, updated_at FROM collections ORDER BY lower(name), id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	collections := []models.Collection{}
	for rows.Next() {
		collection, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		if err := s.loadCollectionRelations(ctx, &collection); err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}
	return collections, rows.Err()
}

func (s *Store) Collection(ctx context.Context, id int64) (models.Collection, error) {
	collection, err := scanCollection(s.db.QueryRowContext(ctx, `SELECT id, name, description, artwork_path, artwork_mode, created_at, updated_at FROM collections WHERE id = ?`, id))
	if err != nil {
		return models.Collection{}, err
	}
	if err := s.loadCollectionRelations(ctx, &collection); err != nil {
		return models.Collection{}, err
	}
	return collection, nil
}

func (s *Store) SaveCollection(ctx context.Context, collection models.Collection) (models.Collection, error) {
	collection.Name = strings.TrimSpace(collection.Name)
	if collection.Name == "" {
		return models.Collection{}, fmt.Errorf("collection name is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Collection{}, err
	}
	defer tx.Rollback()
	if collection.ID == 0 {
		result, err := tx.ExecContext(ctx, `INSERT INTO collections (name, description, artwork_path, artwork_mode) VALUES (?, ?, ?, ?)`, collection.Name, collection.Description, collection.ArtworkPath, collection.ArtworkMode)
		if err != nil {
			return models.Collection{}, err
		}
		collection.ID, err = result.LastInsertId()
		if err != nil {
			return models.Collection{}, err
		}
	} else {
		result, err := tx.ExecContext(ctx, `UPDATE collections SET name = ?, description = ?, artwork_path = ?, artwork_mode = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ?`, collection.Name, collection.Description, collection.ArtworkPath, collection.ArtworkMode, collection.ID)
		if err != nil {
			return models.Collection{}, err
		}
		if changed, err := result.RowsAffected(); err != nil || changed == 0 {
			if err != nil {
				return models.Collection{}, err
			}
			return models.Collection{}, sql.ErrNoRows
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM collection_rules WHERE collection_id = ?`, collection.ID); err != nil {
		return models.Collection{}, err
	}
	for _, rule := range collection.Rules {
		if _, err := tx.ExecContext(ctx, `INSERT INTO collection_rules (collection_id, media_type, field, operator, value) VALUES (?, ?, ?, ?, ?)`, collection.ID, rule.MediaType, rule.Field, rule.Operator, rule.Value); err != nil {
			return models.Collection{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM collection_media WHERE collection_id = ?`, collection.ID); err != nil {
		return models.Collection{}, err
	}
	for index, media := range collection.PinnedMedia {
		if (media.MediaType != "movie" && media.MediaType != "tv") || media.MediaID <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO collection_media (collection_id, media_type, media_id, sort_order) VALUES (?, ?, ?, ?)`, collection.ID, media.MediaType, media.MediaID, index+1); err != nil {
			return models.Collection{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM collection_excluded_media WHERE collection_id = ?`, collection.ID); err != nil {
		return models.Collection{}, err
	}
	for _, media := range collection.ExcludedMedia {
		if (media.MediaType != "movie" && media.MediaType != "tv") || media.MediaID <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO collection_excluded_media (collection_id, media_type, media_id) VALUES (?, ?, ?)`, collection.ID, media.MediaType, media.MediaID); err != nil {
			return models.Collection{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM collection_artwork_media WHERE collection_id = ?`, collection.ID); err != nil {
		return models.Collection{}, err
	}
	for index, media := range collection.ArtworkMedia {
		if index >= 2 || (media.MediaType != "movie" && media.MediaType != "tv") || media.MediaID <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO collection_artwork_media (collection_id, media_type, media_id, sort_order) VALUES (?, ?, ?, ?)`, collection.ID, media.MediaType, media.MediaID, index+1); err != nil {
			return models.Collection{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Collection{}, err
	}
	return s.Collection(ctx, collection.ID)
}

func (s *Store) DeleteCollection(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, err := result.RowsAffected(); err != nil || changed == 0 {
		if err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) loadCollectionRelations(ctx context.Context, collection *models.Collection) error {
	ruleRows, err := s.db.QueryContext(ctx, `SELECT id, collection_id, media_type, field, operator, value FROM collection_rules WHERE collection_id = ? ORDER BY id`, collection.ID)
	if err != nil {
		return err
	}
	collection.Rules = []models.ChannelRule{}
	for ruleRows.Next() {
		var rule models.ChannelRule
		if err := ruleRows.Scan(&rule.ID, &rule.ChannelID, &rule.MediaType, &rule.Field, &rule.Operator, &rule.Value); err != nil {
			ruleRows.Close()
			return err
		}
		collection.Rules = append(collection.Rules, rule)
	}
	if err := ruleRows.Close(); err != nil {
		return err
	}
	mediaRows, err := s.db.QueryContext(ctx, `SELECT media_type, media_id FROM collection_media WHERE collection_id = ? ORDER BY sort_order, media_id`, collection.ID)
	if err != nil {
		return err
	}
	collection.PinnedMedia = []models.CollectionMedia{}
	for mediaRows.Next() {
		var media models.CollectionMedia
		if err := mediaRows.Scan(&media.MediaType, &media.MediaID); err != nil {
			return err
		}
		collection.PinnedMedia = append(collection.PinnedMedia, media)
	}
	if err := mediaRows.Close(); err != nil {
		return err
	}
	excludedRows, err := s.db.QueryContext(ctx, `SELECT media_type, media_id FROM collection_excluded_media WHERE collection_id = ? ORDER BY media_type, media_id`, collection.ID)
	if err != nil {
		return err
	}
	defer excludedRows.Close()
	collection.ExcludedMedia = []models.CollectionMedia{}
	for excludedRows.Next() {
		var media models.CollectionMedia
		if err := excludedRows.Scan(&media.MediaType, &media.MediaID); err != nil {
			return err
		}
		collection.ExcludedMedia = append(collection.ExcludedMedia, media)
	}
	if err := excludedRows.Close(); err != nil {
		return err
	}
	artworkRows, err := s.db.QueryContext(ctx, `SELECT media_type, media_id FROM collection_artwork_media WHERE collection_id = ? ORDER BY sort_order, media_id`, collection.ID)
	if err != nil {
		return err
	}
	defer artworkRows.Close()
	collection.ArtworkMedia = []models.CollectionMedia{}
	for artworkRows.Next() {
		var media models.CollectionMedia
		if err := artworkRows.Scan(&media.MediaType, &media.MediaID); err != nil {
			return err
		}
		collection.ArtworkMedia = append(collection.ArtworkMedia, media)
	}
	return artworkRows.Err()
}

type collectionScanner interface{ Scan(...any) error }

func scanCollection(row collectionScanner) (models.Collection, error) {
	var collection models.Collection
	var created, updated string
	if err := row.Scan(&collection.ID, &collection.Name, &collection.Description, &collection.ArtworkPath, &collection.ArtworkMode, &created, &updated); err != nil {
		return models.Collection{}, err
	}
	var err error
	collection.CreatedAt, err = time.Parse(time.RFC3339, created)
	if err != nil {
		return models.Collection{}, err
	}
	collection.UpdatedAt, err = time.Parse(time.RFC3339, updated)
	return collection, err
}
