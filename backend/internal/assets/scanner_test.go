package assets

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"retrotv/internal/config"
	"retrotv/internal/models"
)

func TestScannerDiscoversChannelAssetsByFolder(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, filepath.Join(root, "bumpers", "cold-open.mp4"))
	writeAsset(t, filepath.Join(root, "commercials", "toys", "toy-ad.mkv"))
	writeAsset(t, filepath.Join(root, "promos", "next.mp4"))
	writeAsset(t, filepath.Join(root, "station_ids", "ident.mov"))
	writeAsset(t, filepath.Join(root, "trailers", "ignored.mp4"))
	writeAsset(t, filepath.Join(root, "bumpers", "ignored.txt"))

	scanner := Scanner{Prober: fakeProber{duration: 29.6}}
	assets, scanErrors := scanner.Scan(context.Background(), []config.AssetSource{{Path: root, Scope: config.AssetScopeGlobal}})
	if len(scanErrors) != 0 {
		t.Fatalf("scanErrors = %#v", scanErrors)
	}
	if len(assets) != 4 {
		t.Fatalf("len(assets) = %d, want 4: %#v", len(assets), assets)
	}

	byFile := map[string]models.ChannelAsset{}
	for _, asset := range assets {
		byFile[asset.Filename] = asset
		if asset.Scope != config.AssetScopeGlobal {
			t.Fatalf("Scope = %q, want global", asset.Scope)
		}
		if asset.DurationSeconds != 30 {
			t.Fatalf("DurationSeconds = %d, want 30", asset.DurationSeconds)
		}
	}
	if byFile["cold-open.mp4"].AssetType != "bumper" {
		t.Fatalf("bumper asset = %#v", byFile["cold-open.mp4"])
	}
	if byFile["toy-ad.mkv"].AssetType != "commercial" || byFile["toy-ad.mkv"].Tag != "toys" {
		t.Fatalf("commercial asset = %#v", byFile["toy-ad.mkv"])
	}
	if byFile["next.mp4"].AssetType != "promo" {
		t.Fatalf("promo asset = %#v", byFile["next.mp4"])
	}
	if byFile["ident.mov"].AssetType != "station_id" {
		t.Fatalf("station id asset = %#v", byFile["ident.mov"])
	}
}

func TestScannerDiscoversChannelBumpersBySlug(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, filepath.Join(root, "channels", "fox_kids", "bumpers", "fox-kids-id.mp4"))
	writeAsset(t, filepath.Join(root, "channels", "fox_kids", "commercials", "ignored-ad.mp4"))
	writeAsset(t, filepath.Join(root, "channels", "unknown_channel", "bumpers", "ignored.mp4"))

	scanner := Scanner{
		Prober: fakeProber{duration: 12},
		Channels: []models.Channel{
			{ID: 120, Name: "FOX Kids"},
		},
	}
	assets, scanErrors := scanner.Scan(context.Background(), []config.AssetSource{{Path: root, Scope: config.AssetScopeGlobal}})
	if len(scanErrors) != 0 {
		t.Fatalf("scanErrors = %#v", scanErrors)
	}
	if len(assets) != 1 {
		t.Fatalf("len(assets) = %d, want 1: %#v", len(assets), assets)
	}
	asset := assets[0]
	if asset.ChannelID != 120 || asset.Scope != "channel" || asset.AssetType != "bumper" || asset.Filename != "fox-kids-id.mp4" {
		t.Fatalf("asset = %#v", asset)
	}
}

func writeAsset(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
}

type fakeProber struct {
	duration float64
}

func (p fakeProber) Probe(ctx context.Context, path string) (models.MediaMetadata, error) {
	return models.MediaMetadata{DurationSeconds: p.duration}, nil
}
