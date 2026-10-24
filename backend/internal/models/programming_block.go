package models

const (
	ProgrammingBlockPlacementAllDay   = "all_day"
	ProgrammingBlockPlacementTimeline = "timeline"
)

type ProgrammingBlock struct {
	ID            int64                   `json:"id"`
	ChannelID     int64                   `json:"channel_id"`
	Title         string                  `json:"title"`
	Name          string                  `json:"name,omitempty"`
	StartTime     string                  `json:"start_time"`
	EndTime       string                  `json:"end_time"`
	Placement     string                  `json:"placement"`
	Days          []string                `json:"days"`
	ContentSource string                  `json:"content_source"`
	CollectionID  int64                   `json:"collection_id,omitempty"`
	RuleSet       string                  `json:"rule_set,omitempty"`
	ScheduleMode  string                  `json:"schedule_mode,omitempty"`
	ExactDate     string                  `json:"exact_date,omitempty"`
	ExcludedDates []string                `json:"excluded_dates,omitempty"`
	PlaybackMode  string                  `json:"playback_mode,omitempty"`
	Priority      int                     `json:"priority"`
	Enabled       bool                    `json:"enabled"`
	MediaItems    []ProgrammingBlockMedia `json:"media_items,omitempty"`
}

type ProgrammingBlockMedia struct {
	ID        int64  `json:"id,omitempty"`
	BlockID   int64  `json:"block_id,omitempty"`
	MediaType string `json:"media_type"`
	MediaID   int64  `json:"media_id"`
	SortOrder int    `json:"sort_order,omitempty"`
	Title     string `json:"title,omitempty"`
}
