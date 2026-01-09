package cdda

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"
)

// CalculateDiscID computes the MusicBrainz disc ID from a TOC.
// This is a pure function: TOC struct → 28-char disc ID string.
//
// Algorithm:
// 1. Format track data as hex ASCII string
// 2. SHA-1 hash the string
// 3. Base64 encode with MusicBrainz URL-safe substitutions
func CalculateDiscID(toc TOC) string {
	// Build the hex string that gets hashed
	// Format: "%02X%02X" + "%08X" * 100
	// - First track number (1 byte as 2 hex chars)
	// - Last track number (1 byte as 2 hex chars)
	// - 100 offsets as 8 hex chars each:
	//   - Index 0: leadout offset
	//   - Index 1-99: track offsets (0 for unused)

	var sb strings.Builder

	// First track and last track
	sb.WriteString(fmt.Sprintf("%02X", toc.FirstTrack))
	sb.WriteString(fmt.Sprintf("%02X", toc.LastTrack))

	// Build offset array: index 0 = leadout, index 1-99 = tracks
	offsets := make([]int, 100)
	offsets[0] = toc.LeadoutLBA

	for _, track := range toc.Tracks {
		if track.Num >= 1 && track.Num <= 99 {
			offsets[track.Num] = track.LBA
		}
	}

	// Write all 100 offsets
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("%08X", offsets[i]))
	}

	// SHA-1 hash
	hash := sha1.Sum([]byte(sb.String()))

	// Base64 encode with MusicBrainz substitutions
	encoded := base64.StdEncoding.EncodeToString(hash[:])

	// MusicBrainz uses URL-safe characters:
	// + → .
	// / → _
	// = → -
	encoded = strings.ReplaceAll(encoded, "+", ".")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	encoded = strings.ReplaceAll(encoded, "=", "-")

	return encoded
}
