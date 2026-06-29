package domain

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Quality string

const (
	Quality4K      Quality = "4K"
	Quality1080p   Quality = "1080p"
	Quality720p    Quality = "720p"
	Quality480p    Quality = "480p"
	QualityUnknown Quality = "unknown"
)

type StreamSort string

const (
	StreamSortRanked  StreamSort = "ranked"
	StreamSortSeeds   StreamSort = "seeds"
	StreamSortQuality StreamSort = "quality"
	StreamSortSize    StreamSort = "size"
)

type StreamSource struct {
	Name      string  `json:"name"`
	Title     string  `json:"title"`
	InfoHash  string  `json:"info_hash"`
	FileIndex *uint32 `json:"file_idx"`
	Seeds     uint32  `json:"seeds"`
	Quality   Quality `json:"quality"`
	SizeBytes *uint64 `json:"size_bytes"`
}

func ParseQuality(value string) Quality {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "4k"), strings.Contains(lower, "2160p"), strings.Contains(lower, "uhd"):
		return Quality4K
	case strings.Contains(lower, "1080p"), strings.Contains(lower, "fhd"):
		return Quality1080p
	case strings.Contains(lower, "720p"), strings.Contains(lower, "hd") && !strings.Contains(lower, "hdcam"):
		return Quality720p
	case strings.Contains(lower, "480p"), strings.Contains(lower, "sd"):
		return Quality480p
	default:
		return QualityUnknown
	}
}

func QualityRank(quality Quality) int {
	switch quality {
	case Quality4K:
		return 4
	case Quality1080p:
		return 3
	case Quality720p:
		return 2
	case Quality480p:
		return 1
	default:
		return 0
	}
}

func ParseSeeds(title string) uint32 {
	emojiPattern := regexp.MustCompile(`👤\s*(\d+(?:\.\d+)?)\s*(k)?`)
	if matches := emojiPattern.FindStringSubmatch(title); len(matches) == 3 {
		value, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0
		}
		if matches[2] == "k" {
			value *= 1000
		}
		return uint32(value)
	}

	seedPattern := regexp.MustCompile(`(?i)seeds?:\s*(\d+)`)
	if matches := seedPattern.FindStringSubmatch(title); len(matches) == 2 {
		value, err := strconv.ParseUint(matches[1], 10, 32)
		if err != nil {
			return 0
		}
		return uint32(value)
	}

	return 0
}

func ParseSize(title string) *uint64 {
	sizePattern := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(GB|MB)`)
	matches := sizePattern.FindStringSubmatch(title)
	if len(matches) != 3 {
		return nil
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return nil
	}

	switch strings.ToUpper(matches[2]) {
	case "GB":
		bytes := uint64(value * 1024 * 1024 * 1024)
		return &bytes
	case "MB":
		bytes := uint64(value * 1024 * 1024)
		return &bytes
	default:
		return nil
	}
}

func (source StreamSource) Magnet(displayName string) string {
	return fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", source.InfoHash, url.PathEscape(displayName))
}

func RankStreams(streams []StreamSource) []StreamSource {
	ranked := append([]StreamSource(nil), streams...)
	sort.SliceStable(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		if QualityRank(left.Quality) != QualityRank(right.Quality) {
			return QualityRank(left.Quality) > QualityRank(right.Quality)
		}
		if left.Seeds != right.Seeds {
			return left.Seeds > right.Seeds
		}
		leftSize := uint64(0)
		rightSize := uint64(0)
		if left.SizeBytes != nil {
			leftSize = *left.SizeBytes
		}
		if right.SizeBytes != nil {
			rightSize = *right.SizeBytes
		}
		if leftSize != rightSize {
			return leftSize > rightSize
		}
		return left.Name < right.Name
	})
	return ranked
}

func SortStreams(streams []StreamSource, criterion StreamSort) []StreamSource {
	switch criterion {
	case StreamSortSeeds:
		return sortStreamsBy(streams, func(left StreamSource, right StreamSource) bool {
			if left.Seeds != right.Seeds {
				return left.Seeds > right.Seeds
			}
			return rankedTieBreak(left, right)
		})
	case StreamSortQuality:
		return sortStreamsBy(streams, func(left StreamSource, right StreamSource) bool {
			if QualityRank(left.Quality) != QualityRank(right.Quality) {
				return QualityRank(left.Quality) > QualityRank(right.Quality)
			}
			return rankedTieBreak(left, right)
		})
	case StreamSortSize:
		return sortStreamsBy(streams, func(left StreamSource, right StreamSource) bool {
			leftSize := streamSize(left)
			rightSize := streamSize(right)
			if leftSize != rightSize {
				return leftSize > rightSize
			}
			return rankedTieBreak(left, right)
		})
	default:
		return RankStreams(streams)
	}
}

func PreferQuality(streams []StreamSource, target Quality) []StreamSource {
	ranked := RankStreams(streams)
	sort.SliceStable(ranked, func(i, j int) bool {
		left := qualityDistance(ranked[i].Quality, target)
		right := qualityDistance(ranked[j].Quality, target)
		if left != right {
			return left < right
		}
		return false
	})
	return ranked
}

func sortStreamsBy(streams []StreamSource, less func(StreamSource, StreamSource) bool) []StreamSource {
	sorted := append([]StreamSource(nil), streams...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return less(sorted[i], sorted[j])
	})
	return sorted
}

func rankedTieBreak(left StreamSource, right StreamSource) bool {
	if QualityRank(left.Quality) != QualityRank(right.Quality) {
		return QualityRank(left.Quality) > QualityRank(right.Quality)
	}
	if left.Seeds != right.Seeds {
		return left.Seeds > right.Seeds
	}
	if streamSize(left) != streamSize(right) {
		return streamSize(left) > streamSize(right)
	}
	return left.Name < right.Name
}

func streamSize(stream StreamSource) uint64 {
	if stream.SizeBytes == nil {
		return 0
	}
	return *stream.SizeBytes
}

func qualityDistance(quality Quality, target Quality) int {
	distance := QualityRank(quality) - QualityRank(target)
	if distance < 0 {
		return -distance
	}
	return distance
}

func FilterMinQuality(streams []StreamSource, minimum Quality) []StreamSource {
	filtered := make([]StreamSource, 0, len(streams))
	minRank := QualityRank(minimum)
	for _, stream := range streams {
		if QualityRank(stream.Quality) >= minRank {
			filtered = append(filtered, stream)
		}
	}
	return filtered
}
