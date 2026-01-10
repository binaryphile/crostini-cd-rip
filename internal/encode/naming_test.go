package encode

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateFilename_Basic(t *testing.T) {
	got := GenerateFilename("Artist", "Album", 0, 1, "Song Title")
	want := "Artist-Album-01-Song_Title.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_SpacesToUnderscores(t *testing.T) {
	got := GenerateFilename("The Beatles", "Abbey Road", 0, 2, "Come Together")
	want := "The_Beatles-Abbey_Road-02-Come_Together.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_SlashReplaced(t *testing.T) {
	// AC/DC should become AC_DC (slash is illegal in filenames)
	got := GenerateFilename("AC/DC", "Back in Black", 0, 1, "Hells Bells")
	want := "AC_DC-Back_in_Black-01-Hells_Bells.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_BackslashReplaced(t *testing.T) {
	got := GenerateFilename("Test\\Artist", "Test\\Album", 0, 1, "Test\\Title")
	want := "Test_Artist-Test_Album-01-Test_Title.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_RemovesApostrophe(t *testing.T) {
	// Apostrophes removed for shell safety (no quoting needed)
	got := GenerateFilename("The Who", "Who's Next", 0, 3, "Won't Get Fooled Again")
	want := "The_Who-Whos_Next-03-Wont_Get_Fooled_Again.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_RemovesDoubleQuotes(t *testing.T) {
	// Double quotes removed for shell safety
	got := GenerateFilename(`Richard "Groove" Holmes`, "Album", 0, 1, "Song")
	want := "Richard_Groove_Holmes-Album-01-Song.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_RemovesSmartQuotes(t *testing.T) {
	// Smart/curly quotes are non-ASCII so they get removed
	got := GenerateFilename("Artist", "Album", 0, 1, "\u201cSmart\u201d \u2018Quotes\u2019")
	want := "Artist-Album-01-Smart_Quotes.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_ShellSafe(t *testing.T) {
	// Verify shell-special characters are replaced, trailing underscores trimmed
	got := GenerateFilename("Test$Artist", "Album!", 0, 1, "Song?")
	want := "Test_Artist-Album-01-Song.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}

	// Verify result doesn't need escaping (simulated check)
	escaped := shellEscape(got)
	if escaped != got {
		t.Errorf("Filename requires escaping: %q -> %q", got, escaped)
	}
}

// shellEscape simulates what printf %q does for simple cases
func shellEscape(s string) string {
	needsEscape := false
	for _, c := range s {
		switch c {
		case '\'', '"', '`', '$', '!', '*', '?', '[', ']', '(', ')', '{', '}', '<', '>', '|', '&', ';', ' ', '\\':
			needsEscape = true
		}
	}
	if needsEscape {
		return "NEEDS_ESCAPE:" + s
	}
	return s
}

func TestGenerateFilename_ShellSafe_Integration(t *testing.T) {
	// Skip if bash not available
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	// Test filenames with various problematic characters (pre-sanitization)
	testCases := []struct {
		artist string
		album  string
		title  string
	}{
		{"AC/DC", "Back in Black", "Hells Bells"},
		{"The Who", "Who's Next", "Won't Get Fooled Again"},
		{`Richard "Groove" Holmes`, "Album", "Song"},
		{"Test$Artist", "Album!", "Song?"},
		{"Artist", "Album [Deluxe]", "Track (Live)"},
		{"Artist", "Album", "Song & Dance"},
		{"Artist", "Album", "Part 1; Part 2"},
	}

	tmpDir := t.TempDir()

	for _, tc := range testCases {
		filename := GenerateFilename(tc.artist, tc.album, 0, 1, tc.title)
		fullPath := filepath.Join(tmpDir, filename)

		// Create the file
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Errorf("Failed to create file %q: %v", filename, err)
			continue
		}

		// Use bash to verify the filename works without quoting
		// bash -c 'cat $1' _ <filename> - passes filename as $1, no shell expansion
		// Then we also test: bash -c "cat $filename" which WOULD fail if filename needs quoting
		cmd := exec.Command("bash", "-c", "cat "+filename)
		cmd.Dir = tmpDir
		output, err := cmd.Output()
		if err != nil {
			t.Errorf("Filename %q requires shell quoting: %v", filename, err)
			continue
		}
		if string(output) != "test" {
			t.Errorf("Filename %q: unexpected output %q", filename, output)
		}
	}
}

