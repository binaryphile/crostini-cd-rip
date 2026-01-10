package metadata

import (
	"strings"
	"testing"
)

func TestParseJSON_StandardAlbum(t *testing.T) {
	album, err := ParseJSON("testdata/standard_album.json")
	if err != nil {
		t.Fatalf("ParseJSON error: %v", err)
	}
	if album.Artist != "Pink Floyd" {
		t.Errorf("Artist = %q, want %q", album.Artist, "Pink Floyd")
	}
	if album.AlbumTitle != "The Dark Side of the Moon" {
		t.Errorf("AlbumTitle = %q, want %q", album.AlbumTitle, "The Dark Side of the Moon")
	}
	if album.Year != "1973" {
		t.Errorf("Year = %q, want %q", album.Year, "1973")
	}
	if album.Genre != "Progressive Rock" {
		t.Errorf("Genre = %q, want %q", album.Genre, "Progressive Rock")
	}
	if len(album.Tracks) != 4 {
		t.Errorf("len(Tracks) = %d, want 4", len(album.Tracks))
	}
	if album.Tracks[0].Title != "Speak to Me" {
		t.Errorf("Tracks[0].Title = %q, want %q", album.Tracks[0].Title, "Speak to Me")
	}
}

func TestParseJSON_Compilation(t *testing.T) {
	album, err := ParseJSON("testdata/compilation.json")
	if err != nil {
		t.Fatalf("ParseJSON error: %v", err)
	}
	if album.Artist != "Various Artists" {
		t.Errorf("Artist = %q, want %q", album.Artist, "Various Artists")
	}
	if album.Disc != 2 {
		t.Errorf("Disc = %d, want 2", album.Disc)
	}
	if album.TotalDiscs != 2 {
		t.Errorf("TotalDiscs = %d, want 2", album.TotalDiscs)
	}
	if len(album.Tracks) < 1 {
		t.Fatal("expected at least 1 track")
	}
	if album.Tracks[0].Artist != "C + C Music Factory" {
		t.Errorf("Tracks[0].Artist = %q, want %q", album.Tracks[0].Artist, "C + C Music Factory")
	}
}

func TestParseJSON_MultiDisc(t *testing.T) {
	album, err := ParseJSON("testdata/multi_disc.json")
	if err != nil {
		t.Fatalf("ParseJSON error: %v", err)
	}
	if album.Disc != 1 {
		t.Errorf("Disc = %d, want 1", album.Disc)
	}
	if album.TotalDiscs != 2 {
		t.Errorf("TotalDiscs = %d, want 2", album.TotalDiscs)
	}
	if album.TotalTracks != 17 {
		t.Errorf("TotalTracks = %d, want 17", album.TotalTracks)
	}
}

func TestParseJSON_MissingArtist(t *testing.T) {
	album, err := ParseJSON("testdata/missing_artist.json")
	if err != nil {
		t.Fatalf("ParseJSON error: %v", err)
	}
	errs := album.Validate(3)
	if len(errs) == 0 {
		t.Error("expected validation error for missing artist")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "artist") {
			found = true
		}
	}
	if !found {
		t.Error("expected error mentioning 'artist'")
	}
}

func TestParseJSON_MissingAlbum(t *testing.T) {
	album := &Album{
		Artist: "Test",
		Tracks: []Track{{Num: 1, Title: "Song"}},
	}
	errs := album.Validate(1)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "album") {
			found = true
		}
	}
	if !found {
		t.Error("expected error mentioning 'album'")
	}
}

func TestParseJSON_MissingTracks(t *testing.T) {
	album := &Album{
		Artist:     "Test",
		AlbumTitle: "Test Album",
		Tracks:     []Track{},
	}
	errs := album.Validate(0)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "tracks") {
			found = true
		}
	}
	if !found {
		t.Error("expected error mentioning 'tracks'")
	}
}

