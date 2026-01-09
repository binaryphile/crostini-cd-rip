package encode

import (
	"fmt"
	"strings"
)

// GenerateFilename creates a filename from track metadata.
// This is a pure function: (artist, album, disc, track, title) → filename
//
// Format: Artist-Album-NN-Title.mp3
// Multi-disc: Artist-Album-CDN-NN-Title.mp3
//
// Character handling (matches cd-overlap convention):
// - Spaces → underscores
// - / and \ → underscores (only illegal chars)
// - All other chars preserved (apostrophes, colons, etc.)
func GenerateFilename(artist, album string, disc, track int, title string) string {
	// Sanitize each component
	artist = sanitize(artist)
	album = sanitize(album)
	title = sanitize(title)

	// Build filename
	var parts []string
	parts = append(parts, artist, album)

	// Add disc number if multi-disc (disc > 0)
	if disc > 0 {
		parts = append(parts, fmt.Sprintf("CD%d", disc))
	}

	// Add track number (zero-padded)
	parts = append(parts, fmt.Sprintf("%02d", track))

	// Add title
	parts = append(parts, title)

	return strings.Join(parts, "-") + ".mp3"
}

// GenerateCompilationFilename creates a filename for compilation/various artists albums.
// This is a pure function.
//
// Format: Compilation-NN-TrackArtist-Title.mp3
// Multi-disc: Compilation-CDN-NN-TrackArtist-Title.mp3
func GenerateCompilationFilename(compilation string, disc, track int, trackArtist, title string) string {
	// Sanitize each component
	compilation = sanitize(compilation)
	trackArtist = sanitize(trackArtist)
	title = sanitize(title)

	// Build filename
	var parts []string
	parts = append(parts, compilation)

	// Add disc number if multi-disc (disc > 0)
	if disc > 0 {
		parts = append(parts, fmt.Sprintf("CD%d", disc))
	}

	// Add track number (zero-padded)
	parts = append(parts, fmt.Sprintf("%02d", track))

	// Add track artist and title
	parts = append(parts, trackArtist, title)

	return strings.Join(parts, "-") + ".mp3"
}

// sanitize prepares a string for use in a filename.
// Replaces characters that are illegal or require shell quoting.
func sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		switch r {
		// Remove quotes (require shell escaping)
		case '\'', '"', '`':
			// skip - remove entirely

		// Replace with underscore
		case ' ': // space
			b.WriteByte('_')
		case '/', '\\': // filesystem-illegal
			b.WriteByte('_')
		case '$', '!': // shell expansion
			b.WriteByte('_')
		case '*', '?', '[', ']': // glob patterns
			b.WriteByte('_')
		case '(', ')': // subshell
			b.WriteByte('_')
		case '{', '}': // brace expansion
			b.WriteByte('_')
		case '<', '>', '|': // redirection/pipe
			b.WriteByte('_')
		case '&', ';': // background/separator
			b.WriteByte('_')

		default:
			b.WriteRune(r)
		}
	}

	return b.String()
}
