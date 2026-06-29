package providers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"streamtui/internal/domain"
)

func TestSubtitleSearchMovieFiltersLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles/movie/tt0234215.json" {
			t.Fatalf("path = %q, want movie subtitle path", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"subtitles":[
			{"id":"1","url":"https://subs/1","lang":"eng"},
			{"id":"2|The.Matrix.1999","url":"https://subs/2","lang":"spa"}
		]}`))
	}))
	defer server.Close()

	client := NewSubtitleClientWithBaseURL(server.URL, t.TempDir())
	results, err := client.Search("0234215", "es")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Language != "spa" || results[0].LanguageName != "Spanish" {
		t.Fatalf("language = %q/%q, want Spanish", results[0].Language, results[0].LanguageName)
	}
	if results[0].Release != "The Matrix 1999" {
		t.Fatalf("release = %q, want The Matrix 1999", results[0].Release)
	}
}

func TestSubtitleSearchEpisodePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subtitles/series/tt0903747:1:5.json" {
			t.Fatalf("path = %q, want episode subtitle path", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"subtitles":[]}`))
	}))
	defer server.Close()

	client := NewSubtitleClientWithBaseURL(server.URL, t.TempDir())
	results, err := client.SearchEpisode("tt0903747", 1, 5, "")
	if err != nil {
		t.Fatalf("SearchEpisode returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
}

func TestSubtitleInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewSubtitleClientWithBaseURL(server.URL, t.TempDir())
	_, err := client.Search("tt0234215", "")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("error = %v, want ErrInvalidResponse", err)
	}
}

func TestSubtitleDownloadCachesWebVTT(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nHello"))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	client := NewSubtitleClientWithBaseURL(server.URL, cacheDir)
	subtitle := domainSubtitle("1", server.URL, "eng")

	content, err := client.Download(subtitle)
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if !strings.Contains(content, "WEBVTT") || !strings.Contains(content, "00:00:01.000") {
		t.Fatalf("content not converted: %q", content)
	}

	content, err = client.Download(subtitle)
	if err != nil {
		t.Fatalf("cached Download returned error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1 cache miss", requests)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "eng_1.vtt")); err != nil {
		t.Fatalf("cache file missing: %v", err)
	}
}

func domainSubtitle(id string, url string, language string) domain.SubtitleResult {
	return domain.SubtitleResult{ID: id, URL: url, Language: language}
}
