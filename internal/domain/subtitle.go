package domain

import (
	"strings"
)

type SubFormat string

const (
	SubFormatSRT    SubFormat = "srt"
	SubFormatWebVTT SubFormat = "webvtt"
	SubFormatSub    SubFormat = "sub"
	SubFormatASS    SubFormat = "ass"
)

type SubtitleResult struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	Language        string    `json:"language"`
	LanguageName    string    `json:"language_name"`
	Release         string    `json:"release"`
	FPS             *float32  `json:"fps"`
	Format          SubFormat `json:"format"`
	Downloads       uint32    `json:"downloads"`
	FromTrusted     bool      `json:"from_trusted"`
	HearingImpaired bool      `json:"hearing_impaired"`
	AITranslated    bool      `json:"ai_translated"`
}

func (subtitle SubtitleResult) TrustScore() int {
	score := 0
	if subtitle.FromTrusted {
		score += 100
	}
	if subtitle.HearingImpaired {
		score += 10
	}
	if subtitle.AITranslated {
		score -= 50
	}
	switch {
	case subtitle.Downloads > 1000:
		score += 30
	case subtitle.Downloads > 100:
		score += 20
	case subtitle.Downloads > 10:
		score += 10
	}
	return score
}

func NormalizeLanguage(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "en", "eng":
		return "eng"
	case "es", "spa":
		return "spa"
	case "fr", "fre", "fra":
		return "fre"
	case "de", "ger", "deu":
		return "ger"
	case "it", "ita":
		return "ita"
	case "pt", "por", "pob":
		return "por"
	default:
		return strings.ToLower(strings.TrimSpace(input))
	}
}

func NormalizeLanguages(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	languages := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := NormalizeLanguage(part)
		if normalized != "" {
			languages = append(languages, normalized)
		}
	}
	return languages
}

func LanguageName(code string) string {
	switch NormalizeLanguage(code) {
	case "eng":
		return "English"
	case "spa":
		return "Spanish"
	case "fre":
		return "French"
	case "ger":
		return "German"
	case "ita":
		return "Italian"
	case "por", "pob":
		return "Portuguese"
	case "rus":
		return "Russian"
	case "jpn":
		return "Japanese"
	case "kor":
		return "Korean"
	case "chi", "zho":
		return "Chinese"
	case "ara":
		return "Arabic"
	case "hin":
		return "Hindi"
	case "dut", "nld":
		return "Dutch"
	case "pol":
		return "Polish"
	case "tur":
		return "Turkish"
	case "swe":
		return "Swedish"
	case "nor":
		return "Norwegian"
	case "dan":
		return "Danish"
	case "fin":
		return "Finnish"
	case "gre", "ell":
		return "Greek"
	case "heb":
		return "Hebrew"
	case "hun":
		return "Hungarian"
	case "cze", "ces":
		return "Czech"
	case "rum", "ron":
		return "Romanian"
	case "bul":
		return "Bulgarian"
	case "hrv":
		return "Croatian"
	case "slv":
		return "Slovenian"
	case "srp":
		return "Serbian"
	case "ukr":
		return "Ukrainian"
	case "vie":
		return "Vietnamese"
	case "tha":
		return "Thai"
	case "ind":
		return "Indonesian"
	case "may", "msa":
		return "Malay"
	case "ice", "isl":
		return "Icelandic"
	default:
		return strings.ToUpper(code)
	}
}

func SRTToWebVTT(srt string) string {
	var builder strings.Builder
	builder.WriteString("WEBVTT\n\n")
	for _, line := range strings.Split(srt, "\n") {
		if strings.Contains(line, " --> ") {
			line = strings.ReplaceAll(line, ",", ".")
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	return builder.String()
}
