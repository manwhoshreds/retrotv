package metadata

import "testing"

func TestParseFFprobeMetadata(t *testing.T) {
	metadata, err := ParseFFprobeMetadata([]byte(`{
		"streams": [
			{"index":0,"codec_name":"h264","codec_type":"video","width":1920,"height":1080,"avg_frame_rate":"24000/1001"},
			{"index":1,"codec_name":"aac","codec_type":"audio","channels":6,"tags":{"language":"eng"}},
			{"index":2,"codec_name":"ac3","codec_type":"audio","channels":2,"tags":{"language":"spa"}},
			{"index":3,"codec_name":"subrip","codec_type":"subtitle","tags":{"language":"eng"}}
		],
		"format": {"duration":"601.234000"}
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if metadata.DurationSeconds != 601.234 {
		t.Fatalf("DurationSeconds = %v", metadata.DurationSeconds)
	}
	if metadata.VideoCodec != "h264" || metadata.AudioCodec != "aac" {
		t.Fatalf("codecs = %q/%q", metadata.VideoCodec, metadata.AudioCodec)
	}
	if metadata.Width != 1920 || metadata.Height != 1080 {
		t.Fatalf("resolution = %dx%d", metadata.Width, metadata.Height)
	}
	if metadata.FrameRate != 23.976 {
		t.Fatalf("FrameRate = %v", metadata.FrameRate)
	}
	if len(metadata.AudioTracks) != 2 {
		t.Fatalf("audio tracks = %#v", metadata.AudioTracks)
	}
	if metadata.AudioTracks[0].Language != "eng" || metadata.AudioTracks[0].Channels != 6 {
		t.Fatalf("first audio track = %#v", metadata.AudioTracks[0])
	}
	if len(metadata.SubtitleTracks) != 1 || metadata.SubtitleTracks[0].Language != "eng" {
		t.Fatalf("subtitle tracks = %#v", metadata.SubtitleTracks)
	}
}
