package models

import "time"

type Channel struct {
	ID                int64               `json:"id"`
	Name              string              `json:"name"`
	Description       string              `json:"description,omitempty"`
	LogoPath          string              `json:"logo_path"`
	ContentType       string              `json:"content_type"`
	SortOrder         int                 `json:"sort_order"`
	Airing            bool                `json:"airing"`
	Presentation      ChannelPresentation `json:"presentation,omitempty"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	Rules             []ChannelRule       `json:"rules,omitempty"`
	ProgrammingBlocks []ProgrammingBlock  `json:"programming_blocks,omitempty"`
}

type ChannelPresentation struct {
	ChannelBugEnabled   bool    `json:"channel_bug_enabled"`
	ChannelBugAssetPath string  `json:"channel_bug_asset_path,omitempty"`
	ChannelBugPosition  string  `json:"channel_bug_position,omitempty"`
	ChannelBugWidth     int     `json:"channel_bug_width,omitempty"`
	ChannelBugOpacity   float64 `json:"channel_bug_opacity,omitempty"`
	ChannelBugMarginX   int     `json:"channel_bug_margin_x,omitempty"`
	ChannelBugMarginY   int     `json:"channel_bug_margin_y,omitempty"`

	RatingOverlayEnabled         bool   `json:"rating_overlay_enabled"`
	RatingOverlayPosition        string `json:"rating_overlay_position,omitempty"`
	RatingOverlayDurationSeconds int    `json:"rating_overlay_duration_seconds,omitempty"`

	NowPlayingEnabled         bool   `json:"now_playing_enabled"`
	NowPlayingPosition        string `json:"now_playing_position,omitempty"`
	NowPlayingDurationSeconds int    `json:"now_playing_duration_seconds,omitempty"`
}

type ChannelRule struct {
	ID        int64  `json:"id"`
	ChannelID int64  `json:"channel_id"`
	MediaType string `json:"media_type,omitempty"`
	Field     string `json:"field"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
}
