package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/binaryphile/crostini-cd-rip/internal/cdda"
	"github.com/binaryphile/crostini-cd-rip/internal/scsi"
	"github.com/google/gousb"
)

func main() {
	// Parse flags
	output := flag.String("o", "/tmp/cd-rip", "Output directory")
	flag.StringVar(output, "output", "/tmp/cd-rip", "Output directory")

	tracks := flag.String("t", "", "Tracks to rip (comma-separated, e.g., 1,3,5)")
	flag.StringVar(tracks, "tracks", "", "Tracks to rip (comma-separated, e.g., 1,3,5)")

	tocOnly := flag.Bool("toc", false, "Show TOC only, don't rip")

	chunkSize := flag.Int("chunk-size", 75, "Frames per USB transfer")

	verbose := flag.Bool("v", false, "Verbose output")
	flag.BoolVar(verbose, "verbose", false, "Verbose output")

	vendorID := flag.String("vendor-id", "", "USB vendor ID (hex, e.g., 0x0e8d)")
	productID := flag.String("product-id", "", "USB product ID (hex, e.g., 0x1887)")

	flag.Parse()

	fmt.Println("cd-rip - USB CD Ripper for ChromeOS/Crostini")
	fmt.Println(strings.Repeat("=", 50))

	// Parse vendor/product IDs
	var vid, pid gousb.ID
	if *vendorID != "" {
		v, err := strconv.ParseUint(strings.TrimPrefix(*vendorID, "0x"), 16, 16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid vendor ID: %s\n", *vendorID)
			os.Exit(1)
		}
		vid = gousb.ID(v)
	}
	if *productID != "" {
		p, err := strconv.ParseUint(strings.TrimPrefix(*productID, "0x"), 16, 16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid product ID: %s\n", *productID)
			os.Exit(1)
		}
		pid = gousb.ID(p)
	}

	// Open device
	dev, err := scsi.OpenDevice(vid, pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Is the USB CD drive shared with Linux?")
		os.Exit(1)
	}
	defer dev.Close()

	// Get device info
	info, err := dev.Inquiry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "INQUIRY failed: %v\n", err)
		os.Exit(1)
	}

	deviceType := "Unknown"
	if info.DeviceType == 5 {
		deviceType = "CD-ROM"
	}
	fmt.Printf("\nDevice: %s %s (rev %s)\n", info.Vendor, info.Product, info.Revision)
	fmt.Printf("Type: %s\n", deviceType)

	// Check if disc is ready
	if !dev.TestUnitReady() {
		fmt.Fprintln(os.Stderr, "\nNo disc in drive or drive not ready")
		os.Exit(1)
	}

	// Read TOC
	tocRaw, err := dev.ReadTOCRaw()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read TOC: %v\n", err)
		os.Exit(1)
	}

	toc, err := cdda.ParseTOC(tocRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse TOC: %v\n", err)
		os.Exit(1)
	}

	// Print TOC
	printTOC(toc, *verbose)

	if *tocOnly {
		return
	}

	// Parse track selection
	tracksToRip := parseTrackList(*tracks)

	// Create output directory
	if err := os.MkdirAll(*output, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nRipping to: %s\n", *output)
	fmt.Printf("Chunk size: %d frames\n\n", *chunkSize)

	// Rip tracks
	wavFiles := ripTracks(dev, toc, *output, tracksToRip, *chunkSize, *verbose)

	// Calculate disc ID and save metadata
	discID := cdda.CalculateDiscID(toc)

	// Save disc ID
	discIDPath := fmt.Sprintf("%s/discid.txt", *output)
	if err := os.WriteFile(discIDPath, []byte(discID+"\n"), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save disc ID: %v\n", err)
	} else {
		fmt.Printf("\nDisc ID: %s\n", discID)
		fmt.Printf("Saved to: %s\n", discIDPath)
	}

	// Save TOC as JSON
	tocPath := fmt.Sprintf("%s/toc.json", *output)
	tocJSON := tocToJSON(toc)
	if err := os.WriteFile(tocPath, tocJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save TOC: %v\n", err)
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("Done! Ripped %d tracks to %s\n", len(wavFiles), *output)
	fmt.Println("\nNext step: cd-encode", *output)
}

func printTOC(toc cdda.TOC, verbose bool) {
	fmt.Printf("\nTable of Contents:\n")
	fmt.Printf("%6s %8s %10s %10s %10s\n", "Track", "Type", "LBA", "Length", "Duration")
	fmt.Println(strings.Repeat("-", 50))

	for i, track := range toc.Tracks {
		trackType := "audio"
		if track.Type == cdda.TrackTypeData {
			trackType = "data"
		}

		var length int
		if i+1 < len(toc.Tracks) {
			length = toc.Tracks[i+1].LBA - track.LBA
		} else {
			length = toc.LeadoutLBA - track.LBA
		}

		duration := float64(length) / float64(scsi.FramesPerSecond)
		fmt.Printf("%6d %8s %10d %10d %9.1fs\n",
			track.Num, trackType, track.LBA, length, duration)
	}

	fmt.Printf("%6s %8s %10d\n", "Lead-out", "-", toc.LeadoutLBA)
}

func parseTrackList(tracks string) map[int]bool {
	if tracks == "" {
		return nil
	}

	result := make(map[int]bool)
	for _, t := range strings.Split(tracks, ",") {
		t = strings.TrimSpace(t)
		if n, err := strconv.Atoi(t); err == nil {
			result[n] = true
		}
	}
	return result
}

func ripTracks(dev *scsi.Device, toc cdda.TOC, outputDir string, tracksToRip map[int]bool, chunkSize int, verbose bool) []string {
	var wavFiles []string

	for i, track := range toc.Tracks {
		// Skip if not in selection
		if tracksToRip != nil && !tracksToRip[track.Num] {
			continue
		}

		// Skip data tracks
		if track.Type == cdda.TrackTypeData {
			fmt.Printf("Track %d: Skipping (data track)\n", track.Num)
			continue
		}

		// Calculate end LBA
		var endLBA int
		if i+1 < len(toc.Tracks) {
			endLBA = toc.Tracks[i+1].LBA
		} else {
			endLBA = toc.LeadoutLBA
		}

		wavFile := ripTrack(dev, track.Num, track.LBA, endLBA, outputDir, chunkSize, verbose)
		if wavFile != "" {
			wavFiles = append(wavFiles, wavFile)
		}
	}

	return wavFiles
}

func ripTrack(dev *scsi.Device, trackNum, startLBA, endLBA int, outputDir string, chunkSize int, verbose bool) string {
	totalFrames := endLBA - startLBA
	durationSec := float64(totalFrames) / float64(scsi.FramesPerSecond)

	fmt.Printf("Track %d: %d frames (%.1fs / %.1fm)\n",
		trackNum, totalFrames, durationSec, durationSec/60)

	filename := fmt.Sprintf("%s/track%02d.wav", outputDir, trackNum)
	audioData := make([]byte, 0, totalFrames*scsi.FrameSize)

	currentLBA := startLBA
	startTime := time.Now()
	errors := 0

	for currentLBA < endLBA {
		framesToRead := chunkSize
		if currentLBA+framesToRead > endLBA {
			framesToRead = endLBA - currentLBA
		}

		data, err := dev.ReadCDFrames(currentLBA, framesToRead)
		if err != nil {
			errors++
			if errors > 10 {
				fmt.Printf("\n  Too many errors at LBA %d, aborting track\n", currentLBA)
				break
			}
			fmt.Printf("\n  Error at LBA %d, retrying...\n", currentLBA)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		audioData = append(audioData, data...)
		currentLBA += framesToRead
		errors = 0

		// Progress
		done := currentLBA - startLBA
		progress := done * 100 / totalFrames
		elapsed := time.Since(startTime).Seconds()
		speed := float64(done) / elapsed
		eta := float64(totalFrames-done) / speed

		fmt.Printf("\r  %3d%% | %d/%d frames | %.0f frames/s | ETA: %.0fs   ",
			progress, done, totalFrames, speed, eta)
	}

	fmt.Println()

	// Write WAV file
	wav := cdda.WriteWAV(audioData)
	if err := os.WriteFile(filename, wav, 0644); err != nil {
		fmt.Printf("  Error writing %s: %v\n", filename, err)
		return ""
	}

	fileSize := len(wav)
	fmt.Printf("  Saved: %s (%.1f MB)\n", filename, float64(fileSize)/1024/1024)

	return filename
}

func tocToJSON(toc cdda.TOC) []byte {
	type jsonTrack struct {
		Num  int    `json:"num"`
		LBA  int    `json:"lba"`
		Type string `json:"type"`
	}

	type jsonTOC struct {
		FirstTrack int         `json:"first_track"`
		LastTrack  int         `json:"last_track"`
		Tracks     []jsonTrack `json:"tracks"`
		LeadoutLBA int         `json:"leadout_lba"`
	}

	tracks := make([]jsonTrack, len(toc.Tracks))
	for i, t := range toc.Tracks {
		trackType := "audio"
		if t.Type == cdda.TrackTypeData {
			trackType = "data"
		}
		tracks[i] = jsonTrack{
			Num:  t.Num,
			LBA:  t.LBA,
			Type: trackType,
		}
	}

	j := jsonTOC{
		FirstTrack: toc.FirstTrack,
		LastTrack:  toc.LastTrack,
		Tracks:     tracks,
		LeadoutLBA: toc.LeadoutLBA,
	}

	data, _ := json.MarshalIndent(j, "", "  ")
	return data
}
