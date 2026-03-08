package subsonic

type SubsonicResponse struct {
	SubsonicResponse struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Playlists *struct {
			Playlist []Playlist `json:"playlist"`
		} `json:"playlists"`
		AlbumList2 *struct {
			Album []Album `json:"album"`
		} `json:"albumList2"`
		Playlist *PlaylistDetail `json:"playlist"`
		Artists  *struct {
			Index []ArtistIndex `json:"index"`
		} `json:"artists"`
		Artist *ArtistDetail `json:"artist"`
		Album  *AlbumDetail  `json:"album"`
	} `json:"subsonic-response"`
}

type Playlist struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
	CoverArt  string `json:"coverArt"`
}

type PlaylistDetail struct {
	Playlist
	Entry []Song `json:"entry"`
}

type Album struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Artist    string `json:"artist"`
	ArtistID  string `json:"artistId"`
	SongCount int    `json:"songCount"`
	Duration  int    `json:"duration"`
	Year      int    `json:"year"`
	CoverArt  string `json:"coverArt"`
}

type Artist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
}

type ArtistIndex struct {
	Name   string   `json:"name"`
	Artist []Artist `json:"artist"`
}

type ArtistDetail struct {
	Artist
	Album []Album `json:"album"`
}

type AlbumDetail struct {
	Album
	Song []Song `json:"song"`
}

type Song struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Album       string `json:"album"`
	Artist      string `json:"artist"`
	Track       int    `json:"track"`
	Year        int    `json:"year"`
	Genre       string `json:"genre"`
	Size        int64  `json:"size"`
	Duration    int    `json:"duration"`
	BitRate     int    `json:"bitRate"`
	ContentType string `json:"contentType"`
	Suffix      string `json:"suffix"`
	Path        string `json:"path"`
	PlayCount   int    `json:"playCount"`
}
