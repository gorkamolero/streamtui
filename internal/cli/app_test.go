package cli

import (
	"errors"
	"testing"
	"time"

	"streamtui/internal/domain"
	"streamtui/internal/providers"
)

type fakeMetadataProvider struct {
	search    []domain.SearchResult
	trending  []domain.SearchResult
	movie     domain.MovieDetail
	tv        domain.TVDetail
	searchErr error
}

func (provider fakeMetadataProvider) Search(query string) ([]domain.SearchResult, error) {
	if provider.searchErr != nil {
		return nil, provider.searchErr
	}
	return provider.search, nil
}

func (provider fakeMetadataProvider) Trending() ([]domain.SearchResult, error) {
	return provider.trending, nil
}

func (provider fakeMetadataProvider) MovieDetail(id uint64) (domain.MovieDetail, error) {
	return provider.movie, nil
}

func (provider fakeMetadataProvider) TVDetail(id uint64) (domain.TVDetail, error) {
	return provider.tv, nil
}

type fakeStreamProvider struct {
	movie   []domain.StreamSource
	episode []domain.StreamSource
	err     error
}

func (provider fakeStreamProvider) MovieStreams(imdbID string) ([]domain.StreamSource, error) {
	if provider.err != nil {
		return nil, provider.err
	}
	return provider.movie, nil
}

func (provider fakeStreamProvider) EpisodeStreams(imdbID string, season uint16, episode uint16) ([]domain.StreamSource, error) {
	if provider.err != nil {
		return nil, provider.err
	}
	return provider.episode, nil
}

type fakeSubtitleProvider struct {
	movie   []domain.SubtitleResult
	episode []domain.SubtitleResult
	err     error
}

func (provider fakeSubtitleProvider) Search(imdbID string, language string) ([]domain.SubtitleResult, error) {
	if provider.err != nil {
		return nil, provider.err
	}
	return provider.movie, nil
}

func (provider fakeSubtitleProvider) SearchEpisode(imdbID string, season uint16, episode uint16, language string) ([]domain.SubtitleResult, error) {
	if provider.err != nil {
		return nil, provider.err
	}
	return provider.episode, nil
}

func (provider fakeSubtitleProvider) Download(subtitle domain.SubtitleResult) (string, error) {
	if provider.err != nil {
		return "", provider.err
	}
	return "WEBVTT", nil
}

func (provider fakeSubtitleProvider) CachePath(subtitle domain.SubtitleResult) string {
	return "/cache/" + subtitle.Language + "_" + subtitle.ID + ".vtt"
}

type fakeDeviceProvider struct {
	devices []domain.CastDevice
	err     error
	calls   int
}

type fakePlaybackAdapter struct {
	requests []domain.PlaybackRequest
	err      error
}

func (adapter *fakePlaybackAdapter) PlayMagnet(request domain.PlaybackRequest) error {
	adapter.requests = append(adapter.requests, request)
	return adapter.err
}

type fakeControlAdapter struct {
	status        domain.PlaybackStatus
	err           error
	calls         []string
	seekCommand   domain.SeekCommand
	volumeCommand domain.VolumeCommand
}

func (adapter *fakeControlAdapter) Status(device string) (domain.PlaybackStatus, error) {
	adapter.calls = append(adapter.calls, "status:"+device)
	if adapter.err != nil {
		return domain.PlaybackStatus{}, adapter.err
	}
	return adapter.status, nil
}

func (adapter *fakeControlAdapter) Play(device string) error {
	adapter.calls = append(adapter.calls, "play:"+device)
	return adapter.err
}

func (adapter *fakeControlAdapter) Pause(device string) error {
	adapter.calls = append(adapter.calls, "pause:"+device)
	return adapter.err
}

func (adapter *fakeControlAdapter) Stop(device string) error {
	adapter.calls = append(adapter.calls, "stop:"+device)
	return adapter.err
}

