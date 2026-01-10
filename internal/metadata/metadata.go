// Package metadata provides JSON parsing for manual album metadata input.
// Used when MusicBrainz lookup fails or returns ambiguous results.
// See docs/metadata-format.md for the JSON schema.
package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/binaryphile/crostini-cd-rip/internal/musicbrainz"
	"github.com/binaryphile/fluentfp/slice"
)

// Album represents album metadata from a JSON file.
type Album struct {
	Artist      string  `json:"artist"`
	AlbumTitle  string  `json:"album"`
	Year        string  `json:"year"`
	Genre       string  `json:"genre"`
	Disc        int     `json:"disc"`
	TotalDiscs  int     `json:"totalDiscs"`
	TotalTracks int     `json:"totalTracks"`
	CoverArt    string  `json:"coverArt"`
	Tracks      []Track `json:"tracks"`
}

// Track represents a single track in the album.
type Track struct {
	Num    int    `json:"num"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

// ParseJSON reads and parses a metadata JSON file.
func ParseJSON(path string) (*Album, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	var album Album
	if err := json.Unmarshal(data, &album); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	return &album, nil
}

// ToRelease converts Album to musicbrainz.Release for the encoding pipeline.
func (a *Album) ToRelease() *musicbrainz.Release {
	year, _ := strconv.Atoi(a.Year) // ignore error, default 0

	// slice.MapTo[R](input).To(fn) maps []Track → []musicbrainz.Track
	tracks := slice.MapTo[musicbrainz.Track](a.Tracks).To(func(t Track) musicbrainz.Track {
		return musicbrainz.Track{
			Num:    t.Num,
			Title:  t.Title,
			Artist: t.Artist,
		}
	})

	return &musicbrainz.Release{
		Title:       a.AlbumTitle,
		Artist:      a.Artist,
		Year:        year,
		TrackCount:  len(a.Tracks),
		DiscCount:   a.TotalDiscs,
		Tracks:      tracks,
		Compilation: a.Artist == "Various Artists",
	}
}

// Validate checks required fields and returns any validation errors.
// All issues are returned as warnings - caller decides whether to proceed.
func (a *Album) Validate(wavCount int) []error {
	var errs []error

	// Required field checks
	if a.Artist == "" {
		errs = append(errs, errors.New("missing required field: artist"))
	}
	if a.AlbumTitle == "" {
		errs = append(errs, errors.New("missing required field: album"))
	}
	if len(a.Tracks) == 0 {
		errs = append(errs, errors.New("missing required field: tracks"))
	}
	if len(a.Tracks) != wavCount {
		errs = append(errs, fmt.Errorf("track count mismatch: JSON has %d, found %d WAV files",
			len(a.Tracks), wavCount))
	}

	// Compilation track artist check
	if a.Artist == "Various Artists" {
		for i, t := range a.Tracks {
			if t.Artist == "" {
				errs = append(errs, fmt.Errorf("track %d missing artist (required for compilations)", i+1))
			}
		}
	}

	return errs
}

// LoadCoverArt reads the cover art file if specified.
// Returns (data, mimeType, error). Returns nil,nil,nil if CoverArt is empty.
func (a *Album) LoadCoverArt() ([]byte, string, error) {
	if a.CoverArt == "" {
		return nil, "", nil
	}
	data, err := os.ReadFile(a.CoverArt)
	if err != nil {
		return nil, "", fmt.Errorf("cover art: %w", err)
	}
	return data, detectMIME(a.CoverArt), nil
}

// detectMIME returns MIME type based on file extension.
func detectMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "image/jpeg" // fallback
	}
}
