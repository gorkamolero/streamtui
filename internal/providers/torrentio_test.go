package providers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"streamtui/internal/domain"
)

func TestTorrentioMovieStreams(t *testing.T) {
	fileIndex := uint32(2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stream/movie/tt1877830.json" {
			t.Fatalf("path = %q, want movie stream path", r.URL.Path)
		}
		if r.Header.Get("User-Agent") != "streamtui/1.0" {
			t.Fatalf("user agent = %q, want streamtui/1.0", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte(`{
			"streams": [
				{"name": "Torrentio 720p", "title": "Release A\n👤 400\n1.1 GB", "infoHash": "aaa"},
				{"name": "Torrentio 1080p", "title": "Release B\n👤 40\n2.2 GB", "infoHash": "bbb", "fileIdx": 2}
			]
		}`))
	}))
	defer server.Close()

	client := NewTorrentioClientWithBaseURL(server.URL)
	streams, err := client.MovieStreams("tt1877830")
	if err != nil {
		t.Fatalf("MovieStreams returned error: %v", err)
	}
	if len(streams) != 2 {
		t.Fatalf("len(streams) = %d, want 2", len(streams))
	}
	if streams[0].Quality != domain.Quality1080p {
		t.Fatalf("first quality = %q, want 1080p", streams[0].Quality)
	}
	if streams[0].FileIndex == nil || *streams[0].FileIndex != fileIndex {
		t.Fatalf("file index = %v, want 2", streams[0].FileIndex)
	}
	if streams[1].Seeds != 400 {
		t.Fatalf("seeds = %d, want 400", streams[1].Seeds)
	}
}

func TestTorrentioEpisodeStreams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stream/series/tt0903747:1:5.json" {
			t.Fatalf("path = %q, want episode stream path", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"streams":[]}`))
	}))
	defer server.Close()

	client := NewTorrentioClientWithBaseURL(server.URL)
	streams, err := client.EpisodeStreams("tt0903747", 1, 5)
	if err != nil {
		t.Fatalf("EpisodeStreams returned error: %v", err)
	}
	if len(streams) != 0 {
		t.Fatalf("len(streams) = %d, want 0", len(streams))
	}
}

func TestTorrentioErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewTorrentioClientWithBaseURL(server.URL)
	_, err := client.MovieStreams("tt0000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestTorrentioInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewTorrentioClientWithBaseURL(server.URL)
	_, err := client.MovieStreams("tt1877830")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want ErrInvalidResponse", err)
	}
}