func (adapter *fakeControlAdapter) Seek(device string, command domain.SeekCommand) error {
	adapter.calls = append(adapter.calls, "seek:"+device)
	adapter.seekCommand = command
	return adapter.err
}

func (adapter *fakeControlAdapter) Volume(device string, command domain.VolumeCommand) error {
	adapter.calls = append(adapter.calls, "volume:"+device)
	adapter.volumeCommand = command
	return adapter.err
}

type memorySessionStore struct {
	session domain.SessionMetadata
	err     error
	saves   int
}

func (store *memorySessionStore) Load() (domain.SessionMetadata, error) {
	if store.err != nil {
		return domain.SessionMetadata{}, store.err
	}
	return store.session, nil
}

func (store *memorySessionStore) Save(metadata domain.SessionMetadata) error {
	if store.err != nil {
		return store.err
	}
	store.session = metadata
	store.saves++
	return nil
}

func (provider *fakeDeviceProvider) Discover() ([]domain.CastDevice, error) {
	provider.calls++
	if provider.err != nil {
		return nil, provider.err
	}
	return provider.devices, nil
}

type memoryDeviceStore struct {
	cache domain.DeviceCache
	err   error
	saves int
}

func (store *memoryDeviceStore) Load() (domain.DeviceCache, error) {
	if store.err != nil {
		return domain.DeviceCache{}, store.err
	}
	return store.cache, nil
}

func (store *memoryDeviceStore) Save(cache domain.DeviceCache) error {
	if store.err != nil {
		return store.err
	}
	store.cache = cache
	store.saves++
	return nil
}

