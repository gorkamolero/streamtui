package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"streamtui/internal/domain"
)

var (
	ErrNotFound        = errors.New("provider resource not found")
	ErrRateLimited     = errors.New("provider rate limited")
	ErrInvalidResponse = errors.New("provider invalid response")
)

type TMDBClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewTMDBClient(apiKey string) TMDBClient {
	return TMDBClient{
		baseURL: "https://api.themoviedb.org/3",
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func NewTMDBClientWithBaseURL(apiKey string, baseURL string) TMDBClient {
	client := NewTMDBClient(apiKey)
	client.baseURL = strings.TrimRight(baseURL, "/")
	return client
}

func (client TMDBClient) Search(query string) ([]domain.SearchResult, error) {
	endpoint := "/search/multi?query=" + url.QueryEscape(query) + "&page=1"
	var response tmdbSearchResponse
	if err := client.get(endpoint, &response); err != nil {
		return nil, err
	}
	return response.results(), nil
}

func (client TMDBClient) Trending() ([]domain.SearchResult, error) {
	var response tmdbSearchResponse
	if err := client.get("/trending/all/week", &response); err != nil {
		return nil, err
	}
	return response.results(), nil
}

func (client TMDBClient) MovieDetail(id uint64) (domain.MovieDetail, error) {
	var response tmdbMovieResponse
	if err := client.get(fmt.Sprintf("/movie/%d?append_to_response=external_ids", id), &response); err != nil {
		return domain.MovieDetail{}, err
	}
	return response.detail(), nil
}

func (client TMDBClient) TVDetail(id uint64) (domain.TVDetail, error) {
	var response tmdbTVResponse
	if err := client.get(fmt.Sprintf("/tv/%d?append_to_response=external_ids", id), &response); err != nil {
		return domain.TVDetail{}, err
	}
	return response.detail(), nil
}

func (client TMDBClient) get(endpoint string, target any) error {
	requestURL := client.baseURL + endpoint
	if len(client.apiKey) < 64 {
		separator := "?"
		if strings.Contains(endpoint, "?") {
			separator = "&"
		}
		requestURL += separator + "api_key=" + url.QueryEscape(client.apiKey)
	}

	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if len(client.apiKey) >= 64 {
		request.Header.Set("Authorization", "Bearer "+client.apiKey)
	}

	response, err := client.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch {
	case response.StatusCode == http.StatusOK:
		if err := json.NewDecoder(response.Body).Decode(target); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}
		return nil
	case response.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case response.StatusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	case response.StatusCode >= 500:
		return fmt.Errorf("provider server error: %d", response.StatusCode)
	default:
		return fmt.Errorf("provider http error: %d", response.StatusCode)
	}
}

type tmdbSearchResponse struct {
	Results []tmdbSearchItem `json:"results"`
}

type tmdbSearchItem struct {
	ID           uint64           `json:"id"`
	MediaType    string           `json:"media_type"`
	Title        string           `json:"title"`
	Name         string           `json:"name"`
	ReleaseDate  string           `json:"release_date"`
	FirstAirDate string           `json:"first_air_date"`
	Overview     string           `json:"overview"`
	PosterPath   *string          `json:"poster_path"`
	VoteAverage  float32          `json:"vote_average"`
	ExternalIDs  *tmdbExternalIDs `json:"external_ids"`
}

type tmdbExternalIDs struct {
	IMDBID string `json:"imdb_id"`
}

func (response tmdbSearchResponse) results() []domain.SearchResult {
	results := make([]domain.SearchResult, 0, len(response.Results))
	for _, item := range response.Results {
		result, ok := item.result()
		if ok {
			results = append(results, result)
		}
	}
	return results
}

func (item tmdbSearchItem) result() (domain.SearchResult, bool) {
	switch item.MediaType {
	case "movie":
		return domain.SearchResult{
			ID:          item.ID,
			MediaType:   domain.MediaMovie,
			Title:       item.Title,
			Year:        yearFromDate(item.ReleaseDate),
			Overview:    item.Overview,
			PosterPath:  item.PosterPath,
			VoteAverage: item.VoteAverage,
			IMDBID:      imdbID(item.ExternalIDs),
		}, true
	case "tv":
		return domain.SearchResult{
			ID:          item.ID,
			MediaType:   domain.MediaTV,
			Title:       item.Name,
			Year:        yearFromDate(item.FirstAirDate),
			Overview:    item.Overview,
			PosterPath:  item.PosterPath,
			VoteAverage: item.VoteAverage,
			IMDBID:      imdbID(item.ExternalIDs),
		}, true
	default:
		return domain.SearchResult{}, false
	}
}

type tmdbMovieResponse struct {
	ID           uint64          `json:"id"`
	ExternalIDs  tmdbExternalIDs `json:"external_ids"`
	Title        string          `json:"title"`
	ReleaseDate  string          `json:"release_date"`
	Runtime      uint32          `json:"runtime"`
	Genres       []tmdbGenre     `json:"genres"`
	Overview     string          `json:"overview"`
	VoteAverage  float32         `json:"vote_average"`
	PosterPath   *string         `json:"poster_path"`
	BackdropPath *string         `json:"backdrop_path"`
}

type tmdbTVResponse struct {
	ID           uint64              `json:"id"`
	ExternalIDs  tmdbExternalIDs     `json:"external_ids"`
	Name         string              `json:"name"`
	FirstAirDate string              `json:"first_air_date"`
	Seasons      []tmdbSeasonSummary `json:"seasons"`
	Genres       []tmdbGenre         `json:"genres"`
	Overview     string              `json:"overview"`
	VoteAverage  float32             `json:"vote_average"`
	PosterPath   *string             `json:"poster_path"`
	BackdropPath *string             `json:"backdrop_path"`
}

type tmdbGenre struct {
	Name string `json:"name"`
}

type tmdbSeasonSummary struct {
	SeasonNumber uint8   `json:"season_number"`
	EpisodeCount uint16  `json:"episode_count"`
	Name         *string `json:"name"`
	AirDate      *string `json:"air_date"`
}

func (response tmdbMovieResponse) detail() domain.MovieDetail {
	return domain.MovieDetail{
		ID:           response.ID,
		IMDBID:       response.ExternalIDs.IMDBID,
		Title:        response.Title,
		Year:         valueOrZero(yearFromDate(response.ReleaseDate)),
		Runtime:      response.Runtime,
		Genres:       genreNames(response.Genres),
		Overview:     response.Overview,
		VoteAverage:  response.VoteAverage,
		PosterPath:   response.PosterPath,
		BackdropPath: response.BackdropPath,
	}
}

func (response tmdbTVResponse) detail() domain.TVDetail {
	seasons := make([]domain.SeasonSummary, 0, len(response.Seasons))
	for _, season := range response.Seasons {
		seasons = append(seasons, domain.SeasonSummary{
			SeasonNumber: season.SeasonNumber,
			EpisodeCount: season.EpisodeCount,
			Name:         season.Name,
			AirDate:      season.AirDate,
		})
	}

	return domain.TVDetail{
		ID:           response.ID,
		IMDBID:       response.ExternalIDs.IMDBID,
		Name:         response.Name,
		Year:         valueOrZero(yearFromDate(response.FirstAirDate)),
		Seasons:      seasons,
		Genres:       genreNames(response.Genres),
		Overview:     response.Overview,
		VoteAverage:  response.VoteAverage,
		PosterPath:   response.PosterPath,
		BackdropPath: response.BackdropPath,
	}
}

func genreNames(genres []tmdbGenre) []string {
	names := make([]string, 0, len(genres))
	for _, genre := range genres {
		names = append(names, genre.Name)
	}
	return names
}

func yearFromDate(date string) *uint16 {
	if len(date) < 4 {
		return nil
	}
	year, err := strconv.ParseUint(date[:4], 10, 16)
	if err != nil {
		return nil
	}
	value := uint16(year)
	return &value
}

func valueOrZero(value *uint16) uint16 {
	if value == nil {
		return 0
	}
	return *value
}

func imdbID(ids *tmdbExternalIDs) string {
	if ids == nil {
		return ""
	}
	return ids.IMDBID
}
