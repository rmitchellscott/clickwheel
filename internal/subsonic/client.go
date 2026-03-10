package subsonic

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	serverURL string
	username  string
	password  string
	http      *http.Client
}

func NewClient(serverURL, username, password string) *Client {
	return &Client{
		serverURL: serverURL,
		username:  username,
		password:  password,
		http:      &http.Client{},
	}
}

func (c *Client) authParams() url.Values {
	salt := strconv.FormatInt(rand.Int63(), 36)
	token := fmt.Sprintf("%x", md5.Sum([]byte(c.password+salt)))
	return url.Values{
		"u": {c.username},
		"t": {token},
		"s": {salt},
		"v": {"1.16.1"},
		"c": {"clickwheel"},
		"f": {"json"},
	}
}

func (c *Client) get(endpoint string, params url.Values) (*SubsonicResponse, error) {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return nil, err
	}
	u.Path += "/rest/" + endpoint

	q := c.authParams()
	for k, v := range params {
		q[k] = v
	}
	u.RawQuery = q.Encode()

	resp, err := c.http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sr SubsonicResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	if sr.SubsonicResponse.Error != nil {
		return nil, fmt.Errorf("subsonic error %d: %s",
			sr.SubsonicResponse.Error.Code,
			sr.SubsonicResponse.Error.Message)
	}

	return &sr, nil
}

func (c *Client) Ping() error {
	_, err := c.get("ping", nil)
	return err
}

func (c *Client) GetPlaylists() ([]Playlist, error) {
	sr, err := c.get("getPlaylists", nil)
	if err != nil {
		return nil, err
	}
	if sr.SubsonicResponse.Playlists == nil {
		return nil, nil
	}
	return sr.SubsonicResponse.Playlists.Playlist, nil
}

func (c *Client) GetPlaylist(id string) (*PlaylistDetail, error) {
	sr, err := c.get("getPlaylist", url.Values{"id": {id}})
	if err != nil {
		return nil, err
	}
	return sr.SubsonicResponse.Playlist, nil
}

func (c *Client) GetAlbums(offset, size int) ([]Album, error) {
	sr, err := c.get("getAlbumList2", url.Values{
		"type":   {"alphabeticalByName"},
		"size":   {strconv.Itoa(size)},
		"offset": {strconv.Itoa(offset)},
	})
	if err != nil {
		return nil, err
	}
	if sr.SubsonicResponse.AlbumList2 == nil {
		return nil, nil
	}
	return sr.SubsonicResponse.AlbumList2.Album, nil
}

func (c *Client) GetArtists() ([]Artist, error) {
	sr, err := c.get("getArtists", nil)
	if err != nil {
		return nil, err
	}
	if sr.SubsonicResponse.Artists == nil {
		return nil, nil
	}
	var artists []Artist
	for _, idx := range sr.SubsonicResponse.Artists.Index {
		artists = append(artists, idx.Artist...)
	}
	return artists, nil
}

func (c *Client) GetArtist(id string) (*ArtistDetail, error) {
	sr, err := c.get("getArtist", url.Values{"id": {id}})
	if err != nil {
		return nil, err
	}
	return sr.SubsonicResponse.Artist, nil
}

func (c *Client) GetAlbum(id string) (*AlbumDetail, error) {
	sr, err := c.get("getAlbum", url.Values{"id": {id}})
	if err != nil {
		return nil, err
	}
	return sr.SubsonicResponse.Album, nil
}

func (c *Client) Stream(songID, format string, maxBitRate int, w io.Writer) error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return err
	}
	u.Path += "/rest/stream"

	if format == "" {
		format = "aac"
	}
	if maxBitRate <= 0 {
		maxBitRate = 256
	}

	q := c.authParams()
	q.Set("id", songID)
	q.Set("format", format)
	q.Set("maxBitRate", fmt.Sprintf("%d", maxBitRate))
	u.RawQuery = q.Encode()

	resp, err := c.http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stream failed: %s", resp.Status)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

func (c *Client) Download(songID string, w io.Writer) error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return err
	}
	u.Path += "/rest/download"

	q := c.authParams()
	q.Set("id", songID)
	u.RawQuery = q.Encode()

	resp, err := c.http.Get(u.String())
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

func (c *Client) Scrobble(songID string) error {
	_, err := c.get("scrobble", url.Values{"id": {songID}})
	return err
}

func (c *Client) GetCoverArt(coverArtID string, size int) ([]byte, error) {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return nil, err
	}
	u.Path += "/rest/getCoverArt"

	q := c.authParams()
	q.Set("id", coverArtID)
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}
	u.RawQuery = q.Encode()

	resp, err := c.http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getCoverArt failed: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
