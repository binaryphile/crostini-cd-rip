package musicbrainz

import (
	"testing"

	"go.uploadedlobster.com/musicbrainzws2"
)

func TestNewClient(t *testing.T) {
	// Smoke test: client can be created and closed
	client := NewClient("test-app", "1.0", "test@example.com")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.client == nil {
		t.Fatal("Inner client is nil")
	}

	// Should close without error
	if err := client.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestGetArtistName_Single(t *testing.T) {
	credit := musicbrainzws2.ArtistCredit{
		{Name: "The Beatles", JoinPhrase: ""},
	}

	got := getArtistName(credit)
	want := "The Beatles"

	if got != want {
		t.Errorf("getArtistName() = %q, want %q", got, want)
	}
}

func TestGetArtistName_Multiple(t *testing.T) {
	credit := musicbrainzws2.ArtistCredit{
		{Name: "Queen", JoinPhrase: " & "},
		{Name: "David Bowie", JoinPhrase: ""},
	}

	got := getArtistName(credit)
	want := "Queen & David Bowie"

	if got != want {
		t.Errorf("getArtistName() = %q, want %q", got, want)
	}
}

func TestGetArtistName_Empty(t *testing.T) {
	credit := musicbrainzws2.ArtistCredit{}

	got := getArtistName(credit)
	want := "Unknown Artist"

	if got != want {
		t.Errorf("getArtistName() = %q, want %q", got, want)
	}
}

func TestIsCompilation_True(t *testing.T) {
	credit := musicbrainzws2.ArtistCredit{
		{Name: "Various Artists", JoinPhrase: ""},
	}

	if !isCompilation(credit) {
		t.Error("isCompilation() = false, want true")
	}
}

func TestIsCompilation_False(t *testing.T) {
	credit := musicbrainzws2.ArtistCredit{
		{Name: "Pink Floyd", JoinPhrase: ""},
	}

	if isCompilation(credit) {
		t.Error("isCompilation() = true, want false")
	}
}

func TestIsCompilation_Empty(t *testing.T) {
	credit := musicbrainzws2.ArtistCredit{}

	if isCompilation(credit) {
		t.Error("isCompilation() = true for empty credit, want false")
	}
}

func TestGetTotalTracks(t *testing.T) {
	media := []musicbrainzws2.Medium{
		{TrackCount: 12},
		{TrackCount: 10},
	}

	got := getTotalTracks(media)
	want := 22

	if got != want {
		t.Errorf("getTotalTracks() = %d, want %d", got, want)
	}
}

func TestGetTotalTracks_Empty(t *testing.T) {
	media := []musicbrainzws2.Medium{}

	got := getTotalTracks(media)
	want := 0

	if got != want {
		t.Errorf("getTotalTracks() = %d, want %d", got, want)
	}
}

func TestGetTotalTracks_Single(t *testing.T) {
	media := []musicbrainzws2.Medium{
		{TrackCount: 8},
	}

	got := getTotalTracks(media)
	want := 8

	if got != want {
		t.Errorf("getTotalTracks() = %d, want %d", got, want)
	}
}

func TestSortReleasesByTrackMatch(t *testing.T) {
	releases := []Release{
		{Title: "Box Set", TrackCount: 38, Year: 2017},
		{Title: "Vol 2", TrackCount: 12, Year: 1994},
		{Title: "Vol 1 GB", TrackCount: 12, Year: 1992},
		{Title: "Vol 1 US", TrackCount: 12, Year: 1992},
		{Title: "Best Of", TrackCount: 13, Year: 2001},
	}

	// Sort for 12-track target
	sorted := SortReleasesByTrackMatch(releases, 12)

	// First 3 should be 12-track releases
	for i := 0; i < 3; i++ {
		if sorted[i].TrackCount != 12 {
			t.Errorf("sorted[%d].TrackCount = %d, want 12", i, sorted[i].TrackCount)
		}
	}

	// 12-track releases should be sorted by year (newest first)
	if sorted[0].Year != 1994 {
		t.Errorf("sorted[0].Year = %d, want 1994 (newest 12-track)", sorted[0].Year)
	}

	// Non-matching should come after
	if sorted[3].TrackCount == 12 {
		t.Error("sorted[3] should not be 12-track")
	}
}

func TestSortReleasesByTrackMatch_NoMatches(t *testing.T) {
	releases := []Release{
		{Title: "A", TrackCount: 10, Year: 2020},
		{Title: "B", TrackCount: 15, Year: 2019},
	}

	// Sort for 12-track target (no matches)
	sorted := SortReleasesByTrackMatch(releases, 12)

	// Should sort by year (newest first) when no matches
	if sorted[0].Year != 2020 {
		t.Errorf("sorted[0].Year = %d, want 2020", sorted[0].Year)
	}
}

func TestRelease_Struct(t *testing.T) {
	// Verify Release struct can hold expected data
	r := Release{
		MBID:        "12345678-1234-1234-1234-123456789012",
		Title:       "Test Album",
		Artist:      "Test Artist",
		Year:        2024,
		Country:     "US",
		TrackCount:  12,
		DiscCount:   1,
		Compilation: false,
		Tracks: []Track{
			{Num: 1, Title: "Track One", Artist: "Test Artist"},
			{Num: 2, Title: "Track Two", Artist: "Test Artist"},
		},
	}

	if r.MBID == "" {
		t.Error("MBID is empty")
	}
	if len(r.Tracks) != 2 {
		t.Errorf("Tracks count = %d, want 2", len(r.Tracks))
	}
	if r.Tracks[0].Num != 1 {
		t.Errorf("Track 1 num = %d, want 1", r.Tracks[0].Num)
	}
}
