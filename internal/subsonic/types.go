package subsonic

type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Album struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Artist    string `json:"artist"`
	ArtistID  string `json:"artistId"`
	Year      int    `json:"year"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
}

type Song struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	ArtistID   string `json:"artistId"`
	Album      string `json:"album"`
	AlbumID    string `json:"albumId"`
	Track      int    `json:"track"`
	Year       int    `json:"year"`
	Duration   int    `json:"duration"`
	BitRate    int    `json:"bitRate"`
	Genre      string `json:"genre"`
	Size       int64  `json:"size"`
	Suffix     string `json:"suffix"`
	Path       string `json:"path"`
	Created    string `json:"created"`
	AlbumArtID string `json:"coverArt"`
}

type Playlist struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
}
