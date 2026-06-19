package music

type NowPlaying struct {
	TrackID      string
	Track        string
	Artist       string
	Album        string
	ArtworkURL   string
	DurationMS   int64
	CurrentSec   float64
	RemainingSec float64
	ShuffleMode  int
	RepeatMode   int
}

type Track struct {
	ID         string
	Title      string
	Artist     string
	Album      string
	URL        string
	DurationMS int64
}

type Playlist struct {
	ID   string
	Name string
	URL  string
}

type SearchResult struct {
	ID         string
	Type       string
	Title      string
	Artist     string
	Album      string
	URL        string
	DurationMS int64
}

type SearchDetail struct {
	Type        string
	Title       string
	Subtitle    string
	Description string
	Tracks      []Track
}
