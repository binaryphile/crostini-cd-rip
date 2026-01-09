package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/binaryphile/crostini-cd-rip/internal/encode"
	"github.com/binaryphile/crostini-cd-rip/internal/musicbrainz"
)

const (
	appName    = "cd-encode"
	appVersion = "1.0"
	appURL     = "https://github.com/binaryphile/crostini-cd-rip"
)

func main() {
	// Parse flags
	quality := flag.Int("q", 2, "LAME VBR quality (0-9, lower is better)")
	flag.IntVar(quality, "quality", 2, "LAME VBR quality")

	discIDFile := flag.String("discid", "", "Disc ID file (default: input-dir/discid.txt)")

	search := flag.String("search", "", "Manual album search instead of disc ID")

	dest := flag.String("dest", "", "Destination directory (default: ~/Music)")

	dryRun := flag.Bool("dry-run", false, "Show what would be done")

	verbose := flag.Bool("v", false, "Verbose output")
	flag.BoolVar(verbose, "verbose", false, "Verbose output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <input-dir>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Encode WAV files to MP3 with MusicBrainz metadata.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputDir := flag.Arg(0)

	// Validate input directory
	if _, err := os.Stat(inputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: input directory not found: %s\n", inputDir)
		os.Exit(1)
	}

	// Find WAV files
	wavFiles, err := findWAVFiles(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding WAV files: %v\n", err)
		os.Exit(1)
	}
	if len(wavFiles) == 0 {
		fmt.Fprintln(os.Stderr, "No WAV files found in input directory")
		os.Exit(1)
	}

	fmt.Printf("cd-encode - MusicBrainz lookup + lame encoding + ID3 tagging\n")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Input: %s (%d WAV files)\n", inputDir, len(wavFiles))

	// Get disc ID (not needed if using --search)
	var discID string
	if *search == "" {
		discIDPath := *discIDFile
		if discIDPath == "" {
			discIDPath = filepath.Join(inputDir, "discid.txt")
		}

		data, err := os.ReadFile(discIDPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No disc ID found. Use --search or create discid.txt\n")
			os.Exit(1)
		}
		discID = strings.TrimSpace(string(data))
		fmt.Printf("Disc ID: %s\n\n", discID)
	}

	// Check lame
	if !encode.LameAvailable() {
		fmt.Fprintln(os.Stderr, "Error: lame not found. Install with: nix-shell -p lame")
		os.Exit(1)
	}

	// Lookup on MusicBrainz
	client := musicbrainz.NewClient(appName, appVersion, appURL)
	defer client.Close()

	var releases []musicbrainz.Release

	if *search != "" {
		fmt.Printf("Searching MusicBrainz for: %s\n", *search)
		releases, err = client.Search(*search)
	} else {
		fmt.Println("Looking up on MusicBrainz...")
		releases, err = client.LookupByDiscID(discID)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "MusicBrainz lookup failed: %v\n", err)
		os.Exit(1)
	}

	if len(releases) == 0 {
		fmt.Fprintln(os.Stderr, "No releases found. Try --search \"Artist Album\"")
		os.Exit(1)
	}

	// Sort releases: exact track count matches first, then by year (newest first)
	releases = musicbrainz.SortReleasesByTrackMatch(releases, len(wavFiles))

	// Present options
	var release *musicbrainz.Release
	if len(releases) == 1 {
		fmt.Printf("Found: %s - %s (%d, %d tracks)\n", releases[0].Artist, releases[0].Title, releases[0].Year, releases[0].TrackCount)
		release = &releases[0]
	} else {
		fmt.Printf("\nFound %d releases:\n", len(releases))
		for i, r := range releases {
			fmt.Printf("  %d. %s - %s (%d, %s, %d tracks)\n", i+1, r.Artist, r.Title, r.Year, r.Country, r.TrackCount)
		}
		fmt.Print("\nSelect release (1): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice := 1
		if input != "" {
			if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(releases) {
				choice = n
			}
		}
		release = &releases[choice-1]
	}

	// Get full track info
	fmt.Println("\nFetching track details...")
	fullRelease, err := client.GetReleaseTracks(release.MBID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get track info: %v\n", err)
		os.Exit(1)
	}

	// Fetch cover art (optional)
	fmt.Print("Fetching cover art... ")
	coverArt, coverMIME, err := client.GetCoverArt(release.MBID)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		coverArt = nil // Ensure we continue without cover
	} else if coverArt == nil {
		fmt.Println("not available")
	} else {
		fmt.Printf("OK (%d KB, %s)\n", len(coverArt)/1024, coverMIME)
	}

	// Validate track count
	if len(fullRelease.Tracks) != len(wavFiles) {
		fmt.Fprintf(os.Stderr, "Warning: Track count mismatch (%d WAV files, %d tracks in release)\n",
			len(wavFiles), len(fullRelease.Tracks))
	}

	// Determine destination
	destDir := *dest
	if destDir == "" {
		home, _ := os.UserHomeDir()
		destDir = filepath.Join(home, "Music")
	}

	fmt.Printf("\nDestination: %s\n", destDir)
	fmt.Printf("Quality: V%d\n", *quality)

	if *dryRun {
		fmt.Println("\n[DRY RUN] Would encode:")
	} else {
		fmt.Println("\nEncoding:")
	}

	// Process each track
	opts := encode.EncodeOptions{
		Quality: *quality,
		Verbose: *verbose,
	}

	for i, wavFile := range wavFiles {
		var trackNum int
		var trackTitle, trackArtist string

		if i < len(fullRelease.Tracks) {
			track := fullRelease.Tracks[i]
			trackNum = track.Num
			trackTitle = track.Title
			trackArtist = track.Artist
		} else {
			trackNum = i + 1
			trackTitle = fmt.Sprintf("Track %d", trackNum)
			trackArtist = fullRelease.Artist
		}

		// Generate filename
		var filename string
		if fullRelease.Compilation {
			filename = encode.GenerateCompilationFilename(
				fullRelease.Title,
				0, // disc
				trackNum,
				trackArtist,
				trackTitle,
			)
		} else {
			filename = encode.GenerateFilename(
				fullRelease.Artist,
				fullRelease.Title,
				0, // disc
				trackNum,
				trackTitle,
			)
		}

		mp3Path := filepath.Join(destDir, filename)

		if *dryRun {
			fmt.Printf("  %s -> %s\n", filepath.Base(wavFile), filename)
			continue
		}

		fmt.Printf("  %02d. %s... ", trackNum, trackTitle)

		// Create dest dir if needed
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		// Encode
		tempMP3 := filepath.Join(inputDir, fmt.Sprintf("track%02d.mp3", trackNum))
		if err := encode.EncodeWAV(wavFile, tempMP3, opts); err != nil {
			fmt.Printf("ENCODE ERROR: %v\n", err)
			continue
		}

		// Tag
		tags := encode.BuildTags(encode.TrackMeta{
			Artist:       trackArtist,
			AlbumArtist:  fullRelease.Artist,
			Album:        fullRelease.Title,
			Title:        trackTitle,
			TrackNum:     trackNum,
			TrackTotal:   len(fullRelease.Tracks),
			Year:         fullRelease.Year,
			Compilation:  fullRelease.Compilation,
			CoverArt:     coverArt,
			CoverArtMIME: coverMIME,
		})

		if err := tags.Apply(tempMP3); err != nil {
			fmt.Printf("TAG ERROR: %v\n", err)
			os.Remove(tempMP3)
			continue
		}

		// Move to destination
		if err := os.Rename(tempMP3, mp3Path); err != nil {
			// Try copy if cross-device
			if data, err := os.ReadFile(tempMP3); err == nil {
				if err := os.WriteFile(mp3Path, data, 0644); err == nil {
					os.Remove(tempMP3)
				}
			}
		}

		fmt.Println("OK")
	}

	if !*dryRun {
		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("Done! Encoded %d tracks to %s\n", len(wavFiles), destDir)
	}
}

func findWAVFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var wavFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".wav") {
			wavFiles = append(wavFiles, filepath.Join(dir, entry.Name()))
		}
	}

	// Sort by filename (track01.wav, track02.wav, etc.)
	sort.Strings(wavFiles)
	return wavFiles, nil
}
