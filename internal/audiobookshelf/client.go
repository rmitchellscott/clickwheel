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
	return br.Results, nil
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
