package encode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/binaryphile/crostini-cd-rip/internal/cdda"
)

func TestTagSet_Apply_Integration(t *testing.T) {
	if !LameAvailable() {
		t.Skip("lame not installed")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tag-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a minimal WAV and encode to MP3
	samples := make([]byte, 2352)
	wavData := cdda.WriteWAV(samples)
	wavPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(wavPath, wavData, 0644); err != nil {
		t.Fatal(err)
	}

	mp3Path := filepath.Join(tmpDir, "test.mp3")
	if err := EncodeWAV(wavPath, mp3Path, DefaultEncodeOptions()); err != nil {
		t.Fatal(err)
	}

	// Apply tags
	tags := BuildTags(TrackMeta{
		Artist:     "Test Artist",
		Album:      "Test Album",
		Title:      "Test Song",
		TrackNum:   3,
		TrackTotal: 12,
		Year:       2024,
		Genre:      "Rock",
	})

	if err := tags.Apply(mp3Path); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Read back and verify
	tag, err := id3v2.Open(mp3Path, id3v2.Options{Parse: true})
	if err != nil {
		t.Fatalf("Failed to read tags: %v", err)
	}
	defer tag.Close()

	if tag.Artist() != "Test Artist" {
		t.Errorf("Artist = %q, want %q", tag.Artist(), "Test Artist")
	}
	if tag.Album() != "Test Album" {
		t.Errorf("Album = %q, want %q", tag.Album(), "Test Album")
	}
	if tag.Title() != "Test Song" {
		t.Errorf("Title = %q, want %q", tag.Title(), "Test Song")
	}
	if tag.Year() != "2024" {
		t.Errorf("Year = %q, want %q", tag.Year(), "2024")
	}
	if tag.Genre() != "Rock" {
		t.Errorf("Genre = %q, want %q", tag.Genre(), "Rock")
	}
}

func TestTagSet_Apply_Compilation(t *testing.T) {
	if !LameAvailable() {
		t.Skip("lame not installed")
	}

	tmpDir, _ := os.MkdirTemp("", "tag-test-")
	defer os.RemoveAll(tmpDir)

	// Create MP3
	samples := make([]byte, 2352)
	wavData := cdda.WriteWAV(samples)
	wavPath := filepath.Join(tmpDir, "test.wav")
	os.WriteFile(wavPath, wavData, 0644)

	mp3Path := filepath.Join(tmpDir, "test.mp3")
	EncodeWAV(wavPath, mp3Path, DefaultEncodeOptions())

	// Apply compilation tags
	tags := BuildTags(TrackMeta{
		Artist:      "Track Artist",
		AlbumArtist: "Various Artists",
		Album:       "Compilation",
		Title:       "Hit Song",
		TrackNum:    5,
		TrackTotal:  20,
		Compilation: true,
	})

	if err := tags.Apply(mp3Path); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Read back
	tag, _ := id3v2.Open(mp3Path, id3v2.Options{Parse: true})
	defer tag.Close()

	if tag.Artist() != "Track Artist" {
		t.Errorf("Artist = %q, want %q", tag.Artist(), "Track Artist")
	}

	// Check TPE2 (album artist)
	tpe2 := tag.GetTextFrame("TPE2")
	if tpe2.Text != "Various Artists" {
		t.Errorf("Album Artist = %q, want %q", tpe2.Text, "Various Artists")
	}

	// Check TCMP (compilation flag)
	tcmp := tag.GetTextFrame("TCMP")
	if tcmp.Text != "1" {
		t.Errorf("Compilation flag = %q, want %q", tcmp.Text, "1")
	}
}
