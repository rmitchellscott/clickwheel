package audiobookshelf

type Library struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
}

type LibrariesResponse struct {
	Libraries []Library `json:"libraries"`
}

type Book struct {
	ID    string    `json:"id"`
	Media BookMedia `json:"media"`
	Size  int64     `json:"size"`
}

type BookMedia struct {
	Metadata   BookMetadata   `json:"metadata"`
	AudioFiles []AudioFile    `json:"audioFiles"`
	Chapters   []Chapter      `json:"chapters"`
	Duration   float64        `json:"duration"`
}

type BookMetadata struct {
	Title  string `json:"title"`
	Author string `json:"authorName"`
}

type AudioFile struct {
	Index    int     `json:"index"`
	Ino     string  `json:"ino"`
	Metadata FileMetadata `json:"metadata"`
	Duration float64 `json:"duration"`
	MimeType string  `json:"mimeType"`
}

type FileMetadata struct {
	Filename string `json:"filename"`
	Ext      string `json:"ext"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

type Chapter struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Title string  `json:"title"`
}

type BooksResponse struct {
	Results []Book `json:"results"`
	Total   int    `json:"total"`
}

type Podcast struct {
	ID    string       `json:"id"`
	Media PodcastMedia `json:"media"`
}

type PodcastMedia struct {
	Metadata    PodcastMetadata  `json:"metadata"`
	Episodes    []PodcastEpisode `json:"episodes"`
	NumEpisodes int              `json:"numEpisodes"`
	Size        int64            `json:"size"`
}

type PodcastMetadata struct {
	Title  string `json:"title"`
	Author string `json:"author"`
}

type PodcastEpisode struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Episode     string    `json:"episode"`
	Season      string    `json:"season"`
	AudioFile   AudioFile `json:"audioFile"`
	PublishedAt int64     `json:"publishedAt"`
}

type PodcastsResponse struct {
	Results []Podcast `json:"results"`
	Total   int       `json:"total"`
}

type MediaProgress struct {
	ID            string  `json:"id"`
	LibraryItemID string  `json:"libraryItemId"`
	EpisodeID     string  `json:"episodeId"`
	CurrentTime   float64 `json:"currentTime"`
	Duration      float64 `json:"duration"`
	Progress      float64 `json:"progress"`
	IsFinished    bool    `json:"isFinished"`
	LastUpdate    int64   `json:"lastUpdate"`
}

type MeResponse struct {
	MediaProgress []MediaProgress `json:"mediaProgress"`
}
