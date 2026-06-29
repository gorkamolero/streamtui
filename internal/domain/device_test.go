package domain

import (
	"testing"
	"time"
)

func TestParseCattScan(t *testing.T) {
	devices := ParseCattScan("Scanning for Chromecast devices...\n192.168.1.36 - Living Room TV - Google Chromecast\n192.168.1.37 - Bedroom TV\n")
	if len(devices) != 2 {
		t.Fatalf("len(devices) = %d, want 2", len(devices))
	}
	if devices[0].Name != "Living Room TV" || devices[0].Address != "192.168.1.36" {
		t.Fatalf("first device = %#v", devices[0])
	}
	if devices[0].Model == nil || *devices[0].Model != "Google Chromecast" {
		t.Fatalf("model = %v, want Google Chromecast", devices[0].Model)
	}
}

func TestDeviceCacheFresh(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	cache := DeviceCache{UpdatedAt: now.Add(-5 * time.Minute)}
	if !cache.Fresh(now, 10*time.Minute) {
		t.Fatalf("cache should be fresh")
	}
	if cache.Fresh(now, time.Minute) {
		t.Fatalf("cache should be stale")
	}
}

func TestResolveDeviceOrder(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	lastUsed := now.Add(-time.Minute)
	cache := DeviceCache{
		DefaultDevice: "Default TV",
		UpdatedAt:     now,
		Devices: []CachedDevice{
			{CastDevice: CastDevice{Name: "Cached TV", Address: "1.1.1.1"}},
			{CastDevice: CastDevice{Name: "Default TV", Address: "1.1.1.2"}},
			{CastDevice: CastDevice{Name: "Last TV", Address: "1.1.1.3"}, LastUsed: &lastUsed},
		},
	}

	device, ok := ResolveDevice(cache, "Cached TV", now, time.Hour)
	if !ok || device.Name != "Cached TV" {
		t.Fatalf("explicit resolution = %#v/%v", device, ok)
	}

	device, ok = ResolveDevice(cache, "", now, time.Hour)
	if !ok || device.Name != "Default TV" {
		t.Fatalf("default resolution = %#v/%v", device, ok)
	}

	cache.DefaultDevice = ""
	device, ok = ResolveDevice(cache, "", now, time.Hour)
	if !ok || device.Name != "Last TV" {
		t.Fatalf("last-used resolution = %#v/%v", device, ok)
	}

	cache.Devices[2].LastUsed = nil
	device, ok = ResolveDevice(cache, "", now, time.Hour)
	if !ok || device.Name != "Cached TV" {
		t.Fatalf("cached resolution = %#v/%v", device, ok)
	}
}

func TestUpdateDeviceCachePreservesLastUsed(t *testing.T) {
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	usedAt := now.Add(-time.Hour)
	cache := DeviceCache{
		Devices: []CachedDevice{{CastDevice: CastDevice{Name: "TV"}, LastUsed: &usedAt}},
	}

	updated := UpdateDeviceCache(cache, []CastDevice{{Name: "TV", Address: "1.1.1.1"}}, now)
	if len(updated.Devices) != 1 {
		t.Fatalf("len = %d, want 1", len(updated.Devices))
	}
	if updated.Devices[0].LastUsed == nil || !updated.Devices[0].LastUsed.Equal(usedAt) {
		t.Fatalf("last_used was not preserved: %#v", updated.Devices[0].LastUsed)
	}
}
