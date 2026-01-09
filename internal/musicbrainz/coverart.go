package musicbrainz

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var coverArtClient = &http.Client{
	Timeout: 10 * time.Second,
	// Default redirect policy is fine (follows up to 10 redirects)
}

// GetCoverArt fetches album cover from Cover Art Archive.
// Returns (data, mimeType, nil) on success.
// Returns (nil, "", nil) if not found (404).
// Returns (nil, "", error) on network/timeout errors.
func (c *Client) GetCoverArt(mbid string) ([]byte, string, error) {
	// Rate limit: 1 request per second (same as MusicBrainz)
	time.Sleep(1 * time.Second)

	url := fmt.Sprintf("https://coverartarchive.org/release/%s/front-250", mbid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("cover art request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := coverArtClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("cover art fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, "", nil // Not found, not an error
	}
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("cover art: HTTP %d", resp.StatusCode)
	}

	// Detect MIME type from response header
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg" // Default fallback
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("cover art read: %w", err)
	}

	return data, mimeType, nil
}
