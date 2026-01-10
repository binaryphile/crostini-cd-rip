# crostini-cd-rip Design Document

## Part 1: Current Architecture

### 1.1 System Overview

Two-binary architecture for ripping CDs on ChromeOS/Crostini:

```
cd-rip       USB CD → WAV extraction via SCSI
    │
    ▼
cd-encode    MusicBrainz lookup → lame → ID3 tags → ~/Music
```

**Boundary**: USB CD drive → organized MP3 files in ~/Music

**Codebase**: ~3K LOC Go

### 1.2 Package Structure

```
cmd/
├── cd-rip/main.go        CLI entry point for USB ripping
└── cd-encode/main.go     CLI entry point for encoding

internal/
├── scsi/                 USB/SCSI protocol layer
│   ├── device.go         USB device management (285 LOC)
│   ├── protocol.go       Mass Storage Bulk-Only protocol (83 LOC)
│   └── commands.go       SCSI command builders (97 LOC)
├── cdda/                 CD audio (pure functions)
│   ├── toc.go            Table of contents parsing (104 LOC)
│   ├── discid.go         MusicBrainz disc ID calculation (62 LOC)
│   └── wav.go            WAV file generation (60 LOC)
├── encode/               MP3 encoding & ID3 tagging
│   ├── lame.go           LAME encoder wrapper (80 LOC)
│   ├── tag.go            ID3v2.4 tag builder (138 LOC)
│   └── naming.go         Filename generation & sanitization (144 LOC)
└── musicbrainz/          Metadata API client
    ├── lookup.go         MusicBrainz API queries (219 LOC)
    └── coverart.go       Cover Art Archive fetcher (56 LOC)
```

### 1.3 Data Flow

**cd-rip flow:**
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  USB CD     │    │   scsi/     │    │   cdda/     │    │  Output     │
│  Drive      │───▶│  device.go  │───▶│  toc.go     │───▶│  Files      │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                         │                  │                  │
                   OpenDevice()        ParseTOC()         /tmp/cd-rip/
                   ReadTOCRaw()        CalculateDiscID()  ├── track01.wav
                   ReadCDFrames()      WriteWAV()         ├── track02.wav
                                                          ├── discid.txt
                                                          └── toc.json
```

**cd-encode flow:**
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Metadata   │    │   encode/   │    │   lame      │    │  Output     │
│  Source     │───▶│  naming.go  │───▶│  (external) │───▶│  ~/Music/   │
└─────────────┘    │  tag.go     │    └─────────────┘    └─────────────┘
      │            └─────────────┘           │
      │                  │                   │
 ┌────┴────┐       GenerateFilename()   WAV → MP3
 │         │       BuildTags()          VBR quality
 ▼         ▼       Apply() [ID3v2.4]
MusicBrainz  JSON
(--metadata)
```

### 1.4 Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **SCSI bypass** | Crostini lacks sr_mod kernel driver; communicate directly via gousb/libusb |
| **Pure functions** | cdda package has no side effects (testable, composable) |
| **Functional core, imperative shell** | encode package separates logic (BuildTags) from I/O (Apply) |
| **External lame** | Shell out to lame binary (simpler than CGO bindings) |
| **Two binaries** | Separation of concerns: ripping vs encoding are independent operations |

### 1.5 Data Structures

**musicbrainz.Release** (`internal/musicbrainz/lookup.go:14-24`)
```go
type Release struct {
    MBID        string   // MusicBrainz ID
    Title       string   // Album title
    Artist      string   // Album artist ("Various Artists" for compilations)
    Year        int      // Release year
    Country     string   // Country code
    TrackCount  int      // Total tracks
    DiscCount   int      // Number of discs
    Tracks      []Track  // Track list
    Compilation bool     // True if artist == "Various Artists"
}
```

**musicbrainz.Track** (`internal/musicbrainz/lookup.go:27-31`)
```go
type Track struct {
    Num    int    // Track position (1-based)
    Title  string // Track title
    Artist string // Track artist (for compilations)
}
```

**encode.TrackMeta** (`internal/encode/tag.go:11-25`)
```go
type TrackMeta struct {
    Artist       string
    AlbumArtist  string
    Album        string
    Title        string
    TrackNum     int
    TrackTotal   int
    DiscNum      int
    DiscTotal    int
    Year         int
    Genre        string
    Compilation  bool
    CoverArt     []byte
    CoverArtMIME string
}
```

---

## Part 2: UC4 Implementation Design (`--metadata`)

**Status**: Planned

### 2.1 Feature Summary

When MusicBrainz lookup fails (no match or too many ambiguous matches), user provides metadata as JSON file extracted by LLM from Discogs or other source.

**CLI**: `cd-encode --metadata /tmp/metadata.json /tmp/cd-rip`

See [metadata-format.md](metadata-format.md) for JSON schema and extraction instructions.

### 2.2 JSON Schema

```json
{
  "artist": "string (required)",
  "album": "string (required)",
  "year": "string (optional)",
  "genre": "string (optional)",
  "disc": "int (optional)",
  "totalDiscs": "int (optional)",
  "totalTracks": "int (optional)",
  "coverArt": "string path (optional)",
  "tracks": [
    {"num": 1, "title": "string", "artist": "string (optional)"}
  ]
}
```

### 2.3 Implementation Approach

**New `internal/metadata/` package** with:
- `Album` struct matching JSON schema
- `ParseJSON(path)` - read and unmarshal JSON file
- `ToRelease()` - convert to `musicbrainz.Release` for existing pipeline
- `Validate(wavCount)` - check required fields and track count
- `LoadCoverArt()` - read cover image file

### 2.4 Field Mapping

