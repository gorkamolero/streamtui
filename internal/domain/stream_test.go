package domain

import "testing"

func TestParseQuality(t *testing.T) {
	tests := map[string]Quality{
		"Movie 2160p UHD":    Quality4K,
		"Movie 1080p BluRay": Quality1080p,
		"Movie 720p WEB":     Quality720p,
		"Movie HDCAM":        QualityUnknown,
		"Movie 480p":         Quality480p,
		"Mystery":            QualityUnknown,
	}

	for input, want := range tests {
		if got := ParseQuality(input); got != want {
			t.Fatalf("ParseQuality(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseSeeds(t *testing.T) {
	tests := map[string]uint32{
		"👤 142":     142,
		"👤 1.2k":    1200,
		"seeds: 88": 88,
		"seed: 12":  12,
		"none":      0,
	}

	for input, want := range tests {
		if got := ParseSeeds(input); got != want {
			t.Fatalf("ParseSeeds(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseSize(t *testing.T) {
	got := ParseSize("Release 4.2 GB")
	if got == nil {
		t.Fatalf("ParseSize returned nil")
	}
	sizeGB := 4.2
	want := uint64(sizeGB * float64(1024*1024*1024))
	if *got != want {
		t.Fatalf("ParseSize = %d, want %d", *got, want)
	}

	got = ParseSize("Release 890 MB")
	if got == nil {
		t.Fatalf("ParseSize returned nil")
	}
	want = uint64(890 * 1024 * 1024)
	if *got != want {
		t.Fatalf("ParseSize = %d, want %d", *got, want)
	}

	if got := ParseSize("Release unknown"); got != nil {
		t.Fatalf("ParseSize = %v, want nil", *got)
	}
}

func TestRankStreamsDeterministic(t *testing.T) {
	sizeSmall := uint64(1)
	sizeLarge := uint64(2)
	streams := []StreamSource{
		{Name: "1080-low", Quality: Quality1080p, Seeds: 10, SizeBytes: &sizeSmall},
		{Name: "4k-low", Quality: Quality4K, Seeds: 1, SizeBytes: &sizeSmall},
		{Name: "1080-high-small", Quality: Quality1080p, Seeds: 50, SizeBytes: &sizeSmall},
		{Name: "1080-high-large", Quality: Quality1080p, Seeds: 50, SizeBytes: &sizeLarge},
		{Name: "720-high", Quality: Quality720p, Seeds: 500, SizeBytes: &sizeLarge},
	}

	ranked := RankStreams(streams)
	wantOrder := []string{"4k-low", "1080-high-large", "1080-high-small", "1080-low", "720-high"}
	for i, want := range wantOrder {
		if ranked[i].Name != want {
			t.Fatalf("ranked[%d] = %q, want %q", i, ranked[i].Name, want)
		}
	}
}

func TestSortStreamsBySeedsQualityAndSize(t *testing.T) {
	sizeSmall := uint64(1)
	sizeLarge := uint64(2)
	streams := []StreamSource{
		{Name: "720-high", Quality: Quality720p, Seeds: 500, SizeBytes: &sizeSmall},
		{Name: "1080-low-large", Quality: Quality1080p, Seeds: 10, SizeBytes: &sizeLarge},
		{Name: "1080-mid-small", Quality: Quality1080p, Seeds: 50, SizeBytes: &sizeSmall},
	}

	bySeeds := SortStreams(streams, StreamSortSeeds)
	if bySeeds[0].Name != "720-high" {
		t.Fatalf("bySeeds[0] = %q, want 720-high", bySeeds[0].Name)
	}

	byQuality := SortStreams(streams, StreamSortQuality)
	if byQuality[0].Name != "1080-mid-small" {
		t.Fatalf("byQuality[0] = %q, want 1080-mid-small", byQuality[0].Name)
	}

	bySize := SortStreams(streams, StreamSortSize)
	if bySize[0].Name != "1080-low-large" {
		t.Fatalf("bySize[0] = %q, want 1080-low-large", bySize[0].Name)
	}
}

func TestPreferQualityKeepsRankedTieBreaks(t *testing.T) {
	streams := []StreamSource{
		{Name: "4k", Quality: Quality4K, Seeds: 500},
		{Name: "1080-low", Quality: Quality1080p, Seeds: 10},
		{Name: "1080-high", Quality: Quality1080p, Seeds: 50},
	}

	preferred := PreferQuality(streams, Quality1080p)
	wantOrder := []string{"1080-high", "1080-low", "4k"}
	for i, want := range wantOrder {
		if preferred[i].Name != want {
			t.Fatalf("preferred[%d] = %q, want %q", i, preferred[i].Name, want)
		}
	}
}

func TestFilterMinQuality(t *testing.T) {
	streams := []StreamSource{
		{Name: "4k", Quality: Quality4K},
		{Name: "1080", Quality: Quality1080p},
		{Name: "720", Quality: Quality720p},
	}

	filtered := FilterMinQuality(streams, Quality1080p)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
	if filtered[0].Name != "4k" || filtered[1].Name != "1080" {
		t.Fatalf("unexpected filtered order: %#v", filtered)
	}
}

func TestMagnetEscapesDisplayName(t *testing.T) {
	source := StreamSource{InfoHash: "abcdef"}
	got := source.Magnet("The Batman")
	want := "magnet:?xt=urn:btih:abcdef&dn=The%20Batman"
	if got != want {
		t.Fatalf("Magnet = %q, want %q", got, want)
	}
}
