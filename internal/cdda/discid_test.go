package cdda

import (
	"testing"
)

func TestCalculateDiscID_KnownDisc(t *testing.T) {
	// Test with a verified disc ID from go-discid documentation
	// TOC: 1 11 242457 150 44942 61305 72755 96360 130485 147315 164275 190702 205412 220437
	// Disc ID: lSOVc5h6IXSuzcamJS1Gp4_tRuA-
	// https://pkg.go.dev/github.com/phw/go-discid

	toc := TOC{
		FirstTrack: 1,
		LastTrack:  11,
		LeadoutLBA: 242457,
		Tracks: []Track{
			{Num: 1, LBA: 150, Type: TrackTypeAudio},
			{Num: 2, LBA: 44942, Type: TrackTypeAudio},
			{Num: 3, LBA: 61305, Type: TrackTypeAudio},
			{Num: 4, LBA: 72755, Type: TrackTypeAudio},
			{Num: 5, LBA: 96360, Type: TrackTypeAudio},
			{Num: 6, LBA: 130485, Type: TrackTypeAudio},
			{Num: 7, LBA: 147315, Type: TrackTypeAudio},
			{Num: 8, LBA: 164275, Type: TrackTypeAudio},
			{Num: 9, LBA: 190702, Type: TrackTypeAudio},
			{Num: 10, LBA: 205412, Type: TrackTypeAudio},
			{Num: 11, LBA: 220437, Type: TrackTypeAudio},
		},
	}

	expected := "lSOVc5h6IXSuzcamJS1Gp4_tRuA-"
	got := CalculateDiscID(toc)

	if got != expected {
		t.Errorf("CalculateDiscID() = %q, want %q", got, expected)
	}
}

func TestCalculateDiscID_SingleTrack(t *testing.T) {
	// Single track disc - simpler case
	toc := TOC{
		FirstTrack: 1,
		LastTrack:  1,
		LeadoutLBA: 20000,
		Tracks: []Track{
			{Num: 1, LBA: 150, Type: TrackTypeAudio},
		},
	}

	// Just verify it returns a valid-looking disc ID (28 chars)
	got := CalculateDiscID(toc)
	if len(got) != 28 {
		t.Errorf("CalculateDiscID() returned %d chars, want 28", len(got))
	}

	// Should only contain valid base64 chars (with MusicBrainz substitutions)
	for _, c := range got {
		if !isValidDiscIDChar(c) {
			t.Errorf("CalculateDiscID() contains invalid char %q", c)
		}
	}
}

func TestCalculateDiscID_Deterministic(t *testing.T) {
	// Same TOC should always produce same disc ID
	toc := TOC{
		FirstTrack: 1,
		LastTrack:  3,
		LeadoutLBA: 54750,
		Tracks: []Track{
			{Num: 1, LBA: 150, Type: TrackTypeAudio},
			{Num: 2, LBA: 18250, Type: TrackTypeAudio},
			{Num: 3, LBA: 36500, Type: TrackTypeAudio},
		},
	}

	id1 := CalculateDiscID(toc)
	id2 := CalculateDiscID(toc)

	if id1 != id2 {
		t.Errorf("CalculateDiscID() not deterministic: %q != %q", id1, id2)
	}
}

func isValidDiscIDChar(c rune) bool {
	// MusicBrainz disc ID uses modified base64:
	// A-Z, a-z, 0-9, and . _ - (instead of + / =)
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	if c == '.' || c == '_' || c == '-' {
		return true
	}
	return false
}
