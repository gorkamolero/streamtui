package domain

import (
	"fmt"
	"net/url"
	"strings"
)

type PlaybackRequest struct {
	Magnet        string `json:"magnet"`
	Device        string `json:"device,omitempty"`
	VLC           bool   `json:"vlc"`
	SubtitleFile  string `json:"subtitle_file,omitempty"`
	SubtitleDelay string `json:"subtitle_delay,omitempty"`
	FileIndex     uint32 `json:"file_idx"`
	Title         string `json:"title,omitempty"`
}

type PlaybackStart struct {
	Status        string `json:"status"`
	Player        string `json:"player,omitempty"`
	Device        string `json:"device,omitempty"`
	Magnet        string `json:"magnet"`
	Title         string `json:"title,omitempty"`
	SubtitleFile  string `json:"subtitle_file,omitempty"`
	SubtitleDelay string `json:"subtitle_delay,omitempty"`
}

func ValidateMagnet(magnet string) error {
	if len(magnet) < len("magnet:?") || magnet[:len("magnet:?")] != "magnet:?" {
		return fmt.Errorf("invalid magnet link")
	}
	return nil
}

func ValidateTorrentID(value string) error {
	if strings.HasPrefix(value, "magnet:?") {
		return nil
	}
	parsed, err := url.Parse(value)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && strings.HasSuffix(strings.ToLower(parsed.Path), ".torrent") {
		return nil
	}
	if strings.HasSuffix(strings.ToLower(value), ".torrent") {
		return nil
	}
	return fmt.Errorf("invalid torrent id")
}
