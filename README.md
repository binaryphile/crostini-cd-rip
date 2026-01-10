# crostini-cd-rip

Rip audio CDs on ChromeOS/Crostini without kernel driver support.

## The Problem

ChromeOS's Linux container (Crostini) can detect USB CD drives at the SCSI level, but the `sr_mod` kernel driver isn't exposed to the container, so `/dev/sr0` is never created. This makes standard CD ripping tools like `abcde`, `cdparanoia`, and `sound-juicer` unusable.

## The Solution

Two Go binaries that bypass the kernel entirely:

```
cd-rip       USB CD → WAV extraction via SCSI
    │
    ▼
cd-encode    MusicBrainz lookup → lame → ID3 tags → ~/Music
```

### cd-rip

- Detaches the `usb-storage` kernel driver
- Communicates directly with USB device via gousb/libusb
- Sends SCSI commands (INQUIRY, READ TOC, READ CD) over USB Mass Storage Bulk-Only protocol
- Extracts raw audio data and saves as WAV files
- Calculates MusicBrainz disc ID

### cd-encode

- Looks up album metadata on MusicBrainz using disc ID
- Encodes WAV to MP3 using lame (VBR quality)
- Writes ID3v2.4 tags (artist, album, title, track, year)
- Renames files to convention: `Artist-Album-NN-Title.mp3`
- Moves to ~/Music

## Requirements

- ChromeOS with Crostini (Linux container) enabled
- USB CD/DVD drive shared with Linux container
- Go 1.21+, libusb, lame

### Nix (recommended)

```bash
nix-shell  # Uses shell.nix
```

### Manual

```bash
# Debian/Ubuntu
sudo apt install libusb-1.0-0-dev lame

# Build
go build -o cd-rip ./cmd/cd-rip
go build -o cd-encode ./cmd/cd-encode
```

## Usage

### Ripping

```bash
# Basic usage - rip all tracks to /tmp/cd-rip
./cd-rip

# Specify output directory
./cd-rip -o ~/Music/rip

# Rip specific tracks
./cd-rip -t 1,3,5

# Show TOC only (don't rip)
./cd-rip --toc

# Adjust chunk size for speed (default: 75 frames)
./cd-rip --chunk-size 100
```

### Encoding

```bash
# Encode with MusicBrainz lookup
./cd-encode /tmp/cd-rip

# Preview without encoding
./cd-encode --dry-run /tmp/cd-rip

# Custom destination
./cd-encode --dest ~/Music/New /tmp/cd-rip

# Adjust quality (0-9, lower is better)
./cd-encode -q 0 /tmp/cd-rip
```

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                      Normal Linux                           │
│  App → /dev/sr0 → sr_mod → usb-storage → USB device        │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   Crostini (this tool)                      │
│  App → gousb → libusb → /dev/bus/usb/X/Y → USB device      │
│         ↓                                                   │
│    SCSI commands via USB Mass Storage CBW/CSW protocol      │
└─────────────────────────────────────────────────────────────┘
```

## Output Files

### cd-rip output

```
/tmp/cd-rip/
├── track01.wav
├── track02.wav
├── ...
├── discid.txt      # MusicBrainz disc ID
└── toc.json        # CD table of contents
```

### cd-encode output

```
~/Music/
├── Artist-Album-01-Song_Title.mp3
├── Artist-Album-02-Another_Song.mp3
└── ...
```

## Supported Devices

Tested with:
- Hitachi-LG GP65NB60 (USB ID: 0e8d:1887)

Should work with any USB CD/DVD drive that supports standard SCSI MMC commands.

## Performance

Benchmarked chunk sizes (frames per USB transfer):

| Chunk | Speed | Notes |
|-------|-------|-------|
| 10 | 121 f/s | High USB overhead |
| 20 | 287 f/s | Improving |
| 50-150 | ~317 f/s | **Optimal** (drive-limited) |
| 200 | 306 f/s | Buffer issues |

Default is 75 frames. At ~317 frames/sec, a 60-minute CD rips in ~14 minutes.

## Project Structure

```
├── cmd/
│   ├── cd-rip/         # USB ripper CLI
│   └── cd-encode/      # Encoder CLI
├── internal/
│   ├── cdda/           # TOC, disc ID, WAV (pure functions)
│   ├── scsi/           # USB/SCSI protocol
│   ├── encode/         # Naming, tagging, lame
│   └── musicbrainz/    # MusicBrainz API client
├── shell.nix
└── go.mod
```

## Claude Code Integration (Optional)

When MusicBrainz doesn't recognize a disc, you can use Claude to extract metadata from Discogs. Add this to your project's `CLAUDE.md`:

```markdown
## CD Encoding Workflow

When MusicBrainz lookup fails:
1. User provides Discogs page (markdown or HTML download)
2. Extract metadata to JSON per docs/metadata-format.md
3. Save to /tmp/metadata.json
4. Run: cd-encode --metadata /tmp/metadata.json /tmp/cd-rip

See docs/metadata-format.md for JSON schema and extraction instructions.
```

See [docs/use-cases.md](docs/use-cases.md) for full workflow documentation.

## License

MIT
