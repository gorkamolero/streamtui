package providers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"streamtui/internal/domain"
)

type CattDeviceProvider struct{}

func NewCattDeviceProvider() CattDeviceProvider {
	return CattDeviceProvider{}
}

func (provider CattDeviceProvider) Discover() ([]domain.CastDevice, error) {
	output, err := exec.Command("catt", "scan").CombinedOutput()
	return parseCattScanOutput(string(output), err)
}

func parseCattScanOutput(output string, err error) ([]domain.CastDevice, error) {
	if err != nil {
		if strings.Contains(strings.ToLower(output), "no devices found") {
			return nil, ErrNotFound
		}
		return nil, err
	}
	devices := domain.ParseCattScan(output)
	if len(devices) == 0 {
		return nil, ErrNotFound
	}
	return devices, nil
}

type FileDeviceStore struct {
	Path string
}

func NewFileDeviceStore() FileDeviceStore {
	return FileDeviceStore{Path: defaultDeviceCachePath()}
}

func (store FileDeviceStore) Load() (domain.DeviceCache, error) {
	content, err := os.ReadFile(store.Path)
	if os.IsNotExist(err) {
		return domain.DeviceCache{}, nil
	}
	if err != nil {
		return domain.DeviceCache{}, err
	}
	return domain.UnmarshalDeviceCache(content)
}

func (store FileDeviceStore) Save(cache domain.DeviceCache) error {
	content, err := domain.MarshalDeviceCache(cache)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(store.Path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(store.Path, content, 0o644)
}

func defaultDeviceCachePath() string {
	if cacheDir, ok := os.LookupEnv("XDG_CACHE_HOME"); ok && cacheDir != "" {
		return filepath.Join(cacheDir, "streamtui", "devices.json")
	}
	if home, ok := os.LookupEnv("HOME"); ok && home != "" {
		return filepath.Join(home, ".cache", "streamtui", "devices.json")
	}
	return filepath.Join(os.TempDir(), "streamtui", "devices.json")
}
