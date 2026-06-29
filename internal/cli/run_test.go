package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"streamtui/internal/domain"
	"streamtui/internal/providers"
)

func TestRunWritesSuccessEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := RunWithApp([]string{"search", "arrival"}, fakeApp(), &stdout, &stderr)
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d; stderr=%s", code, ExitSuccess, stderr.String())
	}

	var env Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v; output=%s", err, stdout.String())
	}
	if !env.OK {
		t.Fatalf("OK = false, want true")
	}
}

func TestRunWritesErrorEnvelopeAndExitCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"seek"}, &stdout, &stderr)
	if code != ExitInvalidArgs {
		t.Fatalf("code = %d, want %d; stderr=%s", code, ExitInvalidArgs, stderr.String())
	}

	var env Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v; output=%s", err, stdout.String())
	}
	if env.OK {
		t.Fatalf("OK = true, want false")
	}
	if env.Error == nil {
		t.Fatalf("Error = nil, want detail")
	}
	if env.Error.Code != ErrorInvalidArgs {
		t.Fatalf("error code = %q, want %q", env.Error.Code, ErrorInvalidArgs)
	}
}

func TestRunAgentFirstPlayWritesSuccessEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	playback := &fakePlaybackAdapter{}
	app := agentPlayApp(playback)

	code := RunWithApp([]string{"play", "the batman", "--lang", "es", "--device", "Living Room TV", "--json"}, app, &stdout, &stderr)
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d; stderr=%s; stdout=%s", code, ExitSuccess, stderr.String(), stdout.String())
	}
	if len(playback.requests) != 1 {
		t.Fatalf("playback requests = %d, want 1", len(playback.requests))
	}
	if playback.requests[0].Device != "Living Room TV" {
		t.Fatalf("device = %q, want Living Room TV", playback.requests[0].Device)
	}

	var env Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v; output=%s", err, stdout.String())
	}
	if !env.OK {
		t.Fatalf("OK = false, want true; output=%s", stdout.String())
	}
	if env.Meta.Version != "v1" {
		t.Fatalf("version = %q, want v1", env.Meta.Version)
	}
}

func TestRunAgentFirstPlayWithDefaultAppAndFakeServices(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "webtorrent-args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(argsPath) + "\nsleep 1\n"
	if err := os.WriteFile(filepath.Join(dir, "webtorrent"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))
	t.Setenv("TMDB_API_KEY", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search/multi":
			_, _ = w.Write([]byte(`{"results":[{"id":414906,"media_type":"movie","title":"The Batman","release_date":"2022-03-01"}]}`))
		case "/movie/414906":
			_, _ = w.Write([]byte(`{"id":414906,"external_ids":{"imdb_id":"tt1877830"},"title":"The Batman","release_date":"2022-03-01","genres":[]}`))
		case "/stream/movie/tt1877830.json":
			_, _ = w.Write([]byte(`{"streams":[{"name":"Torrentio 1080p","title":"Release\n👤 10\n2.2 GB","infoHash":"bbb","fileIdx":0}]}`))
		case "/subtitles/movie/tt1877830.json":
			_, _ = w.Write([]byte(`{"subtitles":[{"id":"sub1","url":"` + serverURL(r) + `/download/sub1.srt","lang":"spa"}]}`))
		case "/download/sub1.srt":
			_, _ = w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nHola\n"))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("STREAMTUI_TMDB_BASE_URL", server.URL)
	t.Setenv("STREAMTUI_TORRENTIO_BASE_URL", server.URL)
	t.Setenv("STREAMTUI_SUBTITLES_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"play", "the batman", "--lang", "es", "--device", "Living Room TV", "--json"}, &stdout, &stderr)
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d; stderr=%s; stdout=%s", code, ExitSuccess, stderr.String(), stdout.String())
	}

	args := waitForRunFile(t, argsPath)
	if !strings.Contains(args, "--chromecast\nLiving Room TV\n") {
		t.Fatalf("webtorrent args = %q, want Chromecast device", args)
	}
	if !strings.Contains(args, "-t\n") {
		t.Fatalf("webtorrent args = %q, want subtitle flag", args)
	}

	var env Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v; output=%s", err, stdout.String())
	}
	if !env.OK || env.Meta.Version != "v1" {
		t.Fatalf("envelope = %#v, want ok v1", env)
	}
}

func TestRunDevicesRefreshDeviceNotFoundEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := App{
		Devices:     &fakeDeviceProvider{err: providers.ErrNotFound},
		DeviceStore: &memoryDeviceStore{},
	}

	code := RunWithApp([]string{"devices", "refresh", "--json"}, app, &stdout, &stderr)
	if code != ExitDeviceNotFound {
		t.Fatalf("code = %d, want %d; stderr=%s; stdout=%s", code, ExitDeviceNotFound, stderr.String(), stdout.String())
	}

	var env Envelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v; output=%s", err, stdout.String())
	}
	if env.OK {
		t.Fatalf("OK = true, want false")
	}
	if env.Error == nil {
		t.Fatalf("Error = nil, want detail")
	}
	if env.Error.Code != ErrorDeviceNotFound {
		t.Fatalf("error code = %q, want %q", env.Error.Code, ErrorDeviceNotFound)
	}
	if env.Error.Message != "no Chromecast devices found" {
		t.Fatalf("message = %q, want no Chromecast devices found", env.Error.Message)
	}
	if env.Meta.Version != "v1" {
		t.Fatalf("version = %q, want v1", env.Meta.Version)
	}
}

func fakeApp() App {
	return App{
		Metadata: fakeMetadataProvider{
			search: []domain.SearchResult{{ID: 1, MediaType: domain.MediaMovie, Title: "Arrival"}},
		},
		Streams: fakeStreamProvider{},
	}
}

func waitForRunFile(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
	return ""
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func serverURL(request *http.Request) string {
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + request.Host
}

func agentPlayApp(playback *fakePlaybackAdapter) App {
	return App{
		Metadata: fakeMetadataProvider{
			search: []domain.SearchResult{{ID: 414906, MediaType: domain.MediaMovie, Title: "The Batman"}},
			movie:  domain.MovieDetail{ID: 414906, IMDBID: "tt1877830", Title: "The Batman"},
		},
		Streams: fakeStreamProvider{movie: []domain.StreamSource{
			{Name: "1080", InfoHash: "bbb", Quality: domain.Quality1080p, Seeds: 10},
		}},
		Subtitles: fakeSubtitleProvider{movie: []domain.SubtitleResult{
			{ID: "trusted", Language: "spa", FromTrusted: true},
		}},
		Playback: playback,
		DeviceStore: &memoryDeviceStore{cache: domain.DeviceCache{
			Devices: []domain.CachedDevice{{CastDevice: domain.CastDevice{Name: "Living Room TV", Address: "192.168.1.36"}}},
		}},
		DeviceCacheTTL: time.Hour,
	}
}