func TestGenerateFilename_KeepsColon(t *testing.T) {
	// Colons are kept (shell-safe, though illegal on Windows)
	got := GenerateFilename("Artist", "Album: Subtitle", 0, 1, "Song: Extended Mix")
	want := "Artist-Album:_Subtitle-01-Song:_Extended_Mix.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_MultiDisc(t *testing.T) {
	// Multi-disc: disc > 0 adds CDN prefix
	// Note: ? is replaced with underscore then trimmed
	got := GenerateFilename("Pink Floyd", "The Wall", 1, 1, "In the Flesh?")
	want := "Pink_Floyd-The_Wall-CD1-01-In_the_Flesh.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_MultiDiscSecond(t *testing.T) {
	got := GenerateFilename("Pink Floyd", "The Wall", 2, 13, "Another Brick in the Wall")
	want := "Pink_Floyd-The_Wall-CD2-13-Another_Brick_in_the_Wall.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_TrackPadding(t *testing.T) {
	// Single digit tracks should be zero-padded
	got := GenerateFilename("Artist", "Album", 0, 9, "Track Nine")
	want := "Artist-Album-09-Track_Nine.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_DoubleDigitTrack(t *testing.T) {
	got := GenerateFilename("Artist", "Album", 0, 12, "Track Twelve")
	want := "Artist-Album-12-Track_Twelve.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateCompilationFilename(t *testing.T) {
	// Compilation: different format with track artist included
	got := GenerateCompilationFilename("80s Hits", 0, 1, "A-ha", "Take On Me")
	want := "80s_Hits-01-A-ha-Take_On_Me.mp3"

	if got != want {
		t.Errorf("GenerateCompilationFilename() = %q, want %q", got, want)
	}
}

func TestGenerateCompilationFilename_MultiDisc(t *testing.T) {
	got := GenerateCompilationFilename("Now 100", 2, 5, "Queen", "Bohemian Rhapsody")
	want := "Now_100-CD2-05-Queen-Bohemian_Rhapsody.mp3"

	if got != want {
		t.Errorf("GenerateCompilationFilename() = %q, want %q", got, want)
	}
}

func TestGenerateCompilationFilename_SlashInArtist(t *testing.T) {
	got := GenerateCompilationFilename("Metal Hits", 0, 3, "AC/DC", "Highway to Hell")
	want := "Metal_Hits-03-AC_DC-Highway_to_Hell.mp3"

	if got != want {
		t.Errorf("GenerateCompilationFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_CollapsesUnderscores(t *testing.T) {
	// "Heavy D & The Boyz" has " & " which becomes "___" without collapsing
	got := GenerateFilename("Heavy D & The Boyz", "Album", 0, 1, "Song")
	want := "Heavy_D_The_Boyz-Album-01-Song.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_CollapsesUnderscores_Multiple(t *testing.T) {
	// Multiple consecutive special chars should collapse
	// Leading/trailing underscores are trimmed
	got := GenerateFilename("A & B", "C (D) [E]", 0, 1, "F / G")
	want := "A_B-C_D_E-01-F_G.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

func TestGenerateFilename_NonASCII(t *testing.T) {
	// Non-ASCII should be normalized to ASCII equivalents
	got := GenerateFilename("Tone-Lōc", "Album", 0, 1, "Café")
	want := "Tone-Loc-Album-01-Cafe.mp3"

	if got != want {
		t.Errorf("GenerateFilename() = %q, want %q", got, want)
	}
}

// BenchmarkSanitize measures the performance of filename sanitization.
// Run with: go test -bench=. -benchmem ./internal/encode/
func BenchmarkSanitize(b *testing.B) {
	// Representative inputs with various special characters
	inputs := []string{
		"Simple Artist",
		"AC/DC",
		"The Who's Greatest Hits",
		`Richard "Groove" Holmes`,
		"Test$Artist & Friends (Live) [Deluxe Edition]",
		"Complex: Title; Part 1 | Part 2 <Remix>",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			_ = GenerateFilename(input, "Album", 0, 1, "Title")
		}
	}
}

// BenchmarkSanitize_WorstCase tests sanitization with maximum special characters.
func BenchmarkSanitize_WorstCase(b *testing.B) {
	// Input with every special character that needs handling
	worstCase := `Artist's "Name" with $pecial! chars? [brackets] (parens) {braces} <angles> | pipe & amp; semi / slash \ backslash`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateFilename(worstCase, worstCase, 0, 1, worstCase)
	}
}
