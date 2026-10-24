package models

import "time"

type ChannelAsset struct {
	ID              int64         `json:"id"`
	ChannelID       int64         `json:"channel_id,omitempty"`
	FilePath        string        `json:"file_path"`
	Filename        string        `json:"filename"`
	AssetType       string        `json:"asset_type"`
	Tag             string        `json:"tag,omitempty"`
	Scope           string        `json:"scope"`
	DurationSeconds int           `json:"duration_seconds"`
	Enabled         bool          `json:"enabled"`
	MediaMetadata   MediaMetadata `json:"media_metadata,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}
