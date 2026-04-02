package audiobookshelf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	serverURL string
	token     string
	http      *http.Client
}

func NewClient(serverURL, token string) *Client {
	return &Client{
		serverURL: serverURL,
		token:     token,
		http:      &http.Client{},
	}
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return nil, err
	}
	u.Path += path

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.http.Do(req)
}

func (c *Client) Ping() error {
	resp, err := c.doRequest("GET", "/api/libraries", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ABS connection failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) GetLibraries() ([]Library, error) {
	resp, err := c.doRequest("GET", "/api/libraries", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lr LibrariesResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return lr.Libraries, nil
}

func (c *Client) GetBooks(libraryID string) ([]Book, error) {
	resp, err := c.doRequest("GET", "/api/libraries/"+libraryID+"/items", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var br BooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&br); err != nil {
		return nil, err
	}

	if br.Total > 0 && len(br.Results) < br.Total {
		all := br.Results
		const pageSize = 100
		for page := 1; ; page++ {
			r, err := c.doRequest("GET", fmt.Sprintf("/api/libraries/%s/items?limit=%d&page=%d", libraryID, pageSize, page), nil)
			if err != nil {
				break
			}
			var next BooksResponse
			if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
				r.Body.Close()
				break
			}
			r.Body.Close()
			all = append(all, next.Results...)
			if len(all) >= br.Total || len(next.Results) < pageSize {
				break
			}
		}
		return all, nil
	}

	return br.Results, nil
}

func (c *Client) GetBook(itemID string) (*Book, error) {
	resp, err := c.doRequest("GET", "/api/items/"+itemID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get book failed: %s", resp.Status)
	}

	var b Book
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (c *Client) GetPodcasts(libraryID string) ([]Podcast, error) {
	resp, err := c.doRequest("GET", "/api/libraries/"+libraryID+"/items", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pr PodcastsResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}

	if pr.Total > 0 && len(pr.Results) < pr.Total {
		all := pr.Results
		const pageSize = 100
		for page := 1; ; page++ {
			r, err := c.doRequest("GET", fmt.Sprintf("/api/libraries/%s/items?limit=%d&page=%d", libraryID, pageSize, page), nil)
			if err != nil {
				break
			}
			var next PodcastsResponse
			if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
				r.Body.Close()
				break
			}
			r.Body.Close()
			all = append(all, next.Results...)
			if len(all) >= pr.Total || len(next.Results) < pageSize {
				break
			}
		}
		return all, nil
	}

	return pr.Results, nil
}

func (c *Client) GetPodcast(itemID string) (*Podcast, error) {
	resp, err := c.doRequest("GET", "/api/items/"+itemID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get podcast failed: %s", resp.Status)
	}

	var p Podcast
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) GetEpisodeProgress(itemID, episodeID string) (*MediaProgress, error) {
	resp, err := c.doRequest("GET", "/api/me/progress/"+itemID+"/"+episodeID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	var mp MediaProgress
	if err := json.NewDecoder(resp.Body).Decode(&mp); err != nil {
		return nil, err
	}
	return &mp, nil
}

func (c *Client) UpdateEpisodeProgress(itemID, episodeID string, currentTime, duration float64, isFinished bool) error {
	body := map[string]any{
		"currentTime": currentTime,
		"duration":    duration,
		"isFinished":  isFinished,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := c.doRequest("PATCH", "/api/me/progress/"+itemID+"/"+episodeID, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update episode progress failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) DownloadEpisodeFile(itemID, ino string, w io.Writer) error {
	resp, err := c.doRequest("GET", "/api/items/"+itemID+"/file/"+ino, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("episode download failed: %s", resp.Status)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

func (c *Client) GetProgress(itemID string) (*MediaProgress, error) {
	resp, err := c.doRequest("GET", "/api/me/progress/"+itemID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	var mp MediaProgress
	if err := json.NewDecoder(resp.Body).Decode(&mp); err != nil {
		return nil, err
	}
	return &mp, nil
}

func (c *Client) GetAllProgress() (map[string]MediaProgress, error) {
	resp, err := c.doRequest("GET", "/api/me", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var me MeResponse
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return nil, err
	}

	result := make(map[string]MediaProgress, len(me.MediaProgress))
	for _, mp := range me.MediaProgress {
		if mp.EpisodeID != "" {
			result[mp.LibraryItemID+"|"+mp.EpisodeID] = mp
		} else {
			result[mp.LibraryItemID] = mp
		}
	}
	return result, nil
}

func (c *Client) UpdateProgress(itemID string, currentTime, duration float64) error {
	body := map[string]interface{}{
		"currentTime": currentTime,
		"duration":    duration,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := c.doRequest("PATCH", "/api/me/progress/"+itemID, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update progress failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) DownloadFile(itemID string, w io.Writer) error {
	resp, err := c.doRequest("GET", "/api/items/"+itemID+"/download", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}
