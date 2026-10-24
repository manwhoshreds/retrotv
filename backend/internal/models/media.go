package models

type MediaMetadata struct {
	DurationSeconds float64         `json:"duration_seconds,omitempty"`
	VideoCodec      string          `json:"video_codec,omitempty"`
	AudioCodec      string          `json:"audio_codec,omitempty"`
	Width           int             `json:"width,omitempty"`
	Height          int             `json:"height,omitempty"`
	FrameRate       float64         `json:"frame_rate,omitempty"`
	AudioTracks     []AudioTrack    `json:"audio_tracks,omitempty"`
	SubtitleTracks  []SubtitleTrack `json:"subtitle_tracks,omitempty"`
	FileSize        int64           `json:"file_size,omitempty"`
	ModTimeUnix     int64           `json:"mod_time_unix,omitempty"`
	ProbeError      string          `json:"probe_error,omitempty"`
}

type AudioTrack struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec,omitempty"`
	Language string `json:"language,omitempty"`
	Channels int    `json:"channels,omitempty"`
}

type SubtitleTrack struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec,omitempty"`
	Language string `json:"language,omitempty"`
}
