package encode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/binaryphile/crostini-cd-rip/internal/cdda"
)

func TestLameAvailable(t *testing.T) {
	// This test documents the dependency on lame
	// In CI without lame, this would need to be skipped
	if !LameAvailable() {
		t.Skip("lame not installed")
	}
}

func TestEncodeWAV_Integration(t *testing.T) {
	if !LameAvailable() {
		t.Skip("lame not installed")
	}

	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "encode-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a minimal WAV file (1/75th second of silence = 1 frame)
	samples := make([]byte, 2352) // One CD frame of silence
	wavData := cdda.WriteWAV(samples)

	wavPath := filepath.Join(tmpDir, "test.wav")
	if err := os.WriteFile(wavPath, wavData, 0644); err != nil {
		t.Fatal(err)
	}

	// Encode to MP3
	mp3Path := filepath.Join(tmpDir, "test.mp3")
	opts := DefaultEncodeOptions()

	err = EncodeWAV(wavPath, mp3Path, opts)
	if err != nil {
		t.Fatalf("EncodeWAV failed: %v", err)
	}

	// Verify MP3 was created
	info, err := os.Stat(mp3Path)
	if err != nil {
		t.Fatalf("MP3 not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("MP3 file is empty")
	}
}

func TestEncodeWAV_BadInput(t *testing.T) {
	if !LameAvailable() {
		t.Skip("lame not installed")
	}

	tmpDir, _ := os.MkdirTemp("", "encode-test-")
	defer os.RemoveAll(tmpDir)

	// Try to encode non-existent file
	err := EncodeWAV("/nonexistent/file.wav", filepath.Join(tmpDir, "out.mp3"), DefaultEncodeOptions())
	if err == nil {
		t.Error("Expected error for non-existent input")
	}
}

func TestDefaultEncodeOptions(t *testing.T) {
	opts := DefaultEncodeOptions()

	if opts.Quality != 2 {
		t.Errorf("Default quality = %d, want 2", opts.Quality)
	}
	if opts.Verbose {
		t.Error("Default verbose should be false")
	}
}