func TestParseJSON_MalformedJSON(t *testing.T) {
	_, err := ParseJSON("testdata/malformed.json")
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestToRelease_FieldMapping(t *testing.T) {
	album := &Album{
		Artist:     "Pink Floyd",
		AlbumTitle: "The Wall",
		Year:       "1979",
		TotalDiscs: 2,
		Tracks: []Track{
			{Num: 1, Title: "In the Flesh?"},
			{Num: 2, Title: "The Thin Ice"},
		},
	}
	release := album.ToRelease()

	if release.Artist != "Pink Floyd" {
		t.Errorf("Artist = %q, want %q", release.Artist, "Pink Floyd")
	}
	if release.Title != "The Wall" {
		t.Errorf("Title = %q, want %q", release.Title, "The Wall")
	}
	if release.Year != 1979 {
		t.Errorf("Year = %d, want 1979", release.Year)
	}
	if release.DiscCount != 2 {
		t.Errorf("DiscCount = %d, want 2", release.DiscCount)
	}
	if release.TrackCount != 2 {
		t.Errorf("TrackCount = %d, want 2", release.TrackCount)
	}
	if len(release.Tracks) != 2 {
		t.Errorf("len(Tracks) = %d, want 2", len(release.Tracks))
	}
	if release.Tracks[0].Title != "In the Flesh?" {
		t.Errorf("Tracks[0].Title = %q, want %q", release.Tracks[0].Title, "In the Flesh?")
	}
}

func TestToRelease_Compilation(t *testing.T) {
	album := &Album{
		Artist:     "Various Artists",
		AlbumTitle: "Hits",
		Tracks:     []Track{{Num: 1, Title: "Song", Artist: "Someone"}},
	}
	release := album.ToRelease()
	if !release.Compilation {
		t.Error("Compilation should be true for Various Artists")
	}
}

func TestToRelease_NotCompilation(t *testing.T) {
	album := &Album{
		Artist:     "Pink Floyd",
		AlbumTitle: "The Wall",
		Tracks:     []Track{{Num: 1, Title: "Song"}},
	}
	release := album.ToRelease()
	if release.Compilation {
		t.Error("Compilation should be false for single artist")
	}
}

func TestValidate_TrackCountMismatch(t *testing.T) {
	album := &Album{
		Artist:     "Test",
		AlbumTitle: "Test Album",
		Tracks:     []Track{{Num: 1, Title: "Song"}},
	}
	errs := album.Validate(5) // 5 WAV files but only 1 track
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "mismatch") {
			found = true
		}
	}
	if !found {
		t.Error("expected error mentioning 'mismatch'")
	}
}

func TestValidate_MissingTrackArtist(t *testing.T) {
	album := &Album{
		Artist:     "Various Artists",
		AlbumTitle: "Compilation",
		Tracks: []Track{
			{Num: 1, Title: "Song 1", Artist: "Artist 1"},
			{Num: 2, Title: "Song 2", Artist: ""}, // missing artist
			{Num: 3, Title: "Song 3", Artist: "Artist 3"},
		},
	}
	errs := album.Validate(3)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "track 2") && strings.Contains(e.Error(), "artist") {
			found = true
		}
	}
	if !found {
		t.Error("expected error about track 2 missing artist")
	}
}

func TestValidate_AllFieldsPresent(t *testing.T) {
	album := &Album{
		Artist:     "Pink Floyd",
		AlbumTitle: "The Wall",
		Tracks:     []Track{{Num: 1, Title: "Song"}},
	}
	errs := album.Validate(1)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestLoadCoverArt_JPEG(t *testing.T) {
	album := &Album{CoverArt: "testdata/cover.jpg"}
	data, mime, err := album.LoadCoverArt()
	if err != nil {
		t.Fatalf("LoadCoverArt error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
	if mime != "image/jpeg" {
		t.Errorf("MIME = %q, want %q", mime, "image/jpeg")
	}
}

func TestLoadCoverArt_PNG(t *testing.T) {
	album := &Album{CoverArt: "testdata/cover.png"}
	data, mime, err := album.LoadCoverArt()
	if err != nil {
		t.Fatalf("LoadCoverArt error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
	if mime != "image/png" {
		t.Errorf("MIME = %q, want %q", mime, "image/png")
	}
}

func TestLoadCoverArt_NotFound(t *testing.T) {
	album := &Album{CoverArt: "testdata/nonexistent.jpg"}
	_, _, err := album.LoadCoverArt()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadCoverArt_Empty(t *testing.T) {
	album := &Album{CoverArt: ""}
	data, mime, err := album.LoadCoverArt()
	if err != nil {
		t.Fatalf("LoadCoverArt error: %v", err)
	}
	if data != nil {
		t.Error("expected nil data for empty CoverArt")
	}
	if mime != "" {
		t.Error("expected empty MIME for empty CoverArt")
	}
}
