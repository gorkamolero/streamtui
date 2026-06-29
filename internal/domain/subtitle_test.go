package domain

import (
	"strings"
	"testing"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := map[string]string{
		"en":   "eng",
		"eng":  "eng",
		"es":   "spa",
		"spa":  "spa",
		" fr ": "fre",
	}

	for input, want := range tests {
		if got := NormalizeLanguage(input); got != want {
			t.Fatalf("NormalizeLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeLanguages(t *testing.T) {
	got := NormalizeLanguages("en, es,fre")
	want := []string{"eng", "spa", "fre"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLanguageName(t *testing.T) {
	if got := LanguageName("spa"); got != "Spanish" {
		t.Fatalf("LanguageName = %q, want Spanish", got)
	}
	if got := LanguageName("unk"); got != "UNK" {
		t.Fatalf("LanguageName = %q, want UNK", got)
	}
}

func TestSRTToWebVTT(t *testing.T) {
	srt := "1\n00:01:23,456 --> 00:01:25,789\nLine"
	got := SRTToWebVTT(srt)
	if !strings.Contains(got, "WEBVTT\n\n") {
		t.Fatalf("missing WEBVTT header: %q", got)
	}
	if !strings.Contains(got, "00:01:23.456 --> 00:01:25.789") {
		t.Fatalf("timestamp not converted: %q", got)
	}
}

func TestSubtitleTrustScore(t *testing.T) {
	trusted := SubtitleResult{FromTrusted: true, Downloads: 1200}
	ai := SubtitleResult{FromTrusted: true, AITranslated: true, Downloads: 1200}
	if trusted.TrustScore() <= ai.TrustScore() {
		t.Fatalf("trusted score should beat ai score")
	}
}