func TestExecuteSearchUsesMetadataProvider(t *testing.T) {
	app := App{Metadata: fakeMetadataProvider{
		search: []domain.SearchResult{
			{ID: 1, MediaType: domain.MediaMovie, Title: "Movie"},
			{ID: 2, MediaType: domain.MediaTV, Title: "Show"},
		},
	}}

	parsed, _, err := Parse([]string{"search", "batman", "--type", "movie", "--limit", "1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}

	results, ok := data.([]domain.SearchResult)
	if !ok {
		t.Fatalf("data type = %T, want []SearchResult", data)
	}
	if len(results) != 1 || results[0].Title != "Movie" {
		t.Fatalf("results = %#v, want one movie", results)
	}
}

func TestExecuteTrendingUsesMetadataProvider(t *testing.T) {
	app := App{Metadata: fakeMetadataProvider{
		trending: []domain.SearchResult{{ID: 1, MediaType: domain.MediaTV, Title: "Show"}},
	}}

	parsed, _, err := Parse([]string{"trending"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	results := data.([]domain.SearchResult)
	if len(results) != 1 || results[0].Title != "Show" {
		t.Fatalf("results = %#v, want Show", results)
	}
}

func TestExecuteInfoDispatchesTV(t *testing.T) {
	app := App{Metadata: fakeMetadataProvider{
		tv: domain.TVDetail{ID: 1396, IMDBID: "tt0903747", Name: "Breaking Bad"},
	}}

	parsed, _, err := Parse([]string{"info", "1396", "--type", "tv"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	detail := data.(domain.TVDetail)
	if detail.IMDBID != "tt0903747" {
		t.Fatalf("imdb_id = %q, want tt0903747", detail.IMDBID)
	}
}

func TestExecuteStreamsRanksFiltersAndIndexes(t *testing.T) {
	app := App{Streams: fakeStreamProvider{
		movie: []domain.StreamSource{
			{Name: "720", Quality: domain.Quality720p, Seeds: 500},
			{Name: "1080", Quality: domain.Quality1080p, Seeds: 10},
			{Name: "480", Quality: domain.Quality480p, Seeds: 999},
		},
	}}

	parsed, _, err := Parse([]string{"streams", "tt1877830", "--quality", "720p", "--limit", "2"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}

	streams := data.([]IndexedStream)
	if len(streams) != 2 {
		t.Fatalf("len(streams) = %d, want 2", len(streams))
	}
	if streams[0].Index != 0 || streams[0].Name != "1080" {
		t.Fatalf("first stream = %#v, want indexed 1080", streams[0])
	}
	if streams[1].Index != 1 || streams[1].Name != "720" {
		t.Fatalf("second stream = %#v, want indexed 720", streams[1])
	}
}

func TestExecuteStreamsHonorsSortOption(t *testing.T) {
	sizeSmall := uint64(1)
	sizeLarge := uint64(2)
	app := App{Streams: fakeStreamProvider{
		movie: []domain.StreamSource{
			{Name: "720-high-seeds", Quality: domain.Quality720p, Seeds: 500, SizeBytes: &sizeSmall},
			{Name: "1080-small", Quality: domain.Quality1080p, Seeds: 10, SizeBytes: &sizeSmall},
			{Name: "1080-large", Quality: domain.Quality1080p, Seeds: 20, SizeBytes: &sizeLarge},
		},
	}}

	parsed, _, err := Parse([]string{"streams", "tt1877830", "--sort", "seeds", "--limit", "1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	streams := data.([]IndexedStream)
	if streams[0].Name != "720-high-seeds" {
		t.Fatalf("stream = %q, want highest seed stream", streams[0].Name)
	}

	parsed, _, err = Parse([]string{"streams", "tt1877830", "--sort", "size", "--limit", "1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err = app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	streams = data.([]IndexedStream)
	if streams[0].Name != "1080-large" {
		t.Fatalf("stream = %q, want largest stream", streams[0].Name)
	}
}

func TestExecuteStreamsRejectsUnknownSort(t *testing.T) {
	app := App{Streams: fakeStreamProvider{movie: []domain.StreamSource{{Name: "1080", Quality: domain.Quality1080p}}}}

	parsed, _, err := Parse([]string{"streams", "tt1877830", "--sort", "random"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}
	if code != ExitInvalidArgs {
		t.Fatalf("code = %d, want %d", code, ExitInvalidArgs)
	}
}

func TestExecuteProviderErrorMapsToExitCode(t *testing.T) {
	app := App{Metadata: fakeMetadataProvider{searchErr: providers.ErrRateLimited}}
	parsed, _, err := Parse([]string{"search", "batman"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	_, code, err := app.Execute(parsed)
	if !errors.Is(err, providers.ErrRateLimited) {
		t.Fatalf("error = %v, want rate limited", err)
	}
	if code != ExitNetwork {
		t.Fatalf("code = %d, want %d", code, ExitNetwork)
	}
}

func TestExecuteSubtitlesFiltersSortsAndLimits(t *testing.T) {
	app := App{Subtitles: fakeSubtitleProvider{
		movie: []domain.SubtitleResult{
			{ID: "ai", Language: "spa", FromTrusted: true, AITranslated: true, Downloads: 1000},
			{ID: "trusted", Language: "spa", FromTrusted: true, Downloads: 100},
			{ID: "untrusted", Language: "spa", Downloads: 5000},
		},
	}}

	parsed, _, err := Parse([]string{"subtitles", "tt1877830", "--lang", "es", "--trusted", "--limit", "1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}

	results := data.([]domain.SubtitleResult)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].ID != "trusted" {
		t.Fatalf("subtitle = %q, want trusted", results[0].ID)
	}
}

func TestExecuteSubtitlesRequiresSeasonAndEpisodeTogether(t *testing.T) {
	app := App{Subtitles: fakeSubtitleProvider{}}
	parsed, _, err := Parse([]string{"subtitles", "tt0903747", "--season", "1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	_, code, err := app.Execute(parsed)
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}
	if code != ExitInvalidArgs {
		t.Fatalf("code = %d, want %d", code, ExitInvalidArgs)
	}
}

func TestDevicesListUsesFreshCache(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	deviceProvider := &fakeDeviceProvider{devices: []domain.CastDevice{{Name: "Network TV"}}}
	store := &memoryDeviceStore{cache: domain.DeviceCache{
		UpdatedAt: now,
		Devices:   []domain.CachedDevice{{CastDevice: domain.CastDevice{Name: "Cached TV"}}},
	}}
	app := App{
		Devices:        deviceProvider,
		DeviceStore:    store,
		Now:            func() time.Time { return now },
		DeviceCacheTTL: time.Hour,
	}

	parsed, _, err := Parse([]string{"devices", "list"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	devices := data.([]domain.CastDevice)
	if len(devices) != 1 || devices[0].Name != "Cached TV" {
		t.Fatalf("devices = %#v, want cached device", devices)
	}
	if deviceProvider.calls != 0 {
		t.Fatalf("discover calls = %d, want 0", deviceProvider.calls)
	}
}

func TestDevicesRefreshDiscoversAndSaves(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	deviceProvider := &fakeDeviceProvider{devices: []domain.CastDevice{{Name: "Living Room TV", Address: "192.168.1.36"}}}
	store := &memoryDeviceStore{}
	app := App{
		Devices:     deviceProvider,
		DeviceStore: store,
		Now:         func() time.Time { return now },
	}

	parsed, _, err := Parse([]string{"devices", "refresh"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	devices := data.([]domain.CastDevice)
	if len(devices) != 1 || devices[0].Name != "Living Room TV" {
		t.Fatalf("devices = %#v, want discovered device", devices)
	}
	if store.saves != 1 {
		t.Fatalf("saves = %d, want 1", store.saves)
	}
	if !store.cache.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at = %v, want %v", store.cache.UpdatedAt, now)
	}
}

func TestDevicesRefreshMapsProviderNotFoundToDeviceNotFound(t *testing.T) {
	app := App{
		Devices:     &fakeDeviceProvider{err: providers.ErrNotFound},
		DeviceStore: &memoryDeviceStore{},
	}

	parsed, _, err := Parse([]string{"devices", "refresh"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}
	if code != ExitDeviceNotFound {
		t.Fatalf("code = %d, want %d", code, ExitDeviceNotFound)
	}
	if err.Error() != "no Chromecast devices found" {
		t.Fatalf("error = %q, want no Chromecast devices found", err.Error())
	}
}

func TestDevicesDefaultSetAndGet(t *testing.T) {
	store := &memoryDeviceStore{}
	app := App{DeviceStore: store}

	parsed, _, err := Parse([]string{"devices", "default", "set", "Living Room TV"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if data.(DefaultDeviceResponse).Device != "Living Room TV" {
		t.Fatalf("set response = %#v", data)
	}

	parsed, _, err = Parse([]string{"devices", "default", "get"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err = app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if data.(DefaultDeviceResponse).Device != "Living Room TV" {
		t.Fatalf("get response = %#v", data)
	}
}

func TestCastMagnetVLCStartsPlayback(t *testing.T) {
	playback := &fakePlaybackAdapter{}
	app := App{Playback: playback}

	parsed, _, err := Parse([]string{"cast-magnet", "magnet:?xt=urn:btih:abc", "--vlc", "--file-idx", "2"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if len(playback.requests) != 1 {
		t.Fatalf("playback requests = %d, want 1", len(playback.requests))
	}
	if !playback.requests[0].VLC || playback.requests[0].FileIndex != 2 {
		t.Fatalf("request = %#v, want VLC file index 2", playback.requests[0])
	}
	if data.(domain.PlaybackStart).Player != "VLC" {
		t.Fatalf("response = %#v, want VLC", data)
	}
}

func TestCastMagnetChromecastResolvesDeviceAndMarksUsed(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	playback := &fakePlaybackAdapter{}
	store := &memoryDeviceStore{cache: domain.DeviceCache{
		UpdatedAt: now,
		Devices:   []domain.CachedDevice{{CastDevice: domain.CastDevice{Name: "Living Room TV", Address: "192.168.1.36"}}},
	}}
	app := App{
		Playback:       playback,
		DeviceStore:    store,
		Now:            func() time.Time { return now },
		DeviceCacheTTL: time.Hour,
	}

	parsed, _, err := Parse([]string{"cast-magnet", "magnet:?xt=urn:btih:abc", "--device", "Living Room TV"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if playback.requests[0].Device != "Living Room TV" {
		t.Fatalf("device = %q, want Living Room TV", playback.requests[0].Device)
	}
	if store.cache.Devices[0].LastUsed == nil {
		t.Fatalf("last_used was not set")
	}
}

func TestCastMagnetExplicitDeviceDoesNotRequireDiscovery(t *testing.T) {
	playback := &fakePlaybackAdapter{}
	deviceProvider := &fakeDeviceProvider{}
	store := &memoryDeviceStore{}
	app := App{
		Playback:    playback,
		Devices:     deviceProvider,
		DeviceStore: store,
	}

	parsed, _, err := Parse([]string{"cast-magnet", "magnet:?xt=urn:btih:abc", "--device", "Living Room TV"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if deviceProvider.calls != 0 {
		t.Fatalf("discover calls = %d, want 0", deviceProvider.calls)
	}
	if playback.requests[0].Device != "Living Room TV" {
		t.Fatalf("device = %q, want Living Room TV", playback.requests[0].Device)
	}
}

func TestCastSelectsStreamDownloadsSubtitleAndStartsPlayback(t *testing.T) {
	fileIndex := uint32(3)
	playback := &fakePlaybackAdapter{}
	app := App{
		Streams: fakeStreamProvider{movie: []domain.StreamSource{
			{Name: "720", InfoHash: "aaa", Quality: domain.Quality720p, Seeds: 50},
			{Name: "1080", InfoHash: "bbb", FileIndex: &fileIndex, Quality: domain.Quality1080p, Seeds: 10},
		}},
		Subtitles: fakeSubtitleProvider{movie: []domain.SubtitleResult{{ID: "sub1", Language: "spa", FromTrusted: true}}},
		Playback:  playback,
	}

	parsed, _, err := Parse([]string{"cast", "tt1877830", "--vlc", "--quality", "1080p", "--lang", "es"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if len(playback.requests) != 1 {
		t.Fatalf("playback requests = %d, want 1", len(playback.requests))
	}
	request := playback.requests[0]
	if request.Magnet != "magnet:?xt=urn:btih:bbb&dn=tt1877830" {
		t.Fatalf("magnet = %q, want selected 1080 stream", request.Magnet)
	}
	if request.FileIndex != fileIndex {
		t.Fatalf("file index = %d, want %d", request.FileIndex, fileIndex)
	}
	if request.SubtitleFile != "/cache/spa_sub1.vtt" {
		t.Fatalf("subtitle file = %q, want cache path", request.SubtitleFile)
	}
	if data.(domain.PlaybackStart).Status != "playing" {
		t.Fatalf("response = %#v", data)
	}
}

func TestCastQualityPrefersExactQualityOverHigherQuality(t *testing.T) {
	playback := &fakePlaybackAdapter{}
	app := App{
		Streams: fakeStreamProvider{movie: []domain.StreamSource{
			{Name: "4k", InfoHash: "aaa", Quality: domain.Quality4K, Seeds: 500},
			{Name: "1080", InfoHash: "bbb", Quality: domain.Quality1080p, Seeds: 10},
		}},
		Playback: playback,
	}

	parsed, _, err := Parse([]string{"cast", "tt1877830", "--vlc", "--quality", "1080p"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if playback.requests[0].Magnet != "magnet:?xt=urn:btih:bbb&dn=tt1877830" {
		t.Fatalf("magnet = %q, want exact 1080p stream", playback.requests[0].Magnet)
	}
}

func TestStatusUsesSessionDevice(t *testing.T) {
	control := &fakeControlAdapter{status: domain.PlaybackStatus{State: domain.PlaybackPlaying}}
	app := App{
		Control:      control,
		SessionStore: &memorySessionStore{session: domain.SessionMetadata{Device: "Living Room TV", Title: "The Batman"}},
	}

	parsed, _, err := Parse([]string{"status"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if control.calls[0] != "status:Living Room TV" {
		t.Fatalf("call = %q, want status:Living Room TV", control.calls[0])
	}
	if data.(domain.PlaybackStatus).Device != "Living Room TV" {
		t.Fatalf("status = %#v, want device filled", data)
	}
	if data.(domain.PlaybackStatus).Title != "The Batman" {
		t.Fatalf("status = %#v, want session title filled", data)
	}
}

func TestPlaybackActionsUseGlobalDevice(t *testing.T) {
	control := &fakeControlAdapter{}
	app := App{Control: control}
	parsed, _, err := Parse([]string{"--device", "Office TV", "pause"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if control.calls[0] != "pause:Office TV" {
		t.Fatalf("call = %q, want pause:Office TV", control.calls[0])
	}
	if data.(domain.ControlResult).Action != "pause" {
		t.Fatalf("result = %#v", data)
	}
}

func TestPlaybackActionMapsControlMissingDevice(t *testing.T) {
	control := &fakeControlAdapter{err: providers.ErrNotFound}
	app := App{Control: control}
	parsed, _, err := Parse([]string{"--device", "Missing TV", "play"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}
	if code != ExitDeviceNotFound {
		t.Fatalf("code = %d, want %d", code, ExitDeviceNotFound)
	}
}

func TestSeekParsesRelativeCommand(t *testing.T) {
	control := &fakeControlAdapter{}
	app := App{
		Control:      control,
		SessionStore: &memorySessionStore{session: domain.SessionMetadata{Device: "TV"}},
	}
	parsed, _, err := Parse([]string{"seek", "+30"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if control.seekCommand.Action != "ffwd" || control.seekCommand.Value != 30 {
		t.Fatalf("seek command = %#v, want ffwd 30", control.seekCommand)
	}
}

func TestVolumeParsesRelativeCommand(t *testing.T) {
	control := &fakeControlAdapter{}
	app := App{
		Control:      control,
		SessionStore: &memorySessionStore{session: domain.SessionMetadata{Device: "TV"}},
	}
	parsed, _, err := Parse([]string{"volume", "-10"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if control.volumeCommand.Action != "volumedown" {
		t.Fatalf("volume command = %#v, want volumedown", control.volumeCommand)
	}
}

func TestCastSavesSessionMetadata(t *testing.T) {
	playback := &fakePlaybackAdapter{}
	sessionStore := &memorySessionStore{}
	app := App{
		Streams:      fakeStreamProvider{movie: []domain.StreamSource{{Name: "1080", InfoHash: "bbb", Quality: domain.Quality1080p, Seeds: 10}}},
		Playback:     playback,
		SessionStore: sessionStore,
	}

	parsed, _, err := Parse([]string{"cast", "tt1877830", "--vlc"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if sessionStore.saves != 1 {
		t.Fatalf("session saves = %d, want 1", sessionStore.saves)
	}
	if sessionStore.session.Title != "tt1877830" {
		t.Fatalf("session = %#v, want title tt1877830", sessionStore.session)
	}
}

func TestAgentFirstPlaySearchesSelectsSubtitleAndStartsPlayback(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	playback := &fakePlaybackAdapter{}
	sessionStore := &memorySessionStore{}
	store := &memoryDeviceStore{cache: domain.DeviceCache{
		UpdatedAt: now,
		Devices:   []domain.CachedDevice{{CastDevice: domain.CastDevice{Name: "Living Room TV", Address: "192.168.1.36"}}},
	}}
	app := App{
		Metadata: fakeMetadataProvider{
			search: []domain.SearchResult{
				{ID: 9, MediaType: domain.MediaTV, Title: "The Batman Show"},
				{ID: 414906, MediaType: domain.MediaMovie, Title: "The Batman"},
			},
			movie: domain.MovieDetail{ID: 414906, IMDBID: "tt1877830", Title: "The Batman"},
		},
		Streams: fakeStreamProvider{movie: []domain.StreamSource{
			{Name: "720", InfoHash: "aaa", Quality: domain.Quality720p, Seeds: 50},
			{Name: "1080", InfoHash: "bbb", Quality: domain.Quality1080p, Seeds: 10},
		}},
		Subtitles: fakeSubtitleProvider{movie: []domain.SubtitleResult{
			{ID: "weak", Language: "spa", Downloads: 5000},
			{ID: "trusted", Language: "spa", FromTrusted: true, Downloads: 100},
		}},
		Playback:       playback,
		DeviceStore:    store,
		SessionStore:   sessionStore,
		Now:            func() time.Time { return now },
		DeviceCacheTTL: time.Hour,
	}

	parsed, _, err := Parse([]string{"play", "the batman", "--lang", "es", "--device", "Living Room TV"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if len(playback.requests) != 1 {
		t.Fatalf("playback requests = %d, want 1", len(playback.requests))
	}
	request := playback.requests[0]
	if request.Device != "Living Room TV" {
		t.Fatalf("device = %q, want Living Room TV", request.Device)
	}
	if request.Magnet != "magnet:?xt=urn:btih:bbb&dn=tt1877830" {
		t.Fatalf("magnet = %q, want selected 1080 stream", request.Magnet)
	}
	if request.SubtitleFile != "/cache/spa_trusted.vtt" {
		t.Fatalf("subtitle file = %q, want trusted Spanish subtitle", request.SubtitleFile)
	}
	if request.Title != "The Batman" {
		t.Fatalf("title = %q, want The Batman", request.Title)
	}
	if data.(domain.PlaybackStart).Title != "The Batman" {
		t.Fatalf("response = %#v, want title", data)
	}
	if sessionStore.session.Title != "The Batman" || sessionStore.session.Device != "Living Room TV" {
		t.Fatalf("session = %#v, want title and device", sessionStore.session)
	}
}

func TestAgentFirstPlaySupportsVLCWithoutDeviceResolution(t *testing.T) {
	playback := &fakePlaybackAdapter{}
	app := App{
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
	}

	parsed, _, err := Parse([]string{"play", "the batman", "--lang", "es", "--vlc"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	data, code, err := app.Execute(parsed)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if len(playback.requests) != 1 {
		t.Fatalf("playback requests = %d, want 1", len(playback.requests))
	}
	request := playback.requests[0]
	if !request.VLC {
		t.Fatalf("VLC = false, want true")
	}
	if request.Device != "" {
		t.Fatalf("device = %q, want empty for VLC", request.Device)
	}
	if data.(domain.PlaybackStart).Player != "VLC" {
		t.Fatalf("response = %#v, want VLC player", data)
	}
}

func TestBestPlayableSearchResultPrefersExactMovieTitle(t *testing.T) {
	results := []domain.SearchResult{
		{ID: 1, MediaType: domain.MediaTV, Title: "The Batman"},
		{ID: 2, MediaType: domain.MediaMovie, Title: "Batman Returns"},
		{ID: 3, MediaType: domain.MediaMovie, Title: "The Batman"},
	}

	result, ok := bestPlayableSearchResult("the batman", results)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if result.ID != 3 {
		t.Fatalf("result ID = %d, want exact movie ID 3", result.ID)
	}
}

func TestBestPlayableSearchResultFallsBackToFirstMovie(t *testing.T) {
	results := []domain.SearchResult{
		{ID: 1, MediaType: domain.MediaTV, Title: "Only TV"},
		{ID: 2, MediaType: domain.MediaMovie, Title: "Close Enough"},
		{ID: 3, MediaType: domain.MediaMovie, Title: "Another Movie"},
	}

	result, ok := bestPlayableSearchResult("missing exact title", results)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if result.ID != 2 {
		t.Fatalf("result ID = %d, want first movie ID 2", result.ID)
	}
}
