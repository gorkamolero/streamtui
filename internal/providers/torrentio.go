package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"streamtui/internal/domain"
)

type TorrentioClient struct {
	baseURL string
	http    *http.Client
}

func NewTorrentioClient() TorrentioClient {
	return TorrentioClient{
		baseURL: "https://torrentio.strem.fun",
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func NewTorrentioClientWithBaseURL(baseURL string) TorrentioClient {
	client := NewTorrentioClient()
	client.baseURL = strings.TrimRight(baseURL, "/")
	return client
}

func (client TorrentioClient) MovieStreams(imdbID string) ([]domain.StreamSource, error) {
	return client.fetchStreams(fmt.Sprintf("/stream/movie/%s.json", imdbID))
}

func (client TorrentioClient) EpisodeStreams(imdbID string, season uint16, episode uint16) ([]domain.StreamSource, error) {
	return client.fetchStreams(fmt.Sprintf("/stream/series/%s:%d:%d.json", imdbID, season, episode))
}

func (client TorrentioClient) fetchStreams(endpoint string) ([]domain.StreamSource, error) {
	request, err := http.NewRequest(http.MethodGet, client.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "streamtui/1.0")

	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	switch {
	case response.StatusCode == http.StatusOK:
	case response.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case response.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case response.StatusCode >= 500:
		return nil, fmt.Errorf("provider server error: %d", response.StatusCode)
	default:
		return nil, fmt.Errorf("provider http error: %d", response.StatusCode)
	}

	var decoded torrentioResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	streams := make([]domain.StreamSource, 0, len(decoded.Streams))
	for _, stream := range decoded.Streams {
		streams = append(streams, stream.source())
	}
	return domain.RankStreams(streams), nil
}

type torrentioResponse struct {
	Streams []torrentioStream `json:"streams"`
}

type torrentioStream struct {
	Name      string  `json:"name"`
	Title     string  `json:"title"`
	InfoHash  string  `json:"infoHash"`
	FileIndex *uint32 `json:"fileIdx"`
}

func (stream torrentioStream) source() domain.StreamSource {
	return domain.StreamSource{
		Name:      stream.Name,
		Title:     stream.Title,
		InfoHash:  stream.InfoHash,
		FileIndex: stream.FileIndex,
		Seeds:     domain.ParseSeeds(stream.Title),
		Quality:   domain.ParseQuality(stream.Name),
		SizeBytes: domain.ParseSize(stream.Title),
	}
}
