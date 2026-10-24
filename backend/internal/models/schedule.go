package models

import "time"

type ScheduledProgram struct {
	ID          int64        `json:"id"`
	ChannelID   int64        `json:"channel_id"`
	MediaID     int64        `json:"media_id"`
	MediaType   string       `json:"media_type"`
	EpisodePath string       `json:"episode_path,omitempty"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	Media       MovieRecord  `json:"media,omitempty"`
	TVMedia     TVShowRecord `json:"tv_media,omitempty"`
	Asset       ChannelAsset `json:"asset,omitempty"`
}

type ChannelScheduleState struct {
	ChannelID           int64         `json:"channel_id"`
	NextMediaIndex      int           `json:"next_media_index"`
	LastMediaID         int64         `json:"last_media_id"`
	EpisodeIndexes      map[int64]int `json:"episode_indexes,omitempty"`
	LastScheduleEndTime time.Time     `json:"last_schedule_end_time"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

// HasClosedCaptions reports whether the scheduled movie or TV episode has an
// embedded subtitle stream. Channel assets are intentionally excluded.
func (p ScheduledProgram) HasClosedCaptions() bool {
	switch p.MediaType {
	case "movie":
		return len(p.Media.MediaMetadata.SubtitleTracks) > 0
	case "tv":
		for _, episode := range p.TVMedia.EpisodeMetadata {
			if episode.Path == p.EpisodePath {
				return len(episode.MediaMetadata.SubtitleTracks) > 0
			}
		}
	}
	return false
}

// IsHD reports whether the scheduled movie or TV episode has a scanned video
// height of at least 720 pixels. Channel assets are intentionally excluded.
func (p ScheduledProgram) IsHD() bool {
	switch p.MediaType {
	case "movie":
		return p.Media.MediaMetadata.Height >= 720
	case "tv":
		for _, episode := range p.TVMedia.EpisodeMetadata {
			if episode.Path == p.EpisodePath {
				return episode.MediaMetadata.Height >= 720
			}
		}
	}
	return false
}
