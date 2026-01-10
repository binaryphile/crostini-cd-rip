package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for cd-encode --metadata flag.
// These tests use --dry-run to avoid needing lame or actual encoding.

func TestMetadata_StandardAlbum(t *testing.T) {
	dir := t.TempDir()

	// Create test WAV files
	for i := 1; i <= 3; i++ {
		f, _ := os.Create(filepath.Join(dir, fmt.Sprintf("track%02d.wav", i)))
		f.Close()
	}

	// Create metadata JSON
	metadata := `{
		"artist": "Test Artist",
		"album": "Test Album",
		"year": "2024",
		"disc": 1,
		"totalDiscs": 2,
		"tracks": [
			{"num": 1, "title": "Track One"},
			{"num": 2, "title": "Track Two"},
			{"num": 3, "title": "Track Three"}
		]
	}`
	metaPath := filepath.Join(dir, "metadata.json")
	os.WriteFile(metaPath, []byte(metadata), 0644)

	// Run cd-encode --metadata --dry-run
	cmd := exec.Command("go", "run", ".", "--metadata", metaPath, "--dry-run", dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cd-encode failed: %v\n%s", err, output)
	}

	out := string(output)

	// Verify output contains expected filenames with CD1 prefix
	if !strings.Contains(out, "CD1-01-Track_One.mp3") {
		t.Errorf("expected CD1-01-Track_One.mp3 in output:\n%s", out)
	}
	if !strings.Contains(out, "Test_Artist-Test_Album") {
		t.Errorf("expected Test_Artist-Test_Album in output:\n%s", out)
	}
}

func TestMetadata_Compilation(t *testing.T) {
	dir := t.TempDir()

	// Create test WAV files
	for i := 1; i <= 2; i++ {
		f, _ := os.Create(filepath.Join(dir, fmt.Sprintf("track%02d.wav", i)))
		f.Close()
	}

	// Create compilation metadata
	metadata := `{
		"artist": "Various Artists",
		"album": "Greatest Hits",
		"year": "2024",
		"tracks": [
			{"num": 1, "title": "Song One", "artist": "Artist A"},
			{"num": 2, "title": "Song Two", "artist": "Artist B"}
		]
	}`
	metaPath := filepath.Join(dir, "metadata.json")
	os.WriteFile(metaPath, []byte(metadata), 0644)

	cmd := exec.Command("go", "run", ".", "--metadata", metaPath, "--dry-run", dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cd-encode failed: %v\n%s", err, output)
	}

	out := string(output)

	// Verify compilation format: Album-NN-TrackArtist-Title
	if !strings.Contains(out, "Greatest_Hits-01-Artist_A-Song_One.mp3") {
		t.Errorf("expected compilation filename format in output:\n%s", out)
	}
}

func TestMetadata_StrictValidation(t *testing.T) {
	dir := t.TempDir()

	// Create 3 WAV files but only 2 tracks in JSON (mismatch)
	for i := 1; i <= 3; i++ {
		f, _ := os.Create(filepath.Join(dir, fmt.Sprintf("track%02d.wav", i)))
		f.Close()
	}

	metadata := `{
		"artist": "Test",
		"album": "Test",
		"tracks": [
			{"num": 1, "title": "One"},
			{"num": 2, "title": "Two"}
		]
	}`
	metaPath := filepath.Join(dir, "metadata.json")
	os.WriteFile(metaPath, []byte(metadata), 0644)

	// Without --strict: should warn but succeed
	cmd := exec.Command("go", "run", ".", "--metadata", metaPath, "--dry-run", dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("without --strict should succeed, got: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Warning") {
		t.Errorf("expected warning about track mismatch:\n%s", output)
	}

	// With --strict: should fail
	cmd = exec.Command("go", "run", ".", "--metadata", metaPath, "--strict", "--dry-run", dir)
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Errorf("with --strict should fail on mismatch:\n%s", output)
	}
	if !strings.Contains(string(output), "Validation failed") {
		t.Errorf("expected 'Validation failed' message:\n%s", output)
	}
}

func TestMetadata_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	// Create WAV file but no metadata file
	f, _ := os.Create(filepath.Join(dir, "track01.wav"))
	f.Close()

	cmd := exec.Command("go", "run", ".", "--metadata", "/nonexistent/metadata.json", "--dry-run", dir)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("should fail when metadata file not found:\n%s", output)
	}
	if !strings.Contains(string(output), "read metadata") {
		t.Errorf("expected 'read metadata' error message:\n%s", output)
	}
}

func TestMetadata_MalformedJSON(t *testing.T) {
	dir := t.TempDir()

	f, _ := os.Create(filepath.Join(dir, "track01.wav"))
	f.Close()

	// Write invalid JSON
	metaPath := filepath.Join(dir, "metadata.json")
	os.WriteFile(metaPath, []byte(`{"artist": "Test", invalid`), 0644)

	cmd := exec.Command("go", "run", ".", "--metadata", metaPath, "--dry-run", dir)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("should fail on malformed JSON:\n%s", output)
	}
	if !strings.Contains(string(output), "parse metadata") {
		t.Errorf("expected 'parse metadata' error message:\n%s", output)
	}
}