```
JSON Field        →  Target
─────────────────────────────────────────────────
artist            →  Release.Artist
album             →  Release.Title
year (string)     →  Release.Year (int, parsed)
disc              →  filename (CDN prefix)
totalDiscs        →  Release.DiscCount
totalTracks       →  Release.TrackCount
tracks[].num      →  Release.Tracks[].Num
tracks[].title    →  Release.Tracks[].Title
tracks[].artist   →  Release.Tracks[].Artist
coverArt          →  LoadCoverArt() → TrackMeta.CoverArt
genre             →  TrackMeta.Genre (direct, not via Release)
─────────────────────────────────────────────────
Derived:
Compilation       →  true if Artist == "Various Artists"
```

### 2.5 Error Handling

| Error Condition | Response |
|-----------------|----------|
| JSON file not found | Fatal: exit with error message |
| JSON parse error | Fatal: show error details |
| Missing required field | Fatal: list missing fields |
| Track count mismatch | Warning + prompt to continue |
| Track numbers not sequential | Warning (proceed anyway) |
| Compilation without track artists | Warning per track |
| Cover art file not found | Warning: proceed without cover |
| Cover art not JPEG/PNG | Warning: skip cover art |

### 2.6 Code Sketches

**Album struct** (`internal/metadata/metadata.go`):
```go
type Album struct {
    Artist     string  `json:"artist"`
    AlbumTitle string  `json:"album"`
    Year       string  `json:"year"`
    Genre      string  `json:"genre"`
    Disc       int     `json:"disc"`
    TotalDiscs int     `json:"totalDiscs"`
    TotalTracks int    `json:"totalTracks"`
    CoverArt   string  `json:"coverArt"`
    Tracks     []Track `json:"tracks"`
}

type Track struct {
    Num    int    `json:"num"`
    Title  string `json:"title"`
    Artist string `json:"artist"`
}
```

**ParseJSON()** - Read and unmarshal JSON file:
```go
func ParseJSON(path string) (*Album, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read metadata: %w", err)
    }
    var album Album
    if err := json.Unmarshal(data, &album); err != nil {
        return nil, fmt.Errorf("parse metadata: %w", err)
    }
    return &album, nil
}
```

**ToRelease()** - Convert JSON Album to musicbrainz.Release:
```go
func (a *Album) ToRelease() *musicbrainz.Release {
    year, _ := strconv.Atoi(a.Year)  // ignore error, default 0

    tracks := make([]musicbrainz.Track, len(a.Tracks))
    for i, t := range a.Tracks {
        tracks[i] = musicbrainz.Track{
            Num:    t.Num,
            Title:  t.Title,
            Artist: t.Artist,
        }
    }

    return &musicbrainz.Release{
        Title:       a.AlbumTitle,
        Artist:      a.Artist,
        Year:        year,
        TrackCount:  len(a.Tracks),
        DiscCount:   a.TotalDiscs,
        Tracks:      tracks,
        Compilation: a.Artist == "Various Artists",
    }
}
```

**Validate()** - Check required fields and track count:
```go
func (a *Album) Validate(wavCount int) []error {
    var errs []error
    if a.Artist == "" {
        errs = append(errs, errors.New("missing required field: artist"))
    }
    if a.AlbumTitle == "" {
        errs = append(errs, errors.New("missing required field: album"))
    }
    if len(a.Tracks) == 0 {
        errs = append(errs, errors.New("missing required field: tracks"))
    }
    if len(a.Tracks) != wavCount {
        errs = append(errs, fmt.Errorf("track count mismatch: JSON has %d, found %d WAV files",
            len(a.Tracks), wavCount))
    }
    if a.Artist == "Various Artists" {
        for i, t := range a.Tracks {
            if t.Artist == "" {
                errs = append(errs, fmt.Errorf("track %d missing artist (required for compilations)", i+1))
            }
        }
    }
    return errs
}
```

**LoadCoverArt()** - Read cover image file:
```go
func (a *Album) LoadCoverArt() ([]byte, string, error) {
    if a.CoverArt == "" {
        return nil, "", nil  // No cover art specified
    }
    data, err := os.ReadFile(a.CoverArt)
    if err != nil {
        return nil, "", fmt.Errorf("cover art: %w", err)
    }
    mime := detectMIME(a.CoverArt)
    return data, mime, nil
}

func detectMIME(path string) string {
    ext := strings.ToLower(filepath.Ext(path))
    switch ext {
    case ".jpg", ".jpeg":
        return "image/jpeg"
    case ".png":
        return "image/png"
    default:
        return "image/jpeg"  // fallback
    }
}
```

### 2.7 Files to Create/Modify

| File | Change |
|------|--------|
| `cmd/cd-encode/main.go` | Add `--metadata` flag, branch on flag |
| `internal/metadata/metadata.go` | Album/Track structs, ParseJSON, ToRelease, Validate |
| `internal/metadata/metadata_test.go` | Unit tests |

### 2.7 Testing Strategy

**Unit Tests**:
- `TestParseJSON_StandardAlbum` - Parse valid JSON
- `TestParseJSON_Compilation` - Various Artists with track artists
- `TestParseJSON_MultiDisc` - disc/totalDiscs fields
- `TestParseJSON_MissingRequired` - Error cases
- `TestToRelease_FieldMapping` - Verify mapping
- `TestValidate_TrackCountMismatch` - Warning behavior
- `TestLoadCoverArt_*` - JPEG, PNG, not found, empty

**Test Data** (`internal/metadata/testdata/`):
- `standard_album.json`, `compilation.json`, `multi_disc.json`
- `missing_artist.json`, `malformed.json`
- `cover.jpg`, `cover.png`

---

## Maintenance Notes

**Living document**: This design doc should be updated when UC4 is implemented:
- Change Part 2 status from "Planned" to "Implemented"
- Update code sketches with actual line numbers
- Add any implementation deviations or learnings
