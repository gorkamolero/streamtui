package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type PlaybackState string

const (
	PlaybackIdle      PlaybackState = "idle"
	PlaybackBuffering PlaybackState = "buffering"
	PlaybackPlaying   PlaybackState = "playing"
	PlaybackPaused    PlaybackState = "paused"
	PlaybackStopped   PlaybackState = "stopped"
	PlaybackError     PlaybackState = "error"
)

type PlaybackStatus struct {
	State    PlaybackState `json:"state"`
	Title    string        `json:"title,omitempty"`
	Device   string        `json:"device,omitempty"`
	Position *uint64       `json:"position,omitempty"`
	Duration *uint64       `json:"duration,omitempty"`
	Progress *float64      `json:"progress,omitempty"`
	Volume   *uint8        `json:"volume,omitempty"`
}

type ControlResult struct {
	Status string `json:"status"`
	Device string `json:"device,omitempty"`
	Action string `json:"action,omitempty"`
	Value  string `json:"value,omitempty"`
}

type SeekCommand struct {
	Action string
	Value  uint64
	Raw    string
}

type VolumeCommand struct {
	Action string
	Value  uint8
	Raw    string
}

type SessionMetadata struct {
	Device    string    `json:"device,omitempty"`
	Title     string    `json:"title,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ParseCattStatus(output string, device string) PlaybackStatus {
	status := PlaybackStatus{State: PlaybackIdle, Device: device}
	var position uint64
	var duration uint64
	var hasPosition bool
	var hasDuration bool

	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch key {
		case "state":
			status.State = parsePlaybackState(value)
		case "title":
			status.Title = value
		case "duration":
			if parsed, ok := parseSeconds(value); ok {
				duration = parsed
				hasDuration = true
				status.Duration = &duration
			}
		case "current time":
			if parsed, ok := parseSeconds(value); ok {
				position = parsed
				hasPosition = true
				status.Position = &position
			}
		case "volume":
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				volume := uint8(parsed)
				status.Volume = &volume
			}
		}
	}

	if hasPosition && hasDuration && duration > 0 {
		progress := float64(position) / float64(duration)
		status.Progress = &progress
	}

	return status
}

func ParseSeek(value string) (SeekCommand, error) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "+") {
		seconds, err := strconv.ParseUint(strings.TrimPrefix(trimmed, "+"), 10, 64)
		if err != nil {
			return SeekCommand{}, fmt.Errorf("invalid seek position")
		}
		return SeekCommand{Action: "ffwd", Value: seconds, Raw: trimmed}, nil
	}
	if strings.HasPrefix(trimmed, "-") {
		seconds, err := strconv.ParseUint(strings.TrimPrefix(trimmed, "-"), 10, 64)
		if err != nil {
			return SeekCommand{}, fmt.Errorf("invalid seek position")
		}
		return SeekCommand{Action: "rewind", Value: seconds, Raw: trimmed}, nil
	}
	if seconds, ok := parseTimestamp(trimmed); ok {
		return SeekCommand{Action: "seek", Value: seconds, Raw: trimmed}, nil
	}
	return SeekCommand{}, fmt.Errorf("invalid seek position")
}

func ParseVolume(value string) (VolumeCommand, error) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "+") {
		_, err := strconv.ParseUint(strings.TrimPrefix(trimmed, "+"), 10, 8)
		if err != nil {
			return VolumeCommand{}, fmt.Errorf("invalid volume level")
		}
		return VolumeCommand{Action: "volumeup", Raw: trimmed}, nil
	}
	if strings.HasPrefix(trimmed, "-") {
		_, err := strconv.ParseUint(strings.TrimPrefix(trimmed, "-"), 10, 8)
		if err != nil {
			return VolumeCommand{}, fmt.Errorf("invalid volume level")
		}
		return VolumeCommand{Action: "volumedown", Raw: trimmed}, nil
	}
	volume, err := strconv.ParseUint(trimmed, 10, 8)
	if err != nil {
		return VolumeCommand{}, fmt.Errorf("invalid volume level")
	}
	if volume > 100 {
		volume = 100
	}
	return VolumeCommand{Action: "volume", Value: uint8(volume), Raw: trimmed}, nil
}

func MarshalSession(metadata SessionMetadata) ([]byte, error) {
	return json.MarshalIndent(metadata, "", "  ")
}

func UnmarshalSession(content []byte) (SessionMetadata, error) {
	var metadata SessionMetadata
	err := json.Unmarshal(content, &metadata)
	return metadata, err
}

func parsePlaybackState(value string) PlaybackState {
	switch strings.ToUpper(value) {
	case "PLAYING":
		return PlaybackPlaying
	case "PAUSED":
		return PlaybackPaused
	case "BUFFERING":
		return PlaybackBuffering
	case "STOPPED":
		return PlaybackStopped
	case "IDLE":
		return PlaybackIdle
	default:
		return PlaybackIdle
	}
}

func parseSeconds(value string) (uint64, bool) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return uint64(parsed), true
}

func parseTimestamp(value string) (uint64, bool) {
	if seconds, err := strconv.ParseUint(value, 10, 64); err == nil {
		return seconds, true
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, false
	}
	total := uint64(0)
	for _, part := range parts {
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return 0, false
		}
		total = total*60 + parsed
	}
	return total, true
}
