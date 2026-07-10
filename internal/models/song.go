package models

type SourceType string

const (
	SourceYouTube   SourceType = "youtube"
	SourceRadio     SourceType = "radio"
	SourceNavidrome SourceType = "navidrome"
	SourceLocal     SourceType = "local"
)

type Song struct {
	ID        string     `json:"id"`
	Source    SourceType `json:"source"`
	SourceID  string     `json:"source_id,omitempty"`
	Title     string     `json:"title"`
	Artist    string     `json:"artist,omitempty"`
	Album     string     `json:"album,omitempty"`
	Duration  int        `json:"duration,omitempty"`
	URL       string     `json:"url,omitempty"`
	Thumbnail string     `json:"thumbnail,omitempty"`
	Bitrate   int        `json:"bitrate,omitempty"`
	Format    string     `json:"format,omitempty"`
	FilePath  string     `json:"file_path,omitempty"`
	Score     float64    `json:"-"`
	Genre     string     `json:"genre,omitempty"`
	Year      int        `json:"year,omitempty"`
}

func (s Song) DisplayTitle() string {
	if s.Title != "" {
		return s.Title
	}
	if s.FilePath != "" {
		return s.FilePath
	}
	return s.ID
}

func (s Song) IsStream() bool {
	return s.Duration <= 0
}

// Key returns a stable identity for deduping favorites/history.
func (s Song) Key() string {
	if s.ID != "" {
		return s.ID
	}
	if s.SourceID != "" {
		return string(s.Source) + ":" + s.SourceID
	}
	if s.URL != "" {
		return string(s.Source) + ":" + s.URL
	}
	if s.FilePath != "" {
		return "local:" + s.FilePath
	}
	return string(s.Source) + ":" + s.Title + ":" + s.Artist
}
