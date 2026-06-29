package providers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"streamtui/internal/domain"
)

func TestTMDBSearchParsesMoviesAndTV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/multi" {
			t.Fatalf("path = %q, want /search/multi", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "batman" {
			t.Fatalf("query = %q, want batman", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": 1, "media_type": "movie", "title": "The Batman", "release_date": "2022-03-04", "overview": "Movie", "vote_average": 7.8},
				{"id": 2, "media_type": "tv", "name": "Batman", "first_air_date": "1992-09-05", "overview": "TV", "vote_average": 8.5},
				{"id": 3, "media_type": "person", "name": "Actor"}
			]
		}`))
	}))
	defer server.Close()

	client := NewTMDBClientWithBaseURL("legacy-key", server.URL)
	results, err := client.Search("batman")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Title != "The Batman" || results[0].MediaType != domain.MediaMovie {
		t.Fatalf("unexpected movie result: %#v", results[0])
	}
	if results[0].Year == nil || *results[0].Year != 2022 {
		t.Fatalf("movie year = %v, want 2022", results[0].Year)
	}
	if results[1].Title != "Batman" || results[1].MediaType != domain.MediaTV {
		t.Fatalf("unexpected tv result: %#v", results[1])
	}
}

func TestTMDBMovieDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/414906" {
			t.Fatalf("path = %q, want /movie/414906", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"id": 414906,
			"external_ids": {"imdb_id": "tt1877830"},
			"title": "The Batman",
			"release_date": "2022-03-04",
			"runtime": 176,
			"genres": [{"name": "Crime"}, {"name": "Mystery"}],
			"overview": "Vengeance",
			"vote_average": 7.8
		}`))
	}))
	defer server.Close()

	client := NewTMDBClientWithBaseURL("legacy-key", server.URL)
	detail, err := client.MovieDetail(414906)
	if err != nil {
		t.Fatalf("MovieDetail returned error: %v", err)
	}
	if detail.IMDBID != "tt1877830" {
		t.Fatalf("imdb_id = %q, want tt1877830", detail.IMDBID)
	}
	if detail.Year != 2022 {
		t.Fatalf("year = %d, want 2022", detail.Year)
	}
	if len(detail.Genres) != 2 || detail.Genres[0] != "Crime" {
		t.Fatalf("genres = %#v, want Crime/Mystery", detail.Genres)
	}
}

func TestTMDBTVDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tv/1396" {
			t.Fatalf("path = %q, want /tv/1396", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"id": 1396,
			"external_ids": {"imdb_id": "tt0903747"},
			"name": "Breaking Bad",
			"first_air_date": "2008-01-20",
			"seasons": [{"season_number": 1, "episode_count": 7, "name": "Season 1"}],
			"genres": [{"name": "Drama"}],
			"overview": "Chemistry",
			"vote_average": 8.9
		}`))
	}))
	defer server.Close()

	client := NewTMDBClientWithBaseURL("legacy-key", server.URL)
	detail, err := client.TVDetail(1396)
	if err != nil {
		t.Fatalf("TVDetail returned error: %v", err)
	}
	if detail.IMDBID != "tt0903747" {
		t.Fatalf("imdb_id = %q, want tt0903747", detail.IMDBID)
	}
	if len(detail.Seasons) != 1 || detail.Seasons[0].EpisodeCount != 7 {
		t.Fatalf("seasons = %#v, want one season with seven episodes", detail.Seasons)
	}
}

func TestTMDBUsesBearerTokenForLongKeys(t *testing.T) {
	longToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3QifQ"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+longToken {
			t.Fatalf("authorization = %q, want bearer token", got)
		}
		if got := r.URL.Query().Get("api_key"); got != "" {
			t.Fatalf("api_key = %q, want empty", got)
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client := NewTMDBClientWithBaseURL(longToken, server.URL)
	if _, err := client.Search("test"); err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
}

func TestTMDBStatusErrors(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{status: http.StatusNotFound, want: ErrNotFound},
		{status: http.StatusTooManyRequests, want: ErrRateLimited},
	}

	for _, test := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.status)
		}))
		client := NewTMDBClientWithBaseURL("legacy-key", server.URL)
		_, err := client.Search("test")
		server.Close()

		if !errors.Is(err, test.want) {
			t.Fatalf("status %d error = %v, want %v", test.status, err, test.want)
		}
	}
}
