package models

import "time"

type Collection struct {
	ID            int64             `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	ArtworkPath   string            `json:"artwork_path,omitempty"`
	ArtworkMode   string            `json:"artwork_mode,omitempty"`
	ArtworkMedia  []CollectionMedia `json:"artwork_media,omitempty"`
	Rules         []ChannelRule     `json:"rules,omitempty"`
	PinnedMedia   []CollectionMedia `json:"pinned_media,omitempty"`
	ExcludedMedia []CollectionMedia `json:"excluded_media,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type CollectionMedia struct {
	MediaType string `json:"media_type"`
	MediaID   int64  `json:"media_id"`
}
