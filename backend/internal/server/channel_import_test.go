package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"retrotv/internal/models"
)

func TestParseRTVChannelsImportsBundle(t *testing.T) {
	data := testRTVBundle(t, "channels.json", []byte(`{
		"channels": [
			{
				"number": 2,
				"name": "Chiller",
				"content_type": "movie",
				"rules": [{"field": "genre", "operator": "is", "value": "Horror"}]
			},
			{
				"number": 1,
				"name": "FOX",
				"content_type": "tv",
				"logo": "logos/fox.png",
				"rules": [{"field": "network", "operator": "is", "value": "FOX"}]
			}
		]
	}`), map[string][]byte{
		"logos/002.png": []byte("chiller"),
		"logos/fox.png": []byte("fox"),
	})

	channels, _, err := parseRTVChannels(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(channels) != 2 {
		t.Fatalf("len(channels) = %d, want 2", len(channels))
	}
	if channels[0].Name != "FOX" || channels[0].LogoHref != "logos/fox.png" {
		t.Fatalf("first channel = %#v, want FOX with explicit logo", channels[0])
	}
	if channels[1].Name != "Chiller" || channels[1].LogoHref != "logos/002.png" {
		t.Fatalf("second channel = %#v, want Chiller with default logo", channels[1])
	}
}

func TestParseRTVChannelsAcceptsTopLevelArray(t *testing.T) {
	data := testRTVBundle(t, "channels.json", []byte(`[
		{"number": 1, "name": "Movies", "content_type": "movie"}
	]`), nil)

	channels, _, err := parseRTVChannels(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 1 || channels[0].Name != "Movies" {
		t.Fatalf("channels = %#v, want Movies", channels)
	}
}

func TestParseRTVChannelsRejectsInvalidRule(t *testing.T) {
	data := testRTVBundle(t, "channels.json", []byte(`{
		"channels": [
			{"number": 1, "name": "Broken", "content_type": "movie", "rules": [{"field": "year", "operator": "is", "value": "old"}]}
		]
	}`), nil)

	_, _, err := parseRTVChannels(data)
	if err == nil {
		t.Fatal("parseRTVChannels succeeded, want invalid rule error")
	}
}

func TestImportChannelsEndpointRequiresRTVExtension(t *testing.T) {
	server, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.db.Close(); err != nil {
			t.Fatal(err)
		}
	})

	response := postChannelBundle(t, server, "channels.zip", testRTVBundle(t, "channels.json", []byte(`[]`), nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestImportChannelsEndpointCreatesChannelsAndLogos(t *testing.T) {
	configDir := t.TempDir()
	server, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        configDir,
		ArtworkDir:       filepath.Join(configDir, "artwork"),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.db.Close(); err != nil {
			t.Fatal(err)
		}
	})

	data := testRTVBundle(t, "channels.json", []byte(`{
		"channels": [
			{
				"number": 1,
				"name": "FOX",
				"content_type": "tv",
				"rules": [{"field": "network", "operator": "is", "value": "FOX"}]
			},
			{
				"number": 2,
				"name": "Chiller",
				"content_type": "movie",
				"rules": [{"field": "genre", "operator": "is", "value": "Horror"}]
			}
		]
	}`), map[string][]byte{
		"logos/001.png": []byte("fox"),
	})

	response := postChannelBundle(t, server, "channels.rtv", data)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var result channelImportResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Imported != 2 || result.Created != 2 || result.Updated != 0 {
		t.Fatalf("result = %#v, want 2 imported/created", result)
	}

	channels, err := server.db.Channels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 2 {
		t.Fatalf("len(channels) = %d, want 2", len(channels))
	}
	if channels[0].Name != "FOX" || channels[1].Name != "Chiller" {
		t.Fatalf("channels = %#v, want FOX then Chiller", channels)
	}
	if channels[0].LogoPath == "" {
		t.Fatal("FOX logo path is empty")
	}
	logoPath := filepath.Join(server.artworkDir, strings.TrimPrefix(channels[0].LogoPath, "/artwork/"))
	if _, err := os.Stat(logoPath); err != nil {
		t.Fatalf("imported logo missing at %s: %v", logoPath, err)
	}
}

func TestExportChannelsEndpointRequiresChannels(t *testing.T) {
	server, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        t.TempDir(),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.db.Close(); err != nil {
			t.Fatal(err)
		}
	})

	request := httptest.NewRequest(http.MethodGet, "/api/channels/export", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestExportChannelsEndpointCreatesRTVBundle(t *testing.T) {
	configDir := t.TempDir()
	server, err := NewServer(Config{
		StaticDir:        ".",
		ConfigDir:        configDir,
		ArtworkDir:       filepath.Join(configDir, "artwork"),
		DisableScheduler: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.db.Close(); err != nil {
			t.Fatal(err)
		}
	})

	logoPath := filepath.Join(server.artworkDir, "channels", "1", "logo.png")
	if err := os.MkdirAll(filepath.Dir(logoPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logoPath, []byte("fox-logo"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := server.db.SaveChannel(context.Background(), models.Channel{
		Name:        "FOX",
		ContentType: "tv",
		LogoPath:    "/artwork/channels/1/logo.png",
		Rules:       []models.ChannelRule{{Field: "network", Operator: "is", Value: "FOX"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := server.db.SaveChannel(context.Background(), models.Channel{
		Name:        "Chiller",
		ContentType: "movie",
		Rules:       []models.ChannelRule{{Field: "genre", Operator: "is", Value: "Horror"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.db.ReorderChannels(context.Background(), []int64{second.ID, first.ID}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/channels/export", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if !strings.Contains(response.Header().Get("Content-Disposition"), "retrotv-channels.rtv") {
		t.Fatalf("content disposition = %q", response.Header().Get("Content-Disposition"))
	}

	reader, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	content, err := readZipFile(reader, "channels.json")
	if err != nil {
		t.Fatal(err)
	}
	var bundle rtvChannelBundle
	if err := json.Unmarshal(content, &bundle); err != nil {
		t.Fatal(err)
	}
	if len(bundle.Channels) != 2 {
		t.Fatalf("channels = %#v", bundle.Channels)
	}
	if bundle.Channels[0].Number != 1 || bundle.Channels[0].Name != "Chiller" {
		t.Fatalf("first exported channel = %#v, want Chiller as channel 1", bundle.Channels[0])
	}
	if bundle.Channels[1].Number != 2 || bundle.Channels[1].Name != "FOX" || bundle.Channels[1].Logo != "logos/002.png" {
		t.Fatalf("second exported channel = %#v, want FOX with logo", bundle.Channels[1])
	}
	logo, err := readZipFile(reader, "logos/002.png")
	if err != nil {
		t.Fatal(err)
	}
	if string(logo) != "fox-logo" {
		t.Fatalf("logo content = %q", logo)
	}
}

func postChannelBundle(t *testing.T, server *Server, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("channels", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/channels/import", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}

func testRTVBundle(t *testing.T, jsonName string, jsonContent []byte, files map[string][]byte) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	contentFile, err := writer.Create(jsonName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contentFile.Write(jsonContent); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
