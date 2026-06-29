package domain

type MediaType string

const (
	MediaMovie MediaType = "movie"
	MediaTV    MediaType = "tv"
)

type SearchResult struct {
	ID          uint64    `json:"id"`
	MediaType   MediaType `json:"media_type"`
	Title       string    `json:"title"`
	Year        *uint16   `json:"year"`
	Overview    string    `json:"overview"`
	PosterPath  *string   `json:"poster_path"`
	VoteAverage float32   `json:"vote_average"`
	IMDBID      string    `json:"imdb_id,omitempty"`
}

type SeasonSummary struct {
	SeasonNumber uint8   `json:"season_number"`
	EpisodeCount uint16  `json:"episode_count"`
	Name         *string `json:"name"`
	AirDate      *string `json:"air_date"`
}

type MovieDetail struct {
	ID           uint64   `json:"id"`
	IMDBID       string   `json:"imdb_id"`
	Title        string   `json:"title"`
	Year         uint16   `json:"year"`
	Runtime      uint32   `json:"runtime"`
	Genres       []string `json:"genres"`
	Overview     string   `json:"overview"`
	VoteAverage  float32  `json:"vote_average"`
	PosterPath   *string  `json:"poster_path"`
	BackdropPath *string  `json:"backdrop_path"`
}

type TVDetail struct {
	ID           uint64          `json:"id"`
	IMDBID       string          `json:"imdb_id"`
	Name         string          `json:"name"`
	Year         uint16          `json:"year"`
	Seasons      []SeasonSummary `json:"seasons"`
	Genres       []string        `json:"genres"`
	Overview     string          `json:"overview"`
	VoteAverage  float32         `json:"vote_average"`
	PosterPath   *string         `json:"poster_path"`
	BackdropPath *string         `json:"backdrop_path"`
}

type Episode struct {
	Season   uint8   `json:"season"`
	Episode  uint16  `json:"episode"`
	Name     string  `json:"name"`
	Overview string  `json:"overview"`
	Runtime  *uint32 `json:"runtime"`
	IMDBID   *string `json:"imdb_id"`
}
