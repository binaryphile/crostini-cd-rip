package cdda

import (
	"testing"
)

func TestParseTOC_ValidLBAFormat(t *testing.T) {
	// READ TOC response in LBA format (byte 1 of CDB = 0x00)
	// This is a 3-track audio CD example
	raw := []byte{
		// Header: TOC length (big-endian), first track, last track
		// Length = 2 (first/last) + 4 tracks × 8 bytes = 34 = 0x22
		0x00, 0x22, // Length: 34 bytes (2 header bytes + 4 track entries including leadout)
		0x01,       // First track: 1
		0x03,       // Last track: 3

		// Track 1 entry (8 bytes)
		0x00,                   // Reserved
		0x00,                   // ADR/Control: 0x00 = audio, no pre-emphasis
		0x01,                   // Track number
		0x00,                   // Reserved
		0x00, 0x00, 0x00, 0x96, // LBA: 150 (standard 2-second pregap)

		// Track 2 entry (8 bytes)
		0x00,
		0x00,
		0x02,
		0x00,
		0x00, 0x00, 0x47, 0x4A, // LBA: 18250

		// Track 3 entry (8 bytes)
		0x00,
		0x00,
		0x03,
		0x00,
		0x00, 0x00, 0x8E, 0x94, // LBA: 36500

		// Lead-out entry (track 0xAA)
		0x00,
		0x00,
		0xAA,                   // Lead-out marker
		0x00,
		0x00, 0x00, 0xD5, 0xDE, // LBA: 54750
	}

	toc, err := ParseTOC(raw)
	if err != nil {
		t.Fatalf("ParseTOC failed: %v", err)
	}

	if toc.FirstTrack != 1 {
		t.Errorf("FirstTrack = %d, want 1", toc.FirstTrack)
	}
	if toc.LastTrack != 3 {
		t.Errorf("LastTrack = %d, want 3", toc.LastTrack)
	}
	if toc.LeadoutLBA != 54750 {
		t.Errorf("LeadoutLBA = %d, want 54750", toc.LeadoutLBA)
	}
	if len(toc.Tracks) != 3 {
		t.Fatalf("len(Tracks) = %d, want 3", len(toc.Tracks))
	}

	// Check track 1
	if toc.Tracks[0].Num != 1 {
		t.Errorf("Track[0].Num = %d, want 1", toc.Tracks[0].Num)
	}
	if toc.Tracks[0].LBA != 150 {
		t.Errorf("Track[0].LBA = %d, want 150", toc.Tracks[0].LBA)
	}
	if toc.Tracks[0].Type != TrackTypeAudio {
		t.Errorf("Track[0].Type = %v, want Audio", toc.Tracks[0].Type)
	}

	// Check track 2
	if toc.Tracks[1].Num != 2 {
		t.Errorf("Track[1].Num = %d, want 2", toc.Tracks[1].Num)
	}
	if toc.Tracks[1].LBA != 18250 {
		t.Errorf("Track[1].LBA = %d, want 18250", toc.Tracks[1].LBA)
	}

	// Check track 3
	if toc.Tracks[2].Num != 3 {
		t.Errorf("Track[2].Num = %d, want 3", toc.Tracks[2].Num)
	}
	if toc.Tracks[2].LBA != 36500 {
		t.Errorf("Track[2].LBA = %d, want 36500", toc.Tracks[2].LBA)
	}
}

func TestParseTOC_DataTrack(t *testing.T) {
	// CD with audio tracks and one data track (control bit 2 set)
	// Length = 2 (first/last) + 3 tracks × 8 bytes = 26 = 0x1A
	raw := []byte{
		0x00, 0x1A, // Length: 26 bytes
		0x01, 0x02, // Tracks 1-2

		// Track 1: Audio
		0x00, 0x00, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x96, // LBA: 150

		// Track 2: Data (control & 0x04 set)
		0x00, 0x04, 0x02, 0x00, // Control=0x04 means data track
		0x00, 0x00, 0x47, 0x4A, // LBA: 18250

		// Lead-out
		0x00, 0x00, 0xAA, 0x00,
		0x00, 0x00, 0x8E, 0x94, // LBA: 36500
	}

	toc, err := ParseTOC(raw)
	if err != nil {
		t.Fatalf("ParseTOC failed: %v", err)
	}

	if toc.Tracks[0].Type != TrackTypeAudio {
		t.Errorf("Track[0].Type = %v, want Audio", toc.Tracks[0].Type)
	}
	if toc.Tracks[1].Type != TrackTypeData {
		t.Errorf("Track[1].Type = %v, want Data", toc.Tracks[1].Type)
	}
}

func TestParseTOC_TooShort(t *testing.T) {
	// Invalid: less than 4 bytes header
	raw := []byte{0x00, 0x00, 0x01}

	_, err := ParseTOC(raw)
	if err == nil {
		t.Error("ParseTOC should fail on short input")
	}
}

func TestParseTOC_Empty(t *testing.T) {
	_, err := ParseTOC(nil)
	if err == nil {
		t.Error("ParseTOC should fail on nil input")
	}

	_, err = ParseTOC([]byte{})
	if err == nil {
		t.Error("ParseTOC should fail on empty input")
	}
}
