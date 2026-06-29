package providers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"streamtui/internal/domain"
)

type CattControlAdapter struct{}

func NewCattControlAdapter() CattControlAdapter {
	return CattControlAdapter{}
}

func (adapter CattControlAdapter) Status(device string) (domain.PlaybackStatus, error) {
	args := cattArgs(device, "status")
	output, err := exec.Command("catt", args...).CombinedOutput()
	if err := cattControlError(output, err); err != nil {
		return domain.PlaybackStatus{}, err
	}
	return domain.ParseCattStatus(string(output), device), nil
}

func (adapter CattControlAdapter) Play(device string) error {
	return runCattControl(cattArgs(device, "play")...)
}

func (adapter CattControlAdapter) Pause(device string) error {
	return runCattControl(cattArgs(device, "pause")...)
}

func (adapter CattControlAdapter) Stop(device string) error {
	return runCattControl(cattArgs(device, "stop")...)
}

func (adapter CattControlAdapter) Seek(device string, command domain.SeekCommand) error {
	args := cattArgs(device, command.Action, strconv.FormatUint(command.Value, 10))
	return runCattControl(args...)
}

func (adapter CattControlAdapter) Volume(device string, command domain.VolumeCommand) error {
	args := cattArgs(device, command.Action)
	if command.Action == "volume" {
		args = cattArgs(device, command.Action, strconv.FormatUint(uint64(command.Value), 10))
	}
	return runCattControl(args...)
}

func BuildCattArgs(device string, args ...string) []string {
	return cattArgs(device, args...)
}

func cattArgs(device string, args ...string) []string {
	if device == "" {
		return args
	}
	return append([]string{"-d", device}, args...)
}

func runCattControl(args ...string) error {
	output, err := exec.Command("catt", args...).CombinedOutput()
	return cattControlError(output, err)
}

func cattControlError(output []byte, err error) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	lower := strings.ToLower(message)
	if strings.Contains(lower, "no devices found") || (strings.Contains(lower, "specified device") && strings.Contains(lower, "not found")) {
		if message == "" {
			return ErrNotFound
		}
		return fmt.Errorf("%w: %s", ErrNotFound, message)
	}
	if message == "" {
		return err
	}
	return fmt.Errorf("%s: %w", message, err)
}

type FileSessionStore struct {
	Path string
}

func NewFileSessionStore() FileSessionStore {
	return FileSessionStore{Path: defaultSessionPath()}
}

func (store FileSessionStore) Load() (domain.SessionMetadata, error) {
	content, err := os.ReadFile(store.Path)
	if os.IsNotExist(err) {
		return domain.SessionMetadata{}, nil
	}
	if err != nil {
		return domain.SessionMetadata{}, err
	}
	return domain.UnmarshalSession(content)
}

func (store FileSessionStore) Save(metadata domain.SessionMetadata) error {
	content, err := domain.MarshalSession(metadata)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(store.Path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(store.Path, content, 0o644)
}

func defaultSessionPath() string {
	if cacheDir, ok := os.LookupEnv("XDG_CACHE_HOME"); ok && cacheDir != "" {
		return filepath.Join(cacheDir, "streamtui", "session.json")
	}
	if home, ok := os.LookupEnv("HOME"); ok && home != "" {
		return filepath.Join(home, ".cache", "streamtui", "session.json")
	}
	return filepath.Join(os.TempDir(), "streamtui", "session.json")
}
