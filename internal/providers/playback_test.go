package providers

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"streamtui/internal/domain"
)

func TestBuildWebtorrentArgsVLC(t *testing.T) {
	args, err := BuildWebtorrentArgs(domain.PlaybackRequest{
		Magnet:        "magnet:?xt=urn:btih:abc",
		VLC:           true,
		SubtitleFile:  "/tmp/sub.vtt",
		SubtitleDelay: "2.5",
		FileIndex:     2,
	})
	if err != nil {
		t.Fatalf("BuildWebtorrentArgs returned error: %v", err)
	}
	want := []string{"magnet:?xt=urn:btih:abc", "--vlc", "--not-on-top", "-s", "2", "--player-args=--sub-file=/tmp/sub.vtt --sub-delay=2.5"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestBuildWebtorrentArgsChromecast(t *testing.T) {
	args, err := BuildWebtorrentArgs(domain.PlaybackRequest{
		Magnet:       "magnet:?xt=urn:btih:abc",
		Device:       "Living Room TV",
		SubtitleFile: "/tmp/sub.vtt",
		FileIndex:    1,
	})
	if err != nil {
		t.Fatalf("BuildWebtorrentArgs returned error: %v", err)
	}
	want := []string{"magnet:?xt=urn:btih:abc", "--chromecast", "Living Room TV", "--not-on-top", "-s", "1", "-t", "/tmp/sub.vtt"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestBuildWebtorrentArgsTorrentFile(t *testing.T) {
	args, err := BuildWebtorrentArgs(domain.PlaybackRequest{
		Magnet:    "/tmp/movie.torrent",
		VLC:       true,
		FileIndex: 4,
	})
	if err != nil {
		t.Fatalf("BuildWebtorrentArgs returned error: %v", err)
	}
	want := []string{"/tmp/movie.torrent", "--vlc", "--not-on-top", "-s", "4"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestBuildWebtorrentArgsRequiresDeviceForChromecast(t *testing.T) {
	_, err := BuildWebtorrentArgs(domain.PlaybackRequest{Magnet: "magnet:?xt=urn:btih:abc"})
	if err == nil {
		t.Fatalf("BuildWebtorrentArgs returned nil error")
	}
}

func TestWebtorrentPlaybackAdapterStartsFakeBinary(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(argsPath) + "\nsleep 1\n"
	if err := os.WriteFile(filepath.Join(dir, "webtorrent"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	adapter := WebtorrentPlaybackAdapter{}
	err := adapter.PlayMagnet(domain.PlaybackRequest{
		Magnet:    "magnet:?xt=urn:btih:abc",
		VLC:       true,
		FileIndex: 2,
	})
	if err != nil {
		t.Fatalf("PlayMagnet returned error: %v", err)
	}

	args := waitForFile(t, argsPath)
	want := "magnet:?xt=urn:btih:abc\n--vlc\n--not-on-top\n-s\n2\n"
	if args != want {
		t.Fatalf("args = %q, want %q", args, want)
	}
}

func TestWebtorrentPlaybackAdapterReportsImmediateExit(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nexit 7\n"
	if err := os.WriteFile(filepath.Join(dir, "webtorrent"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	adapter := WebtorrentPlaybackAdapter{}
	err := adapter.PlayMagnet(domain.PlaybackRequest{
		Magnet:    "magnet:?xt=urn:btih:abc",
		VLC:       true,
		FileIndex: 2,
	})
	if err == nil {
		t.Fatalf("PlayMagnet returned nil error")
	}
	if !strings.Contains(err.Error(), "webtorrent exited immediately") {
		t.Fatalf("error = %v, want immediate exit", err)
	}
}

func waitForFile(t *testing.T, path string) string {
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
