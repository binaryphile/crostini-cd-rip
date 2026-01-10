package metadata

import (
	"strings"
	"testing"
)

// Example shows typical usage of the metadata package.
func Example() {
	album, err := ParseJSON("testdata/standard_album.json")
	if err != nil {
		panic(err)
	}

	// Validate against WAV file count
	if errs := album.Validate(4); len(errs) > 0 {
		for _, e := range errs {
			println("Warning:", e.Error())
		}
	}

	// Convert to Release for encoding pipeline
	release := album.ToRelease()
	_ = release
}

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

func TestValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		album    *Album
		wavCount int
		wantErr  string
	}{
		{
			name: "missing artist",
			album: &Album{
				AlbumTitle: "Test Album",
				Tracks:     []Track{{Num: 1, Title: "Song"}},
			},
			wavCount: 1,
			wantErr:  "artist",
		},
		{
			name: "missing album",
			album: &Album{
				Artist: "Test",
				Tracks: []Track{{Num: 1, Title: "Song"}},
			},
			wavCount: 1,
			wantErr:  "album",
		},
		{
			name: "missing tracks",
			album: &Album{
				Artist:     "Test",
				AlbumTitle: "Test Album",
				Tracks:     []Track{},
			},
			wavCount: 0,
			wantErr:  "tracks",
		},
		{
			name: "track count mismatch",
			album: &Album{
				Artist:     "Test",
				AlbumTitle: "Test Album",
				Tracks:     []Track{{Num: 1, Title: "Song"}},
			},
			wavCount: 5,
			wantErr:  "mismatch",
		},
		{
			name: "compilation missing track artist",
			album: &Album{
				Artist:     "Various Artists",
				AlbumTitle: "Compilation",
				Tracks: []Track{
					{Num: 1, Title: "Song 1", Artist: "Artist 1"},
					{Num: 2, Title: "Song 2", Artist: ""},
				},
			},
			wavCount: 2,
			wantErr:  "track 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.album.Validate(tt.wavCount)
			if len(errs) == 0 {
				t.Fatalf("expected validation error containing %q", tt.wantErr)
			}
			found := false
			for _, e := range errs {
				if strings.Contains(e.Error(), tt.wantErr) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, errs)
			}
		})
	}
}

func TestParseJSON_MissingArtistFile(t *testing.T) {
	album, err := ParseJSON("testdata/missing_artist.json")
	if err != nil {
		t.Fatalf("ParseJSON error: %v", err)
	}
	errs := album.Validate(3)
	if len(errs) == 0 {
		t.Error("expected validation error for missing artist")
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
	tests := []struct {
		name   string
		artist string
		want   bool
	}{
		{"Various Artists is compilation", "Various Artists", true},
		{"single artist not compilation", "Pink Floyd", false},
		{"lowercase various not compilation", "various artists", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			album := &Album{
				Artist:     tt.artist,
				AlbumTitle: "Test",
				Tracks:     []Track{{Num: 1, Title: "Song"}},
			}
			release := album.ToRelease()
			if release.Compilation != tt.want {
				t.Errorf("Compilation = %v, want %v", release.Compilation, tt.want)
			}
		})
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

func TestLoadCoverArt(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantMIME string
		wantData bool
		wantErr  bool
	}{
		{"JPEG file", "testdata/cover.jpg", "image/jpeg", true, false},
		{"PNG file", "testdata/cover.png", "image/png", true, false},
		{"not found", "testdata/nonexistent.jpg", "", false, true},
		{"empty path", "", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			album := &Album{CoverArt: tt.path}
			data, mime, err := album.LoadCoverArt()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantData && len(data) == 0 {
				t.Error("expected non-empty data")
			}
			if !tt.wantData && data != nil {
				t.Error("expected nil data")
			}
			if mime != tt.wantMIME {
				t.Errorf("MIME = %q, want %q", mime, tt.wantMIME)
			}
		})
	}
}

// Benchmarks

func BenchmarkParseJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseJSON("testdata/standard_album.json")
	}
}

func BenchmarkToRelease(b *testing.B) {
	album, _ := ParseJSON("testdata/standard_album.json")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = album.ToRelease()
	}
}

func BenchmarkValidate(b *testing.B) {
	album, _ := ParseJSON("testdata/standard_album.json")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = album.Validate(4)
	}
}
