package audiobookshelf

type Library struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LibrariesResponse struct {
	Libraries []Library `json:"libraries"`
}

type Book struct {
	ID    string   `json:"id"`
	Media BookMedia `json:"media"`
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
}

type Chapter struct {
	ID    int     `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Title string  `json:"title"`
}

type BooksResponse struct {
	Results []Book `json:"results"`
}

type MediaProgress struct {
	ID          string  `json:"id"`
	CurrentTime float64 `json:"currentTime"`
	Duration    float64 `json:"duration"`
	Progress    float64 `json:"progress"`
	IsFinished  bool    `json:"isFinished"`
	UpdatedAt   int64   `json:"updatedAt"`
}

type MeResponse struct {
	MediaProgress []MediaProgress `json:"mediaProgress"`
}
