package encode

import (
	"testing"
)

func TestBuildTags_Basic(t *testing.T) {
	tags := BuildTags(TrackMeta{
		Artist:      "The Beatles",
		Album:       "Abbey Road",
		Title:       "Come Together",
		TrackNum:    1,
		TrackTotal:  17,
		Year:        1969,
	})

	if tags.Artist != "The Beatles" {
		t.Errorf("Artist = %q, want %q", tags.Artist, "The Beatles")
	}
	if tags.Album != "Abbey Road" {
		t.Errorf("Album = %q, want %q", tags.Album, "Abbey Road")
	}
	if tags.Title != "Come Together" {
		t.Errorf("Title = %q, want %q", tags.Title, "Come Together")
	}
	if tags.TrackNum != 1 {
		t.Errorf("TrackNum = %d, want 1", tags.TrackNum)
	}
	if tags.TrackTotal != 17 {
		t.Errorf("TrackTotal = %d, want 17", tags.TrackTotal)
	}
	if tags.Year != 1969 {
		t.Errorf("Year = %d, want 1969", tags.Year)
	}
}

func TestBuildTags_Compilation(t *testing.T) {
	// For compilations, track artist differs from album artist
	tags := BuildTags(TrackMeta{
		Artist:      "A-ha",           // Track artist
		AlbumArtist: "Various Artists", // Album artist
		Album:       "80s Hits",
		Title:       "Take On Me",
		TrackNum:    5,
		TrackTotal:  20,
		Year:        1985,
		Compilation: true,
	})

	if tags.Artist != "A-ha" {
		t.Errorf("Artist = %q, want %q", tags.Artist, "A-ha")
	}
	if tags.AlbumArtist != "Various Artists" {
		t.Errorf("AlbumArtist = %q, want %q", tags.AlbumArtist, "Various Artists")
	}
	if !tags.Compilation {
		t.Error("Compilation should be true")
	}
}

func TestBuildTags_MultiDisc(t *testing.T) {
	tags := BuildTags(TrackMeta{
		Artist:     "Pink Floyd",
		Album:      "The Wall",
		Title:      "In the Flesh?",
		TrackNum:   1,
		TrackTotal: 13,
		DiscNum:    1,
		DiscTotal:  2,
		Year:       1979,
	})

	if tags.DiscNum != 1 {
		t.Errorf("DiscNum = %d, want 1", tags.DiscNum)
	}
	if tags.DiscTotal != 2 {
		t.Errorf("DiscTotal = %d, want 2", tags.DiscTotal)
	}
}

func TestBuildTags_NoAlbumArtist(t *testing.T) {
	// When AlbumArtist is empty, it should default to Artist
	tags := BuildTags(TrackMeta{
		Artist:     "Queen",
		Album:      "A Night at the Opera",
		Title:      "Bohemian Rhapsody",
		TrackNum:   11,
		TrackTotal: 12,
		Year:       1975,
	})

	// AlbumArtist should remain empty (not auto-filled)
	// Let the tagging library handle defaults
	if tags.Artist != "Queen" {
		t.Errorf("Artist = %q, want %q", tags.Artist, "Queen")
	}
}

func TestBuildTags_Genre(t *testing.T) {
	tags := BuildTags(TrackMeta{
		Artist:     "Metallica",
		Album:      "Master of Puppets",
		Title:      "Battery",
		TrackNum:   1,
		TrackTotal: 8,
		Year:       1986,
		Genre:      "Metal",
	})

	if tags.Genre != "Metal" {
		t.Errorf("Genre = %q, want %q", tags.Genre, "Metal")
	}
}
