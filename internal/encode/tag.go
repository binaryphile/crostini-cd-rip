package encode

import (
	"fmt"
	"strconv"

	"github.com/bogem/id3v2/v2"
)

// TrackMeta contains metadata for a track to be tagged
type TrackMeta struct {
	Artist      string
	AlbumArtist string // For compilations - empty means same as Artist
	Album       string
	Title       string
	TrackNum    int
	TrackTotal  int
	DiscNum     int // 0 = single disc
	DiscTotal   int // 0 = single disc
	Year        int
	Genre       string
	Compilation bool
}

// TagSet contains the ID3 tags to be written
type TagSet struct {
	Artist      string
	AlbumArtist string
	Album       string
	Title       string
	TrackNum    int
	TrackTotal  int
	DiscNum     int
	DiscTotal   int
	Year        int
	Genre       string
	Compilation bool
}

// BuildTags creates a TagSet from track metadata.
// This is a pure function: TrackMeta → TagSet
// No I/O is performed - use Apply() to write tags to a file.
func BuildTags(meta TrackMeta) TagSet {
	return TagSet{
		Artist:      meta.Artist,
		AlbumArtist: meta.AlbumArtist,
		Album:       meta.Album,
		Title:       meta.Title,
		TrackNum:    meta.TrackNum,
		TrackTotal:  meta.TrackTotal,
		DiscNum:     meta.DiscNum,
		DiscTotal:   meta.DiscTotal,
		Year:        meta.Year,
		Genre:       meta.Genre,
		Compilation: meta.Compilation,
	}
}

// Apply writes the tags to an MP3 file.
// This is boundary code - performs file I/O.
func (t TagSet) Apply(filepath string) error {
	tag, err := id3v2.Open(filepath, id3v2.Options{Parse: false})
	if err != nil {
		return fmt.Errorf("open mp3: %w", err)
	}
	defer tag.Close()

	// Set ID3v2.4
	tag.SetDefaultEncoding(id3v2.EncodingUTF8)
	tag.SetVersion(4)

	// Basic tags
	tag.SetArtist(t.Artist)
	tag.SetAlbum(t.Album)
	tag.SetTitle(t.Title)
	tag.SetGenre(t.Genre)

	// Year
	if t.Year > 0 {
		tag.SetYear(strconv.Itoa(t.Year))
	}

	// Track number (format: N/Total)
	if t.TrackTotal > 0 {
		tag.AddTextFrame("TRCK", id3v2.EncodingUTF8,
			fmt.Sprintf("%d/%d", t.TrackNum, t.TrackTotal))
	} else if t.TrackNum > 0 {
		tag.AddTextFrame("TRCK", id3v2.EncodingUTF8,
			strconv.Itoa(t.TrackNum))
	}

	// Disc number (format: N/Total)
	if t.DiscTotal > 0 {
		tag.AddTextFrame("TPOS", id3v2.EncodingUTF8,
			fmt.Sprintf("%d/%d", t.DiscNum, t.DiscTotal))
	} else if t.DiscNum > 0 {
		tag.AddTextFrame("TPOS", id3v2.EncodingUTF8,
			strconv.Itoa(t.DiscNum))
	}

	// Album artist (TPE2)
	if t.AlbumArtist != "" {
		tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, t.AlbumArtist)
	}

	// Compilation flag (TCMP)
	if t.Compilation {
		tag.AddTextFrame("TCMP", id3v2.EncodingUTF8, "1")
	}

	if err := tag.Save(); err != nil {
		return fmt.Errorf("save tags: %w", err)
	}

	return nil
}
