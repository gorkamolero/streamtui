package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"streamtui/internal/domain"
)

type SubtitleClient struct {
	baseURL  string
	cacheDir string
	http     *http.Client
}

func NewSubtitleClient() SubtitleClient {
	return SubtitleClient{
		baseURL:  "https://opensubtitles-v3.strem.io",
		cacheDir: defaultSubtitleCacheDir(),
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

func NewSubtitleClientWithBaseURL(baseURL string, cacheDir string) SubtitleClient {
	client := NewSubtitleClient()
	client.baseURL = strings.TrimRight(baseURL, "/")
	if cacheDir != "" {
		client.cacheDir = cacheDir
	}
	return client
}

func (client SubtitleClient) Search(imdbID string, language string) ([]domain.SubtitleResult, error) {
	return client.fetch(fmt.Sprintf("/subtitles/movie/%s.json", normalizeIMDBID(imdbID)), language)
}

func (client SubtitleClient) SearchEpisode(imdbID string, season uint16, episode uint16, language string) ([]domain.SubtitleResult, error) {
	return client.fetch(fmt.Sprintf("/subtitles/series/%s:%d:%d.json", normalizeIMDBID(imdbID), season, episode), language)
}

func (client SubtitleClient) Download(subtitle domain.SubtitleResult) (string, error) {
	cachePath := client.CachePath(subtitle)
	if content, err := os.ReadFile(cachePath); err == nil {
		return string(content), nil
	}

	response, err := client.http.Get(subtitle.URL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("subtitle download http error: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	webvtt := domain.SRTToWebVTT(string(body))

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(cachePath, []byte(webvtt), 0o644); err != nil {
		return "", err
	}
	return webvtt, nil
}

func (client SubtitleClient) CachePath(subtitle domain.SubtitleResult) string {
	return filepath.Join(client.cacheDir, fmt.Sprintf("%s_%s.vtt", subtitle.Language, subtitle.ID))
}

func (client SubtitleClient) fetch(endpoint string, language string) ([]domain.SubtitleResult, error) {
	response, err := client.http.Get(client.baseURL + endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subtitle provider http error: %d", response.StatusCode)
	}

	var decoded stremioSubtitleResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	languages := domain.NormalizeLanguages(language)
	results := make([]domain.SubtitleResult, 0, len(decoded.Subtitles))
	for _, subtitle := range decoded.Subtitles {
		normalized := domain.NormalizeLanguage(subtitle.Language)
		if len(languages) > 0 && !languageMatches(normalized, languages) {
			continue
		}
		results = append(results, domain.SubtitleResult{
			ID:              subtitle.ID,
			URL:             subtitle.URL,
			Language:        normalized,
			LanguageName:    domain.LanguageName(normalized),
			Release:         releaseFromSubtitleID(subtitle.ID),
			Format:          domain.SubFormatSRT,
			Downloads:       0,
			FromTrusted:     true,
			HearingImpaired: false,
			AITranslated:    false,
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].TrustScore() > results[j].TrustScore()
	})
	return results, nil
}

type stremioSubtitleResponse struct {
	Subtitles []stremioSubtitle `json:"subtitles"`
}

type stremioSubtitle struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Language string `json:"lang"`
}

func languageMatches(language string, targets []string) bool {
	for _, target := range targets {
		if language == target || strings.HasPrefix(language, target) || strings.HasPrefix(target, language) {
			return true
		}
	}
	return false
}

func normalizeIMDBID(imdbID string) string {
	if strings.HasPrefix(imdbID, "tt") {
		return imdbID
	}
	return "tt" + imdbID
}

func releaseFromSubtitleID(id string) string {
	if index := strings.Index(id, "|"); index >= 0 && index+1 < len(id) {
		release := strings.ReplaceAll(id[index+1:], ".", " ")
		return truncateString(strings.TrimSpace(release), 50)
	}
	if strings.Contains(id, ".") || strings.Contains(id, "-") {
		return truncateString(strings.TrimSpace(strings.ReplaceAll(id, ".", " ")), 50)
	}
	return "OpenSubtitles"
}

func truncateString(value string, length int) string {
	if len(value) <= length {
		return value
	}
	return value[:length]
}

func defaultSubtitleCacheDir() string {
	if cacheDir, ok := os.LookupEnv("XDG_CACHE_HOME"); ok && cacheDir != "" {
		return filepath.Join(cacheDir, "streamtui", "subtitles")
	}
	if home, ok := os.LookupEnv("HOME"); ok && home != "" {
		return filepath.Join(home, ".cache", "streamtui", "subtitles")
	}
	return filepath.Join(os.TempDir(), "streamtui", "subtitles")
}
