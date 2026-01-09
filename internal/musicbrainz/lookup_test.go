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
