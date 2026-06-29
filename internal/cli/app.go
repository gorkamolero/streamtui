package cli

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"streamtui/internal/domain"
	"streamtui/internal/providers"
)

type MetadataProvider interface {
	Search(query string) ([]domain.SearchResult, error)
	Trending() ([]domain.SearchResult, error)
	MovieDetail(id uint64) (domain.MovieDetail, error)
	TVDetail(id uint64) (domain.TVDetail, error)
}

type StreamProvider interface {
	MovieStreams(imdbID string) ([]domain.StreamSource, error)
	EpisodeStreams(imdbID string, season uint16, episode uint16) ([]domain.StreamSource, error)
}

type SubtitleProvider interface {
	Search(imdbID string, language string) ([]domain.SubtitleResult, error)
	SearchEpisode(imdbID string, season uint16, episode uint16, language string) ([]domain.SubtitleResult, error)
}

type DeviceProvider interface {
	Discover() ([]domain.CastDevice, error)
}

type DeviceStore interface {
	Load() (domain.DeviceCache, error)
	Save(cache domain.DeviceCache) error
}

type PlaybackAdapter interface {
	PlayMagnet(request domain.PlaybackRequest) error
}

type ControlAdapter interface {
	Status(device string) (domain.PlaybackStatus, error)
	Play(device string) error
	Pause(device string) error
	Stop(device string) error
	Seek(device string, command domain.SeekCommand) error
	Volume(device string, command domain.VolumeCommand) error
}

type SessionStore interface {
	Load() (domain.SessionMetadata, error)
	Save(metadata domain.SessionMetadata) error
}

type SubtitleDownloader interface {
	Download(subtitle domain.SubtitleResult) (string, error)
}

type SubtitleCachePather interface {
	CachePath(subtitle domain.SubtitleResult) string
}

type App struct {
	Metadata       MetadataProvider
	Streams        StreamProvider
	Subtitles      SubtitleProvider
	Devices        DeviceProvider
	DeviceStore    DeviceStore
	Playback       PlaybackAdapter
	Control        ControlAdapter
	SessionStore   SessionStore
	Now            func() time.Time
	DeviceCacheTTL time.Duration
}

func (app App) Execute(input ParseResult) (any, ExitCode, error) {
	command := input.Command

	switch command.Name {
	case "search":
		return app.search(command)
	case "trending":
		return app.trending(command)
	case "info":
		return app.info(command)
	case "streams":
		return app.streams(command)
	case "subtitles":
		return app.subtitles(command)
	case "devices list":
		return app.devicesList()
	case "devices refresh":
		return app.devicesRefresh()
	case "devices default get":
		return app.devicesDefaultGet()
	case "devices default set":
		return app.devicesDefaultSet(command)
	case "cast-magnet":
		return app.castMagnet(input)
	case "cast":
		return app.cast(input)
	case "play":
		return app.playQuery(input)
	case "status":
		return app.status(input)
	case "playback play":
		return app.playbackAction(input, "play")
	case "pause":
		return app.playbackAction(input, "pause")
	case "stop":
		return app.playbackAction(input, "stop")
	case "seek":
		return app.seek(input)
	case "volume":
		return app.volume(input)
	default:
		return nil, ExitError, fmt.Errorf("command %q is parsed but not implemented yet", command.Name)
	}
}

func (app App) search(command ParsedCommand) (any, ExitCode, error) {
	if app.Metadata == nil {
		return nil, ExitError, fmt.Errorf("metadata provider is not configured")
	}

	results, err := app.Metadata.Search(command.Args[0])
	if err != nil {
		return nil, providerExit(err), err
	}

	if mediaType, ok := stringOption(command, "type"); ok {
		results = filterMediaType(results, mediaType)
	}

	limit, exit, err := limitOption(command, len(results))
	if err != nil {
		return nil, exit, err
	}
	results = limitSearchResults(results, limit)

	return results, ExitSuccess, nil
}

func (app App) trending(command ParsedCommand) (any, ExitCode, error) {
	if app.Metadata == nil {
		return nil, ExitError, fmt.Errorf("metadata provider is not configured")
	}

	results, err := app.Metadata.Trending()
	if err != nil {
		return nil, providerExit(err), err
	}

	if mediaType, ok := stringOption(command, "type"); ok {
		results = filterMediaType(results, mediaType)
	}

	limit, exit, err := limitOption(command, len(results))
	if err != nil {
		return nil, exit, err
	}
	results = limitSearchResults(results, limit)

	return results, ExitSuccess, nil
}

