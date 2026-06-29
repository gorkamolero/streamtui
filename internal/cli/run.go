package cli

import (
	"io"
	"os"

	"streamtui/internal/config"
	"streamtui/internal/providers"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) ExitCode {
	return RunWithApp(args, DefaultApp(), stdout, stderr)
}

func DefaultApp() App {
	metadata := providers.NewTMDBClient(config.TMDBAPIKey())
	if baseURL := os.Getenv("STREAMTUI_TMDB_BASE_URL"); baseURL != "" {
		metadata = providers.NewTMDBClientWithBaseURL(config.TMDBAPIKey(), baseURL)
	}

	streams := providers.NewTorrentioClient()
	if baseURL := os.Getenv("STREAMTUI_TORRENTIO_BASE_URL"); baseURL != "" {
		streams = providers.NewTorrentioClientWithBaseURL(baseURL)
	}

	subtitles := providers.NewSubtitleClient()
	if baseURL := os.Getenv("STREAMTUI_SUBTITLES_BASE_URL"); baseURL != "" {
		subtitles = providers.NewSubtitleClientWithBaseURL(baseURL, "")
	}

	return App{
		Metadata:     metadata,
		Streams:      streams,
		Subtitles:    subtitles,
		Devices:      providers.NewCattDeviceProvider(),
		DeviceStore:  providers.NewFileDeviceStore(),
		Playback:     providers.NewWebtorrentPlaybackAdapter(),
		Control:      providers.NewCattControlAdapter(),
		SessionStore: providers.NewFileSessionStore(),
	}
}

func RunWithApp(args []string, app App, stdout io.Writer, stderr io.Writer) ExitCode {
	result, exit, err := Parse(args)
	if err != nil {
		if writeErr := writeJSON(stdout, errorEnvelope(codeForExit(exit), err.Error())); writeErr != nil {
			_, _ = io.WriteString(stderr, writeErr.Error()+"\n")
			return ExitError
		}
		return exit
	}

	data, exit, err := app.Execute(result)
	if err != nil {
		if writeErr := writeJSON(stdout, errorEnvelope(codeForExit(exit), err.Error())); writeErr != nil {
			_, _ = io.WriteString(stderr, writeErr.Error()+"\n")
			return ExitError
		}
		return exit
	}

	if writeErr := writeJSON(stdout, successEnvelope(data)); writeErr != nil {
		_, _ = io.WriteString(stderr, writeErr.Error()+"\n")
		return ExitError
	}
	return ExitSuccess
}
