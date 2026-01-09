package cdda

import (
	"encoding/binary"
	"errors"
)

// TrackType indicates whether a track is audio or data
type TrackType int

const (
	TrackTypeAudio TrackType = iota
	TrackTypeData
)

// Track represents a single track from the CD TOC
type Track struct {
	Num  int
	LBA  int
	Type TrackType
}

// IsAudio returns true if this is an audio track
func (t Track) IsAudio() bool {
	return t.Type == TrackTypeAudio
}

// TOC represents a CD Table of Contents
type TOC struct {
	FirstTrack int
	LastTrack  int
	LeadoutLBA int
	Tracks     []Track
}

// ParseTOC parses raw bytes from a SCSI READ TOC command (LBA format).
// The raw bytes should be the response from READ TOC with format 0 and
// LBA addressing (CDB byte 1 = 0x00, not MSF format 0x02).
//
// This is a pure function: input bytes → TOC struct.
func ParseTOC(raw []byte) (TOC, error) {
	if len(raw) < 4 {
		return TOC{}, errors.New("TOC data too short: need at least 4 bytes")
	}

	// Header: 2-byte length (big-endian), first track, last track
	// Length includes the header bytes after the length field (i.e., first/last track + entries)
	tocLen := int(binary.BigEndian.Uint16(raw[0:2]))
	firstTrack := int(raw[2])
	lastTrack := int(raw[3])

	toc := TOC{
		FirstTrack: firstTrack,
		LastTrack:  lastTrack,
	}

	// Calculate how much data to parse: tocLen + 2 (length field itself)
	// But don't exceed what we actually have
	dataEnd := tocLen + 2
	if dataEnd > len(raw) {
		dataEnd = len(raw)
	}

	// Parse track entries (8 bytes each, starting at offset 4)
	offset := 4
	for offset+8 <= dataEnd {
		// Track entry format:
		// Byte 0: Reserved
		// Byte 1: ADR (upper 4 bits) / Control (lower 4 bits)
		// Byte 2: Track number (0xAA = lead-out)
		// Byte 3: Reserved
		// Bytes 4-7: LBA (big-endian)

		control := raw[offset+1]
		trackNum := int(raw[offset+2])
		lba := int(binary.BigEndian.Uint32(raw[offset+4 : offset+8]))

		if trackNum == 0xAA {
			// Lead-out marker
			toc.LeadoutLBA = lba
			offset += 8
			break // Lead-out is the last entry
		}

		// Skip invalid track numbers (should be between first and last)
		if trackNum >= firstTrack && trackNum <= lastTrack {
			trackType := TrackTypeAudio
			if control&0x04 != 0 {
				// Control bit 2 set = data track
				trackType = TrackTypeData
			}

			toc.Tracks = append(toc.Tracks, Track{
				Num:  trackNum,
				LBA:  lba,
				Type: trackType,
			})
		}

		offset += 8
	}

	return toc, nil
}
