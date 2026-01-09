package musicbrainz

import (
	"context"
	"fmt"
	"time"

	"go.uploadedlobster.com/mbtypes"
	"go.uploadedlobster.com/musicbrainzws2"
)

// Release contains metadata for an album/release
type Release struct {
	MBID        string  // MusicBrainz ID
	Title       string  // Album title
	Artist      string  // Artist name (may be "Various Artists" for compilations)
	Year        int     // Release year
	Country     string  // Release country code
	TrackCount  int     // Number of tracks
	DiscCount   int     // Number of discs
	Tracks      []Track // Track list
	Compilation bool    // True if Various Artists
}

// Track contains metadata for a single track
type Track struct {
	Num    int
	Title  string
	Artist string // May differ from album artist on compilations
}

// Client wraps the MusicBrainz API
type Client struct {
	client *musicbrainzws2.Client
}

// NewClient creates a new MusicBrainz API client
func NewClient(appName, version, contact string) *Client {
	client := musicbrainzws2.NewClient(musicbrainzws2.AppInfo{
		Name:    appName,
		Version: version,
		URL:     contact,
	})
	return &Client{client: client}
}

// Close releases client resources
func (c *Client) Close() error {
	return c.client.Close()
}

// LookupByDiscID looks up releases by MusicBrainz disc ID.
// Returns a list of matching releases (may be multiple pressings/editions).
func (c *Client) LookupByDiscID(discID string) ([]Release, error) {
	// Rate limit: 1 request per second
	time.Sleep(1 * time.Second)

	ctx := context.Background()
	filter := musicbrainzws2.DiscIDFilter{
		Includes: []string{"recordings", "artists", "release-groups"},
	}

	disc, err := c.client.LookupDiscID(ctx, discID, filter)
	if err != nil {
		return nil, fmt.Errorf("disc lookup: %w", err)
	}

	var releases []Release
	for _, r := range disc.Releases {
		release := Release{
			MBID:        string(r.ID),
			Title:       r.Title,
			Artist:      getArtistName(r.ArtistCredit),
			Year:        r.Date.Year,
			Country:     string(r.CountryCode),
			TrackCount:  getTotalTracks(r.Media),
			DiscCount:   len(r.Media),
			Compilation: isCompilation(r.ArtistCredit),
		}
		releases = append(releases, release)
	}

	return releases, nil
}

// GetReleaseTracks fetches full track information for a release.
// Call this after selecting a release from LookupByDiscID.
func (c *Client) GetReleaseTracks(mbid string) (*Release, error) {
	// Rate limit
	time.Sleep(1 * time.Second)

	ctx := context.Background()
	filter := musicbrainzws2.IncludesFilter{
		Includes: []string{"recordings", "artists", "artist-credits"},
	}

	r, err := c.client.LookupRelease(ctx, mbtypes.MBID(mbid), filter)
	if err != nil {
		return nil, fmt.Errorf("release lookup: %w", err)
	}

	release := Release{
		MBID:        string(r.ID),
		Title:       r.Title,
		Artist:      getArtistName(r.ArtistCredit),
		Year:        r.Date.Year,
		Country:     string(r.CountryCode),
		TrackCount:  getTotalTracks(r.Media),
		DiscCount:   len(r.Media),
		Compilation: isCompilation(r.ArtistCredit),
	}

	// Extract tracks from all media
	for _, medium := range r.Media {
		for _, track := range medium.Tracks {
			t := Track{
				Num:    track.Position,
				Title:  track.Title,
				Artist: getTrackArtist(track, r.ArtistCredit),
			}
			release.Tracks = append(release.Tracks, t)
		}
	}

	return &release, nil
}

func getArtistName(credit musicbrainzws2.ArtistCredit) string {
	if len(credit) == 0 {
		return "Unknown Artist"
	}
	return credit.String()
}

func getTrackArtist(track musicbrainzws2.Track, albumCredit musicbrainzws2.ArtistCredit) string {
	// Use track's artist credit if present
	if len(track.ArtistCredit) > 0 {
		return track.ArtistCredit.String()
	}
	// Use recording's artist credit if different from album
	if len(track.Recording.ArtistCredit) > 0 {
		return track.Recording.ArtistCredit.String()
	}
	// Fall back to album artist
	return getArtistName(albumCredit)
}

func isCompilation(credit musicbrainzws2.ArtistCredit) bool {
	if len(credit) == 0 {
		return false
	}
	name := getArtistName(credit)
	return name == "Various Artists"
}

func getTotalTracks(media []musicbrainzws2.Medium) int {
	total := 0
	for _, m := range media {
		total += m.TrackCount
	}
	return total
}

// Search searches for releases by text query (artist, album, etc).
func (c *Client) Search(query string) ([]Release, error) {
	// Rate limit
	time.Sleep(1 * time.Second)

	ctx := context.Background()
	filter := musicbrainzws2.SearchFilter{
		Query: query,
	}

	paginator := musicbrainzws2.DefaultPaginator()

	result, err := c.client.SearchReleases(ctx, filter, paginator)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	var releases []Release
	for _, r := range result.Releases {
		release := Release{
			MBID:        string(r.ID),
			Title:       r.Title,
			Artist:      getArtistName(r.ArtistCredit),
			Year:        r.Date.Year,
			Country:     string(r.CountryCode),
			TrackCount:  getTotalTracks(r.Media),
			DiscCount:   len(r.Media),
			Compilation: isCompilation(r.ArtistCredit),
		}
		releases = append(releases, release)
	}

	return releases, nil
}
