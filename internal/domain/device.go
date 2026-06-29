package domain

import (
	"encoding/json"
	"net"
	"strings"
	"time"
)

type CastDevice struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
	Port    uint16  `json:"port"`
	Model   *string `json:"model"`
}

type CachedDevice struct {
	CastDevice
	LastSeen time.Time  `json:"last_seen"`
	LastUsed *time.Time `json:"last_used"`
}

type DeviceCache struct {
	DefaultDevice string         `json:"default_device,omitempty"`
	Devices       []CachedDevice `json:"devices"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func ParseCattScan(output string) []CastDevice {
	devices := []CastDevice{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Scanning") || strings.Contains(line, "No devices") {
			continue
		}

		parts := strings.SplitN(line, " - ", 3)
		if len(parts) < 2 {
			continue
		}
		address := strings.TrimSpace(parts[0])
		if net.ParseIP(address) == nil {
			continue
		}

		var model *string
		if len(parts) == 3 {
			trimmed := strings.TrimSpace(parts[2])
			model = &trimmed
		}

		devices = append(devices, CastDevice{
			ID:      address,
			Name:    strings.TrimSpace(parts[1]),
			Address: address,
			Port:    8009,
			Model:   model,
		})
	}
	return devices
}

func (cache DeviceCache) Fresh(now time.Time, ttl time.Duration) bool {
	if cache.UpdatedAt.IsZero() {
		return false
	}
	return now.Sub(cache.UpdatedAt) <= ttl
}

func (cache DeviceCache) DeviceList() []CastDevice {
	devices := make([]CastDevice, 0, len(cache.Devices))
	for _, device := range cache.Devices {
		devices = append(devices, device.CastDevice)
	}
	return devices
}

func UpdateDeviceCache(cache DeviceCache, devices []CastDevice, now time.Time) DeviceCache {
	previousLastUsed := map[string]*time.Time{}
	for _, device := range cache.Devices {
		if device.LastUsed != nil {
			value := *device.LastUsed
			previousLastUsed[device.Name] = &value
		}
	}

	cached := make([]CachedDevice, 0, len(devices))
	for _, device := range devices {
		cached = append(cached, CachedDevice{
			CastDevice: device,
			LastSeen:   now,
			LastUsed:   previousLastUsed[device.Name],
		})
	}
	cache.Devices = cached
	cache.UpdatedAt = now
	return cache
}

func MarkDeviceUsed(cache DeviceCache, deviceName string, now time.Time) DeviceCache {
	for index := range cache.Devices {
		if cache.Devices[index].Name == deviceName {
			value := now
			cache.Devices[index].LastUsed = &value
		}
	}
	return cache
}

func ResolveDevice(cache DeviceCache, explicit string, now time.Time, ttl time.Duration) (CastDevice, bool) {
	if explicit != "" {
		return findDevice(cache.Devices, explicit)
	}
	if cache.DefaultDevice != "" {
		if device, ok := findDevice(cache.Devices, cache.DefaultDevice); ok {
			return device, true
		}
	}
	if device, ok := lastUsedDevice(cache.Devices); ok {
		return device, true
	}
	if cache.Fresh(now, ttl) && len(cache.Devices) > 0 {
		return cache.Devices[0].CastDevice, true
	}
	return CastDevice{}, false
}

func MarshalDeviceCache(cache DeviceCache) ([]byte, error) {
	return json.MarshalIndent(cache, "", "  ")
}

func UnmarshalDeviceCache(content []byte) (DeviceCache, error) {
	var cache DeviceCache
	err := json.Unmarshal(content, &cache)
	return cache, err
}

func findDevice(devices []CachedDevice, name string) (CastDevice, bool) {
	for _, device := range devices {
		if device.Name == name || device.ID == name || device.Address == name {
			return device.CastDevice, true
		}
	}
	return CastDevice{}, false
}

func lastUsedDevice(devices []CachedDevice) (CastDevice, bool) {
	var selected CachedDevice
	found := false
	for _, device := range devices {
		if device.LastUsed == nil {
			continue
		}
		if !found || device.LastUsed.After(*selected.LastUsed) {
			selected = device
			found = true
		}
	}
	return selected.CastDevice, found
}
