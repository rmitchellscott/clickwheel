package restore

import (
	"archive/zip"
	"crypto/sha1"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"howett.net/plist"
)

//go:embed ipsw_catalog.json
var ipswCatalogData []byte

type IPSWEntry struct {
	Model    string `json:"model"`
	Variant  string `json:"variant,omitempty"`
	Version  string `json:"version"`
	Build    string `json:"build,omitempty"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	SHA1     string `json:"sha1,omitempty"`
}

var catalog []IPSWEntry

func init() {
	_ = json.Unmarshal(ipswCatalogData, &catalog)
}

func GetCatalog() []IPSWEntry {
	return catalog
}

func FindFirmware(identifier string) []IPSWEntry {
	var matches []IPSWEntry
	for _, e := range catalog {
		if strings.Contains(strings.ToLower(e.Model), strings.ToLower(identifier)) {
			matches = append(matches, e)
		}
	}
	return matches
}

func ipswCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "clickwheel", "ipsw")
	return p, os.MkdirAll(p, 0755)
}

func DownloadIPSW(entry IPSWEntry, onProgress func(downloaded, total int64)) (string, error) {
	cacheDir, err := ipswCacheDir()
	if err != nil {
		return "", fmt.Errorf("cache dir: %w", err)
	}

	destPath := filepath.Join(cacheDir, entry.Filename)

	if _, err := os.Stat(destPath); err == nil {
		if entry.SHA1 != "" {
			if ok, _ := verifySHA1(destPath, entry.SHA1); ok {
				return destPath, nil
			}
			os.Remove(destPath)
		} else {
			return destPath, nil
		}
	}

	resp, err := http.Get(entry.URL)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				f.Close()
				os.Remove(tmpPath)
				return "", err
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written, resp.ContentLength)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			f.Close()
			os.Remove(tmpPath)
			return "", readErr
		}
	}
	f.Close()

	if entry.SHA1 != "" {
		if ok, got := verifySHA1(tmpPath, entry.SHA1); !ok {
			os.Remove(tmpPath)
			return "", fmt.Errorf("SHA1 mismatch: expected %s, got %s", entry.SHA1, got)
		}
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}

func verifySHA1(path, expected string) (bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return false, ""
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, ""
	}

	got := hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(got, expected), got
}

type RestorePlist struct {
	FirmwareComponents map[string]FirmwareComponent `plist:"FirmwareComponents"`
}

type FirmwareComponent struct {
	Filename string `plist:"Filename"`
	Type     string `plist:"Type"`
}

func ParseRestorePlist(ipswPath string) (*RestorePlist, error) {
	r, err := zip.OpenReader(ipswPath)
	if err != nil {
		return nil, fmt.Errorf("open IPSW: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.EqualFold(f.Name, "Restore.plist") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("read Restore.plist: %w", err)
			}

			var rp RestorePlist
			if _, err := plist.Unmarshal(data, &rp); err != nil {
				return nil, fmt.Errorf("decode Restore.plist: %w", err)
			}
			return &rp, nil
		}
	}

	return nil, fmt.Errorf("Restore.plist not found in IPSW")
}

func ExtractFirmwareFile(ipswPath, filename string) ([]byte, error) {
	r, err := zip.OpenReader(ipswPath)
	if err != nil {
		return nil, fmt.Errorf("open IPSW: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.EqualFold(f.Name, filename) || strings.HasSuffix(f.Name, "/"+filename) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("file %q not found in IPSW", filename)
}