func (app App) info(command ParsedCommand) (any, ExitCode, error) {
	if app.Metadata == nil {
		return nil, ExitError, fmt.Errorf("metadata provider is not configured")
	}

	id, err := strconv.ParseUint(command.Args[0], 10, 64)
	if err != nil {
		return nil, ExitInvalidArgs, fmt.Errorf("info requires a numeric TMDB id")
	}

	if mediaType, ok := stringOption(command, "type"); ok && mediaType == "tv" {
		detail, err := app.Metadata.TVDetail(id)
		if err != nil {
			return nil, providerExit(err), err
		}
		return detail, ExitSuccess, nil
	}

	detail, err := app.Metadata.MovieDetail(id)
	if err != nil {
		return nil, providerExit(err), err
	}
	return detail, ExitSuccess, nil
}

func (app App) streams(command ParsedCommand) (any, ExitCode, error) {
	if app.Streams == nil {
		return nil, ExitError, fmt.Errorf("stream provider is not configured")
	}

	var streams []domain.StreamSource
	var err error
	seasonValue, hasSeason := stringOption(command, "season")
	episodeValue, hasEpisode := stringOption(command, "episode")

	switch {
	case hasSeason && hasEpisode:
		season, parseErr := parseUint16Option("season", seasonValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		episode, parseErr := parseUint16Option("episode", episodeValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		streams, err = app.Streams.EpisodeStreams(command.Args[0], season, episode)
	case hasSeason || hasEpisode:
		return nil, ExitInvalidArgs, fmt.Errorf("streams requires both season and episode for TV episodes")
	default:
		streams, err = app.Streams.MovieStreams(command.Args[0])
	}

	if err != nil {
		return nil, providerExit(err), err
	}
	if len(streams) == 0 {
		return nil, ExitNoResults, fmt.Errorf("no streams found")
	}

	if minimum, ok := stringOption(command, "quality"); ok {
		quality := domain.ParseQuality(minimum)
		if quality == domain.QualityUnknown {
			return nil, ExitInvalidArgs, fmt.Errorf("unknown quality %q", minimum)
		}
		streams = domain.FilterMinQuality(streams, quality)
	}

	sortCriterion, exit, err := streamSortOption(command)
	if err != nil {
		return nil, exit, err
	}
	streams = domain.SortStreams(streams, sortCriterion)
	limit, exit, err := limitOption(command, len(streams))
	if err != nil {
		return nil, exit, err
	}
	streams = limitStreams(streams, limit)

	if len(streams) == 0 {
		return nil, ExitNoResults, fmt.Errorf("no streams found")
	}

	return indexStreams(streams), ExitSuccess, nil
}

func (app App) subtitles(command ParsedCommand) (any, ExitCode, error) {
	if app.Subtitles == nil {
		return nil, ExitError, fmt.Errorf("subtitle provider is not configured")
	}

	language, _ := stringOption(command, "lang")
	seasonValue, hasSeason := stringOption(command, "season")
	episodeValue, hasEpisode := stringOption(command, "episode")

	var subtitles []domain.SubtitleResult
	var err error
	switch {
	case hasSeason && hasEpisode:
		season, parseErr := parseUint16Option("season", seasonValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		episode, parseErr := parseUint16Option("episode", episodeValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		subtitles, err = app.Subtitles.SearchEpisode(command.Args[0], season, episode, language)
	case hasSeason || hasEpisode:
		return nil, ExitInvalidArgs, fmt.Errorf("subtitles requires both season and episode for TV episodes")
	default:
		subtitles, err = app.Subtitles.Search(command.Args[0], language)
	}

	if err != nil {
		return nil, providerExit(err), err
	}

	subtitles = filterSubtitles(command, subtitles)
	sort.SliceStable(subtitles, func(i, j int) bool {
		return subtitles[i].TrustScore() > subtitles[j].TrustScore()
	})

	limit, exit, err := limitOption(command, len(subtitles))
	if err != nil {
		return nil, exit, err
	}
	if limit > len(subtitles) {
		limit = len(subtitles)
	}
	subtitles = subtitles[:limit]

	if len(subtitles) == 0 {
		return nil, ExitNoResults, fmt.Errorf("no subtitles found")
	}

	return subtitles, ExitSuccess, nil
}

func (app App) devicesList() (any, ExitCode, error) {
	if app.DeviceStore == nil {
		return nil, ExitError, fmt.Errorf("device store is not configured")
	}
	cache, err := app.DeviceStore.Load()
	if err != nil {
		return nil, ExitError, err
	}
	now := app.now()
	if cache.Fresh(now, app.deviceCacheTTL()) && len(cache.Devices) > 0 {
		return cache.DeviceList(), ExitSuccess, nil
	}
	return app.discoverAndSave(cache)
}

func (app App) devicesRefresh() (any, ExitCode, error) {
	if app.DeviceStore == nil {
		return nil, ExitError, fmt.Errorf("device store is not configured")
	}
	cache, err := app.DeviceStore.Load()
	if err != nil {
		return nil, ExitError, err
	}
	return app.discoverAndSave(cache)
}

func (app App) devicesDefaultGet() (any, ExitCode, error) {
	if app.DeviceStore == nil {
		return nil, ExitError, fmt.Errorf("device store is not configured")
	}
	cache, err := app.DeviceStore.Load()
	if err != nil {
		return nil, ExitError, err
	}
	return DefaultDeviceResponse{Device: cache.DefaultDevice}, ExitSuccess, nil
}

func (app App) devicesDefaultSet(command ParsedCommand) (any, ExitCode, error) {
	if app.DeviceStore == nil {
		return nil, ExitError, fmt.Errorf("device store is not configured")
	}
	cache, err := app.DeviceStore.Load()
	if err != nil {
		return nil, ExitError, err
	}
	cache.DefaultDevice = command.Args[0]
	if err := app.DeviceStore.Save(cache); err != nil {
		return nil, ExitError, err
	}
	return DefaultDeviceResponse{Device: cache.DefaultDevice}, ExitSuccess, nil
}

type DefaultDeviceResponse struct {
	Device string `json:"device"`
}

func (app App) discoverAndSave(cache domain.DeviceCache) (any, ExitCode, error) {
	if app.Devices == nil {
		return nil, ExitError, fmt.Errorf("device provider is not configured")
	}
	devices, err := app.Devices.Discover()
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			return nil, ExitDeviceNotFound, fmt.Errorf("no Chromecast devices found")
		}
		return nil, ExitNetwork, err
	}
	if len(devices) == 0 {
		return nil, ExitDeviceNotFound, fmt.Errorf("no Chromecast devices found")
	}
	cache = domain.UpdateDeviceCache(cache, devices, app.now())
	if err := app.DeviceStore.Save(cache); err != nil {
		return nil, ExitError, err
	}
	return devices, ExitSuccess, nil
}

func (app App) castMagnet(input ParseResult) (any, ExitCode, error) {
	command := input.Command
	if app.Playback == nil {
		return nil, ExitError, fmt.Errorf("playback adapter is not configured")
	}
	magnet := command.Args[0]
	if err := domain.ValidateTorrentID(magnet); err != nil {
		return nil, ExitInvalidArgs, err
	}

	fileIndex, exit, err := fileIndexOption(command)
	if err != nil {
		return nil, exit, err
	}
	request := domain.PlaybackRequest{
		Magnet:        magnet,
		VLC:           boolOption(command, "vlc"),
		SubtitleFile:  stringOptionValue(command, "subtitle_file"),
		SubtitleDelay: stringOptionValue(command, "subtitle_delay"),
		FileIndex:     fileIndex,
	}
	if exit, err := validateSubtitleDelay(request); err != nil {
		return nil, exit, err
	}

	var cache domain.DeviceCache
	if !request.VLC {
		device, loadedCache, exit, err := app.resolveDevice(effectiveDevice(input))
		if err != nil {
			return nil, exit, err
		}
		cache = loadedCache
		request.Device = device.Name
	}

	if err := app.Playback.PlayMagnet(request); err != nil {
		return nil, ExitPlaybackFailed, err
	}

	if !request.VLC {
		_ = app.markDeviceUsed(cache, request.Device)
	}
	_ = app.saveSession(domain.SessionMetadata{Device: request.Device, Title: request.Title, UpdatedAt: app.now()})

	return domain.PlaybackStart{
		Status:        "playing",
		Player:        playbackPlayer(request),
		Device:        request.Device,
		Magnet:        request.Magnet,
		SubtitleFile:  request.SubtitleFile,
		SubtitleDelay: request.SubtitleDelay,
	}, ExitSuccess, nil
}

func (app App) cast(input ParseResult) (any, ExitCode, error) {
	command := input.Command
	if app.Streams == nil {
		return nil, ExitError, fmt.Errorf("stream provider is not configured")
	}
	if app.Playback == nil {
		return nil, ExitError, fmt.Errorf("playback adapter is not configured")
	}

	streams, exit, err := app.fetchStreamsForCommand(command)
	if err != nil {
		return nil, exit, err
	}
	stream, exit, err := selectStreamForCast(command, streams)
	if err != nil {
		return nil, exit, err
	}

	request := domain.PlaybackRequest{
		Magnet:    stream.Magnet(command.Args[0]),
		VLC:       boolOption(command, "vlc"),
		FileIndex: streamFileIndex(stream),
		Title:     command.Args[0],
	}
	request.SubtitleDelay = stringOptionValue(command, "subtitle_delay")
	subtitleFile, exit, err := app.subtitleFileForCast(command)
	if err != nil {
		return nil, exit, err
	}
	request.SubtitleFile = subtitleFile
	if exit, err := validateSubtitleDelay(request); err != nil {
		return nil, exit, err
	}

	var cache domain.DeviceCache
	if !request.VLC {
		device, loadedCache, exit, err := app.resolveDevice(effectiveDevice(input))
		if err != nil {
			return nil, exit, err
		}
		cache = loadedCache
		request.Device = device.Name
	}

	if err := app.Playback.PlayMagnet(request); err != nil {
		return nil, ExitPlaybackFailed, err
	}
	if !request.VLC {
		_ = app.markDeviceUsed(cache, request.Device)
	}
	_ = app.saveSession(domain.SessionMetadata{Device: request.Device, Title: request.Title, UpdatedAt: app.now()})

	return domain.PlaybackStart{
		Status:        "playing",
		Player:        playbackPlayer(request),
		Device:        request.Device,
		Magnet:        request.Magnet,
		Title:         request.Title,
		SubtitleFile:  request.SubtitleFile,
		SubtitleDelay: request.SubtitleDelay,
	}, ExitSuccess, nil
}

func (app App) playQuery(input ParseResult) (any, ExitCode, error) {
	command := input.Command
	if app.Metadata == nil {
		return nil, ExitError, fmt.Errorf("metadata provider is not configured")
	}
	if app.Streams == nil {
		return nil, ExitError, fmt.Errorf("stream provider is not configured")
	}
	if app.Playback == nil {
		return nil, ExitError, fmt.Errorf("playback adapter is not configured")
	}

	results, err := app.Metadata.Search(command.Args[0])
	if err != nil {
		return nil, providerExit(err), err
	}
	result, ok := bestPlayableSearchResult(command.Args[0], results)
	if !ok {
		return nil, ExitNoResults, fmt.Errorf("no playable search result found")
	}

	imdbID, title, exit, err := app.resolveSearchResultIdentity(result)
	if err != nil {
		return nil, exit, err
	}

	castCommand := command
	castCommand.Args = []string{imdbID}
	streams, err := app.Streams.MovieStreams(imdbID)
	if err != nil {
		return nil, providerExit(err), err
	}
	if len(streams) == 0 {
		return nil, ExitNoResults, fmt.Errorf("no streams found")
	}
	stream, exit, err := selectStreamForCast(castCommand, streams)
	if err != nil {
		return nil, exit, err
	}

	request := domain.PlaybackRequest{
		Magnet:    stream.Magnet(imdbID),
		FileIndex: streamFileIndex(stream),
		Title:     title,
		VLC:       boolOption(command, "vlc"),
	}
	request.SubtitleDelay = stringOptionValue(command, "subtitle_delay")
	subtitleFile, exit, err := app.subtitleFileForCast(castCommand)
	if err != nil {
		return nil, exit, err
	}
	request.SubtitleFile = subtitleFile
	if exit, err := validateSubtitleDelay(request); err != nil {
		return nil, exit, err
	}

	var cache domain.DeviceCache
	if !request.VLC {
		device, loadedCache, exit, err := app.resolveDevice(effectiveDevice(input))
		if err != nil {
			return nil, exit, err
		}
		cache = loadedCache
		request.Device = device.Name
	}

	if err := app.Playback.PlayMagnet(request); err != nil {
		return nil, ExitPlaybackFailed, err
	}
	if !request.VLC {
		_ = app.markDeviceUsed(cache, request.Device)
	}
	_ = app.saveSession(domain.SessionMetadata{Device: request.Device, Title: request.Title, UpdatedAt: app.now()})

	return domain.PlaybackStart{
		Status:        "playing",
		Player:        playbackPlayer(request),
		Device:        request.Device,
		Magnet:        request.Magnet,
		Title:         request.Title,
		SubtitleFile:  request.SubtitleFile,
		SubtitleDelay: request.SubtitleDelay,
	}, ExitSuccess, nil
}

func (app App) status(input ParseResult) (any, ExitCode, error) {
	if app.Control == nil {
		return nil, ExitError, fmt.Errorf("control adapter is not configured")
	}
	device, _, _, err := app.resolveControlDevice(input)
	if err != nil {
		return nil, ExitDeviceNotFound, normalizeDeviceNotFoundError(err)
	}
	status, err := app.Control.Status(device)
	if err != nil {
		return nil, controlExit(err), normalizeControlError(err)
	}
	if status.Device == "" {
		status.Device = device
	}
	if status.Title == "" && app.SessionStore != nil {
		session, err := app.SessionStore.Load()
		if err != nil {
			return nil, ExitError, err
		}
		if session.Device == "" || session.Device == device {
			status.Title = session.Title
		}
	}
	return status, ExitSuccess, nil
}

func (app App) playbackAction(input ParseResult, action string) (any, ExitCode, error) {
	if app.Control == nil {
		return nil, ExitError, fmt.Errorf("control adapter is not configured")
	}
	device, _, _, err := app.resolveControlDevice(input)
	if err != nil {
		return nil, ExitDeviceNotFound, normalizeDeviceNotFoundError(err)
	}
	switch action {
	case "play":
		err = app.Control.Play(device)
	case "pause":
		err = app.Control.Pause(device)
	case "stop":
		err = app.Control.Stop(device)
	default:
		err = fmt.Errorf("unknown playback action %q", action)
	}
	if err != nil {
		return nil, controlExit(err), normalizeControlError(err)
	}
	return domain.ControlResult{Status: "ok", Device: device, Action: action}, ExitSuccess, nil
}

func (app App) seek(input ParseResult) (any, ExitCode, error) {
	if app.Control == nil {
		return nil, ExitError, fmt.Errorf("control adapter is not configured")
	}
	command, err := domain.ParseSeek(input.Command.Args[0])
	if err != nil {
		return nil, ExitInvalidArgs, err
	}
	device, _, _, err := app.resolveControlDevice(input)
	if err != nil {
		return nil, ExitDeviceNotFound, normalizeDeviceNotFoundError(err)
	}
	if err := app.Control.Seek(device, command); err != nil {
		return nil, controlExit(err), normalizeControlError(err)
	}
	return domain.ControlResult{Status: "ok", Device: device, Action: command.Action, Value: command.Raw}, ExitSuccess, nil
}

func (app App) volume(input ParseResult) (any, ExitCode, error) {
	if app.Control == nil {
		return nil, ExitError, fmt.Errorf("control adapter is not configured")
	}
	command, err := domain.ParseVolume(input.Command.Args[0])
	if err != nil {
		return nil, ExitInvalidArgs, err
	}
	device, _, _, err := app.resolveControlDevice(input)
	if err != nil {
		return nil, ExitDeviceNotFound, normalizeDeviceNotFoundError(err)
	}
	if err := app.Control.Volume(device, command); err != nil {
		return nil, controlExit(err), normalizeControlError(err)
	}
	return domain.ControlResult{Status: "ok", Device: device, Action: command.Action, Value: command.Raw}, ExitSuccess, nil
}

type IndexedStream struct {
	Index int `json:"index"`
	domain.StreamSource
}

func indexStreams(streams []domain.StreamSource) []IndexedStream {
	indexed := make([]IndexedStream, 0, len(streams))
	for index, stream := range streams {
		indexed = append(indexed, IndexedStream{Index: index, StreamSource: stream})
	}
	return indexed
}

func providerExit(err error) ExitCode {
	switch {
	case errors.Is(err, providers.ErrNotFound):
		return ExitNoResults
	case errors.Is(err, providers.ErrRateLimited), errors.Is(err, providers.ErrInvalidResponse):
		return ExitNetwork
	default:
		return ExitNetwork
	}
}

func controlExit(err error) ExitCode {
	if errors.Is(err, providers.ErrNotFound) {
		return ExitDeviceNotFound
	}
	return ExitPlaybackFailed
}

func normalizeControlError(err error) error {
	if errors.Is(err, providers.ErrNotFound) {
		return normalizeDeviceNotFoundError(err)
	}
	return err
}

func stringOption(command ParsedCommand, key string) (string, bool) {
	value, ok := command.Options[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

func limitOption(command ParsedCommand, fallback int) (int, ExitCode, error) {
	value, ok := stringOption(command, "limit")
	if !ok {
		return fallback, ExitSuccess, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0, ExitInvalidArgs, fmt.Errorf("limit must be a non-negative integer")
	}
	return limit, ExitSuccess, nil
}

func filterMediaType(results []domain.SearchResult, mediaType string) []domain.SearchResult {
	var target domain.MediaType
	switch mediaType {
	case "movie":
		target = domain.MediaMovie
	case "tv":
		target = domain.MediaTV
	default:
		return []domain.SearchResult{}
	}

	filtered := make([]domain.SearchResult, 0, len(results))
	for _, result := range results {
		if result.MediaType == target {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func bestPlayableSearchResult(query string, results []domain.SearchResult) (domain.SearchResult, bool) {
	normalizedQuery := normalizedSearchTitle(query)
	var firstMovie *domain.SearchResult
	for _, result := range results {
		if result.MediaType != domain.MediaMovie {
			continue
		}
		if firstMovie == nil {
			copy := result
			firstMovie = &copy
		}
		if normalizedSearchTitle(result.Title) == normalizedQuery {
			return result, true
		}
	}
	if firstMovie != nil {
		return *firstMovie, true
	}
	return domain.SearchResult{}, false
}

func normalizedSearchTitle(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func (app App) resolveSearchResultIdentity(result domain.SearchResult) (string, string, ExitCode, error) {
	title := result.Title
	if result.IMDBID != "" {
		return result.IMDBID, title, ExitSuccess, nil
	}
	switch result.MediaType {
	case domain.MediaMovie:
		detail, err := app.Metadata.MovieDetail(result.ID)
		if err != nil {
			return "", "", providerExit(err), err
		}
		if detail.Title != "" {
			title = detail.Title
		}
		if detail.IMDBID == "" {
			return "", "", ExitNoResults, fmt.Errorf("selected result has no IMDb id")
		}
		return detail.IMDBID, title, ExitSuccess, nil
	default:
		return "", "", ExitNoResults, fmt.Errorf("no playable movie result found")
	}
}

func limitSearchResults(results []domain.SearchResult, limit int) []domain.SearchResult {
	if limit > len(results) {
		limit = len(results)
	}
	return results[:limit]
}

func limitStreams(streams []domain.StreamSource, limit int) []domain.StreamSource {
	if limit > len(streams) {
		limit = len(streams)
	}
	return streams[:limit]
}

func parseUint16Option(name string, value string) (uint16, error) {
	parsed, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return uint16(parsed), nil
}

func boolOption(command ParsedCommand, key string) bool {
	value, ok := command.Options[key]
	if !ok {
		return false
	}
	enabled, ok := value.(bool)
	return ok && enabled
}

func filterSubtitles(command ParsedCommand, subtitles []domain.SubtitleResult) []domain.SubtitleResult {
	filtered := make([]domain.SubtitleResult, 0, len(subtitles))
	for _, subtitle := range subtitles {
		if boolOption(command, "trusted") && !subtitle.FromTrusted {
			continue
		}
		if boolOption(command, "hearing_impaired") && !subtitle.HearingImpaired {
			continue
		}
		filtered = append(filtered, subtitle)
	}
	return filtered
}

func (app App) fetchStreamsForCommand(command ParsedCommand) ([]domain.StreamSource, ExitCode, error) {
	seasonValue, hasSeason := stringOption(command, "season")
	episodeValue, hasEpisode := stringOption(command, "episode")

	var streams []domain.StreamSource
	var err error
	switch {
	case hasSeason && hasEpisode:
		season, parseErr := parseUint16Option("season", seasonValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		episode, parseErr := parseUint16Option("episode", episodeValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		streams, err = app.Streams.EpisodeStreams(command.Args[0], season, episode)
	case hasSeason || hasEpisode:
		return nil, ExitInvalidArgs, fmt.Errorf("cast requires both season and episode for TV episodes")
	default:
		streams, err = app.Streams.MovieStreams(command.Args[0])
	}
	if err != nil {
		return nil, providerExit(err), err
	}
	if len(streams) == 0 {
		return nil, ExitNoResults, fmt.Errorf("no streams found")
	}
	return streams, ExitSuccess, nil
}

func selectStreamForCast(command ParsedCommand, streams []domain.StreamSource) (domain.StreamSource, ExitCode, error) {
	streams = domain.RankStreams(streams)
	if preferred, ok := stringOption(command, "quality"); ok {
		quality := domain.ParseQuality(preferred)
		if quality == domain.QualityUnknown {
			return domain.StreamSource{}, ExitInvalidArgs, fmt.Errorf("unknown quality %q", preferred)
		}
		streams = domain.PreferQuality(streams, quality)
	}
	if indexValue, ok := stringOption(command, "index"); ok {
		index, err := strconv.Atoi(indexValue)
		if err != nil || index < 0 || index >= len(streams) {
			return domain.StreamSource{}, ExitInvalidArgs, fmt.Errorf("stream index out of range")
		}
		return streams[index], ExitSuccess, nil
	}
	return streams[0], ExitSuccess, nil
}

func streamSortOption(command ParsedCommand) (domain.StreamSort, ExitCode, error) {
	value, ok := stringOption(command, "sort")
	if !ok || value == "" {
		return domain.StreamSortRanked, ExitSuccess, nil
	}
	switch value {
	case string(domain.StreamSortRanked), string(domain.StreamSortSeeds), string(domain.StreamSortQuality), string(domain.StreamSortSize):
		return domain.StreamSort(value), ExitSuccess, nil
	default:
		return "", ExitInvalidArgs, fmt.Errorf("unknown stream sort %q", value)
	}
}

func (app App) subtitleFileForCast(command ParsedCommand) (string, ExitCode, error) {
	if boolOption(command, "no_subtitle") {
		return "", ExitSuccess, nil
	}
	if file := stringOptionValue(command, "subtitle_file"); file != "" {
		return file, ExitSuccess, nil
	}
	if app.Subtitles == nil {
		return "", ExitSuccess, nil
	}
	language := stringOptionValue(command, "lang")
	subtitleID := stringOptionValue(command, "subtitle_id")
	if language == "" && subtitleID == "" {
		return "", ExitSuccess, nil
	}

	subtitles, exit, err := app.searchSubtitlesForCast(command, language)
	if err != nil {
		return "", exit, err
	}
	if len(subtitles) == 0 {
		return "", ExitSuccess, nil
	}

	var selected domain.SubtitleResult
	if subtitleID != "" {
		found := false
		for _, subtitle := range subtitles {
			if subtitle.ID == subtitleID {
				selected = subtitle
				found = true
				break
			}
		}
		if !found {
			return "", ExitNoResults, fmt.Errorf("subtitle id %q not found", subtitleID)
		}
	} else {
		sort.SliceStable(subtitles, func(i, j int) bool {
			return subtitles[i].TrustScore() > subtitles[j].TrustScore()
		})
		selected = subtitles[0]
	}

	downloader, ok := app.Subtitles.(SubtitleDownloader)
	if !ok {
		return "", ExitSuccess, nil
	}
	if _, err := downloader.Download(selected); err != nil {
		return "", ExitNetwork, err
	}
	if pather, ok := app.Subtitles.(SubtitleCachePather); ok {
		return pather.CachePath(selected), ExitSuccess, nil
	}
	return subtitleCachePath(selected), ExitSuccess, nil
}

func (app App) searchSubtitlesForCast(command ParsedCommand, language string) ([]domain.SubtitleResult, ExitCode, error) {
	seasonValue, hasSeason := stringOption(command, "season")
	episodeValue, hasEpisode := stringOption(command, "episode")
	if hasSeason && hasEpisode {
		season, parseErr := parseUint16Option("season", seasonValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		episode, parseErr := parseUint16Option("episode", episodeValue)
		if parseErr != nil {
			return nil, ExitInvalidArgs, parseErr
		}
		subtitles, err := app.Subtitles.SearchEpisode(command.Args[0], season, episode, language)
		if err != nil {
			return nil, providerExit(err), err
		}
		return subtitles, ExitSuccess, nil
	}
	if hasSeason || hasEpisode {
		return nil, ExitInvalidArgs, fmt.Errorf("subtitles require both season and episode for TV episodes")
	}
	subtitles, err := app.Subtitles.Search(command.Args[0], language)
	if err != nil {
		return nil, providerExit(err), err
	}
	return subtitles, ExitSuccess, nil
}

func (app App) resolveDevice(explicit string) (domain.CastDevice, domain.DeviceCache, ExitCode, error) {
	if app.DeviceStore == nil {
		return domain.CastDevice{}, domain.DeviceCache{}, ExitError, fmt.Errorf("device store is not configured")
	}
	cache, err := app.DeviceStore.Load()
	if err != nil {
		return domain.CastDevice{}, domain.DeviceCache{}, ExitError, err
	}
	now := app.now()
	if device, ok := domain.ResolveDevice(cache, explicit, now, app.deviceCacheTTL()); ok {
		return device, cache, ExitSuccess, nil
	}
	if explicit != "" {
		return domain.CastDevice{ID: explicit, Name: explicit, Address: explicit}, cache, ExitSuccess, nil
	}
	if app.Devices == nil {
		return domain.CastDevice{}, cache, ExitDeviceNotFound, fmt.Errorf("no Chromecast devices found")
	}
	discovered, err := app.Devices.Discover()
	if err != nil {
		if errors.Is(err, providers.ErrNotFound) {
			return domain.CastDevice{}, cache, ExitDeviceNotFound, fmt.Errorf("no Chromecast devices found")
		}
		return domain.CastDevice{}, cache, ExitNetwork, err
	}
	if len(discovered) == 0 {
		return domain.CastDevice{}, cache, ExitDeviceNotFound, fmt.Errorf("no Chromecast devices found")
	}
	cache = domain.UpdateDeviceCache(cache, discovered, now)
	if err := app.DeviceStore.Save(cache); err != nil {
		return domain.CastDevice{}, cache, ExitError, err
	}
	if device, ok := domain.ResolveDevice(cache, explicit, now, app.deviceCacheTTL()); ok {
		return device, cache, ExitSuccess, nil
	}
	return domain.CastDevice{}, cache, ExitDeviceNotFound, fmt.Errorf("no Chromecast devices found")
}

func (app App) resolveControlDevice(input ParseResult) (string, domain.DeviceCache, ExitCode, error) {
	explicit := effectiveDevice(input)
	if explicit != "" {
		return explicit, domain.DeviceCache{}, ExitSuccess, nil
	}
	if app.SessionStore != nil {
		session, err := app.SessionStore.Load()
		if err != nil {
			return "", domain.DeviceCache{}, ExitError, err
		}
		if session.Device != "" {
			return session.Device, domain.DeviceCache{}, ExitSuccess, nil
		}
	}
	device, cache, exit, err := app.resolveDevice("")
	if err != nil {
		return "", cache, exit, err
	}
	return device.Name, cache, ExitSuccess, nil
}

func (app App) markDeviceUsed(cache domain.DeviceCache, deviceName string) error {
	if app.DeviceStore == nil || deviceName == "" {
		return nil
	}
	cache = domain.MarkDeviceUsed(cache, deviceName, app.now())
	return app.DeviceStore.Save(cache)
}

func (app App) saveSession(metadata domain.SessionMetadata) error {
	if app.SessionStore == nil {
		return nil
	}
	return app.SessionStore.Save(metadata)
}

func (app App) now() time.Time {
	if app.Now != nil {
		return app.Now()
	}
	return time.Now()
}

func (app App) deviceCacheTTL() time.Duration {
	if app.DeviceCacheTTL > 0 {
		return app.DeviceCacheTTL
	}
	return 10 * time.Minute
}

func stringOptionValue(command ParsedCommand, key string) string {
	value, _ := stringOption(command, key)
	return value
}

func effectiveDevice(input ParseResult) string {
	if device := stringOptionValue(input.Command, "device"); device != "" {
		return device
	}
	return input.Globals.Device
}

func fileIndexOption(command ParsedCommand) (uint32, ExitCode, error) {
	value, ok := stringOption(command, "file_idx")
	if !ok {
		return 0, ExitSuccess, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, ExitInvalidArgs, fmt.Errorf("file index must be an integer")
	}
	return uint32(parsed), ExitSuccess, nil
}

func validateSubtitleDelay(request domain.PlaybackRequest) (ExitCode, error) {
	if request.SubtitleDelay == "" {
		return ExitSuccess, nil
	}
	if !request.VLC {
		return ExitInvalidArgs, fmt.Errorf("subtitle delay is only supported with VLC playback")
	}
	if _, err := strconv.ParseFloat(request.SubtitleDelay, 64); err != nil {
		return ExitInvalidArgs, fmt.Errorf("subtitle delay must be a number of seconds")
	}
	return ExitSuccess, nil
}

func streamFileIndex(stream domain.StreamSource) uint32 {
	if stream.FileIndex == nil {
		return 0
	}
	return *stream.FileIndex
}

func playbackPlayer(request domain.PlaybackRequest) string {
	if request.VLC {
		return "VLC"
	}
	return "Chromecast"
}

func normalizeDeviceNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("no Chromecast devices found")
}

func subtitleCachePath(subtitle domain.SubtitleResult) string {
	return fmt.Sprintf("%s_%s.vtt", subtitle.Language, subtitle.ID)
}
