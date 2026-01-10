package encode

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// GenerateFilename creates a filename from track metadata.
// This is a pure function: (artist, album, disc, track, title) → filename
//
// Format: Artist-Album-NN-Title.mp3
// Multi-disc: Artist-Album-CDN-NN-Title.mp3
//
// Character handling:
// - Non-ASCII → normalized to ASCII equivalents (ō→o, é→e)
// - Spaces → underscores
// - / and \ → underscores (filesystem-illegal)
// - Shell metacharacters ($ ! * ? & ; | < > etc.) → underscores
// - Quotes (' " `) → removed
// - Multiple consecutive underscores → collapsed to single underscore
// - Leading/trailing underscores → trimmed
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
// Normalizes non-ASCII characters to ASCII equivalents (ō→o, é→e, etc.).
// Collapses multiple consecutive underscores to a single underscore.
func sanitize(s string) string {
	// First normalize non-ASCII to ASCII equivalents
	s = normalizeToASCII(s)

	var b strings.Builder
	b.Grow(len(s))

	lastWasUnderscore := false
	for _, r := range s {
		switch r {
		// Remove quotes (require shell escaping)
		case '\'', '"', '`':
			// skip - remove entirely

		// Replace with underscore
		case ' ': // space
			fallthrough
		case '/', '\\': // filesystem-illegal
			fallthrough
		case '$', '!': // shell expansion
			fallthrough
		case '*', '?', '[', ']': // glob patterns
			fallthrough
		case '(', ')': // subshell
			fallthrough
		case '{', '}': // brace expansion
			fallthrough
		case '<', '>', '|': // redirection/pipe
			fallthrough
		case '&', ';': // background/separator
			if !lastWasUnderscore {
				b.WriteByte('_')
				lastWasUnderscore = true
			}

		default:
			b.WriteRune(r)
			lastWasUnderscore = r == '_'
		}
	}

	// Trim leading/trailing underscores
	return strings.Trim(b.String(), "_")
}

// normalizeToASCII converts non-ASCII characters to their ASCII equivalents.
// Uses NFKD normalization to decompose characters (ō→o, é→e, etc.)
// and strips any remaining non-ASCII characters.
func normalizeToASCII(s string) string {
	t := transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)))
	result, _, _ := transform.String(t, s)

	// Strip any remaining non-ASCII
	var b strings.Builder
	for _, r := range result {
		if r < 128 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
