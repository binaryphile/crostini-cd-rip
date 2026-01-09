# crostini-cd-rip

Rip audio CDs on ChromeOS/Crostini without kernel driver support.

## The Problem

ChromeOS's Linux container (Crostini) can detect USB CD drives at the SCSI level, but the `sr_mod` kernel driver isn't exposed to the container, so `/dev/sr0` is never created. This makes standard CD ripping tools like `abcde`, `cdparanoia`, and `sound-juicer` unusable.

## The Solution

This tool bypasses the kernel entirely by:
1. Detaching the `usb-storage` kernel driver
2. Communicating directly with the USB device via PyUSB/libusb
3. Sending SCSI commands (INQUIRY, READ TOC, READ CD) over USB Mass Storage Bulk-Only protocol
4. Extracting raw audio data and saving as WAV files

## Requirements

- ChromeOS with Crostini (Linux container) enabled
- USB CD/DVD drive shared with Linux container
- Python 3 with PyUSB

### Nix (recommended)

```bash
nix-shell -p python3 python3Packages.pyusb
```

### Pip

```bash
pip install pyusb
```

## Usage

```bash
# Basic usage - rip all tracks to /tmp/cd-rip
./crostini-cd-rip.py

# Specify output directory
./crostini-cd-rip.py -o ~/Music/rip

# Rip specific tracks
./crostini-cd-rip.py -t 1,3,5

# Show TOC only (don't rip)
./crostini-cd-rip.py --toc

# Adjust chunk size for speed (default: 27 frames)
./crostini-cd-rip.py --chunk-size 50
```

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                      Normal Linux                           │
│  App → /dev/sr0 → sr_mod → usb-storage → USB device        │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   Crostini (this tool)                      │
│  App → PyUSB → libusb → /dev/bus/usb/X/Y → USB device      │
│         ↓                                                   │
│    SCSI commands via USB Mass Storage CBW/CSW protocol      │
└─────────────────────────────────────────────────────────────┘
```

## Supported Devices

Tested with:
- Hitachi-LG GP65NB60 (USB ID: 0e8d:1887)

Should work with any USB CD/DVD drive that supports standard SCSI MMC commands.

## Output

By default, tracks are saved as WAV files:
- `track01.wav`
- `track02.wav`
- etc.

Convert to MP3 with:
```bash
for f in *.wav; do lame -V 2 "$f" "${f%.wav}.mp3"; done
```

## Limitations

- No MusicBrainz/CDDB lookup (yet)
- No automatic MP3 encoding (use lame separately)
- Slower than native cdparanoia (USB overhead + small chunk sizes)
- No error correction/paranoia mode (yet)

## License

MIT
