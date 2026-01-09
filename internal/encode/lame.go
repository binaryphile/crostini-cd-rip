package encode

import (
	"fmt"
	"os"
	"os/exec"
)

// EncodeOptions configures the lame encoder
type EncodeOptions struct {
	Quality int  // VBR quality (0-9, lower is better, default 2)
	Verbose bool // Show lame output
}

// DefaultEncodeOptions returns sensible defaults for encoding
func DefaultEncodeOptions() EncodeOptions {
	return EncodeOptions{
		Quality: 2, // -V 2 is high quality VBR (~190 kbps)
		Verbose: false,
	}
}

// EncodeWAV encodes a WAV file to MP3 using lame.
// This is boundary code - calls external lame process.
//
// Returns the output file path on success.
func EncodeWAV(inputPath, outputPath string, opts EncodeOptions) error {
	// Validate input exists
	if _, err := os.Stat(inputPath); err != nil {
		return fmt.Errorf("input file: %w", err)
	}

	// Build lame command
	args := []string{
		fmt.Sprintf("-V%d", opts.Quality), // VBR quality
		"--quiet",                          // Suppress output unless verbose
		inputPath,
		outputPath,
	}

	if opts.Verbose {
		// Remove --quiet for verbose mode
		args = []string{
			fmt.Sprintf("-V%d", opts.Quality),
			inputPath,
			outputPath,
		}
	}

	cmd := exec.Command("lame", args...)

	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		// Clean up partial output on failure
		os.Remove(outputPath)
		return fmt.Errorf("lame encoding failed: %w", err)
	}

	// Verify output was created
	info, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("output file not created: %w", err)
	}
	if info.Size() == 0 {
		os.Remove(outputPath)
		return fmt.Errorf("output file is empty")
	}

	return nil
}

// LameAvailable checks if lame is installed and accessible
func LameAvailable() bool {
	_, err := exec.LookPath("lame")
	return err == nil
}
