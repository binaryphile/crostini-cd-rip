# crostini-cd-rip Use Cases

## System-in-Use Story

> "Ted, wanting to digitize his CD collection, inserts a disc into his USB drive. He runs `cd-rip` which extracts the audio and calculates the MusicBrainz disc ID. Then `cd-encode` looks up the album metadata and produces properly tagged MP3s in ~/Music. When MusicBrainz doesn't recognize an obscure compilation, Ted downloads the Discogs page and asks Claude to extract the metadata into JSON, then runs `cd-encode --metadata metadata.json` to encode with that metadata."

## System Scope

**Boundary**: USB CD drive → organized MP3 files in ~/Music

**In Scope**: CD audio extraction, metadata lookup, MP3 encoding, ID3 tagging, filename generation

**Out of Scope**: CD burning, video DVDs, streaming, library management

## Actor-Goal List

| Actor | Goal | Level | Status |
|-------|------|-------|--------|
| User | Rip CD to WAV files | Blue | Implemented |
| User | View CD table of contents | Blue | Implemented |
| User | Encode WAVs with MusicBrainz metadata | Blue | Implemented |
| User | Encode WAVs with manual metadata JSON | Blue | Implemented |
| User | Encode compilation album | Blue | Implemented |
| User | Handle multi-disc album | Blue | Implemented |
| User | Fire-and-forget rip+encode | Blue | Implemented |

## Use Cases

### UC1: Rip CD to WAV (Implemented)

User inserts CD and runs `cd-rip`. System detects USB drive, reads TOC, extracts audio track-by-track, saves WAV files to /tmp/cd-rip/, generates discid.txt and toc.json.

**Extensions:**
- No USB CD drive found → error with troubleshooting hint
- Read error on track → retry with smaller chunk size, or skip track

### UC2: View TOC Only (Implemented)

User runs `cd-rip --toc`. System displays track listing with durations without ripping.

### UC3: Encode with MusicBrainz (Implemented)

User runs `cd-encode /tmp/cd-rip`. System reads discid.txt, queries MusicBrainz, fetches cover art, encodes each WAV to MP3 with full ID3 tags, moves to ~/Music with standardized filename.

**Extensions:**
- No MusicBrainz match → suggest `--search "Artist Album"` or `--metadata`
- Multiple releases match → present menu for user selection
- No cover art available → continue without embedding art

### UC4: Encode with Manual Metadata (Implemented)

User provides metadata as JSON (extracted by LLM from Discogs/other source), runs `cd-encode --metadata metadata.json /tmp/cd-rip`. System reads structured metadata, encodes with provided artist/album/tracks.

**CLI:** `cd-encode --metadata <metadata.json> [--strict] <input-dir>`

**Flags:**
- `--metadata` - JSON file with album/track metadata (bypasses MusicBrainz)
- `--strict` - Exit on validation errors (default: warn and proceed)

**Why LLM extraction vs coded parser:**
- Format-agnostic (handles Discogs markdown, HTML, other sources)
- No fragile parsing code to maintain
- Adapts to source format changes automatically

**Extensions:**
- Track count mismatch → warning (or fatal with `--strict`)
- Missing required field → warning (or fatal with `--strict`)
- Cover art file not found → warning, proceed without cover

See [metadata-format.md](metadata-format.md) for JSON schema and LLM extraction instructions.

### UC5: Encode Compilation Album (Implemented)

User encodes Various Artists album. System detects compilation (album artist = "Various Artists"), uses per-track artist in filename: `Album-NN-TrackArtist-Title.mp3`.

### UC6: Handle Multi-Disc Album (Implemented)

User rips disc 2 of a set. System includes disc number in filename: `Album-CD2-NN-Title.mp3` to avoid collisions with disc 1.

### UC7: Pipeline Rip+Encode (Implemented)

User runs `cd-pipeline`. Shell script runs cd-rip then cd-encode in sequence, fire-and-forget.
