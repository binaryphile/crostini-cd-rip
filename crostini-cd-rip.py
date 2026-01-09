#!/usr/bin/env python3
"""
crostini-cd-rip - Rip audio CDs on ChromeOS/Crostini without kernel driver support.

Bypasses the missing /dev/sr0 by communicating directly with USB CD drives
via PyUSB, sending SCSI commands over USB Mass Storage Bulk-Only protocol.
"""

import argparse
import os
import struct
import sys
import time
import wave

try:
    import usb.core
    import usb.util
except ImportError:
    print("Error: PyUSB not installed. Install with: pip install pyusb")
    print("Or use: nix-shell -p python3 python3Packages.pyusb")
    sys.exit(1)

# USB Mass Storage constants
CBW_SIGNATURE = 0x43425355
CSW_SIGNATURE = 0x53425355

# CD audio constants
CD_FRAMESIZE_RAW = 2352  # Raw audio frame size (2352 bytes = 1/75th second)
FRAMES_PER_SECOND = 75


class USBCDRipper:
    """USB CD Ripper using direct SCSI commands via PyUSB."""

    # Known USB CD drive IDs (vendor_id, product_id, name)
    KNOWN_DEVICES = [
        (0x0e8d, 0x1887, "Hitachi-LG/MediaTek Slim Portable DVD Writer"),
        (0x152d, 0x2339, "JMicron USB CD/DVD"),
        (0x13fd, 0x0840, "Initio USB CD/DVD"),
        (0x1c6b, 0xa223, "Philips USB CD/DVD"),
    ]

    def __init__(self, vendor_id=None, product_id=None, chunk_size=27):
        """
        Initialize USB CD ripper.

        Args:
            vendor_id: USB vendor ID (auto-detect if None)
            product_id: USB product ID (auto-detect if None)
            chunk_size: Number of CD frames to read per USB transfer
        """
        self.chunk_size = chunk_size
        self.dev = None
        self.ep_out = None
        self.ep_in = None
        self.tag = 1

        # Find device
        if vendor_id and product_id:
            self.dev = usb.core.find(idVendor=vendor_id, idProduct=product_id)
        else:
            # Try known devices
            for vid, pid, name in self.KNOWN_DEVICES:
                self.dev = usb.core.find(idVendor=vid, idProduct=pid)
                if self.dev:
                    print(f"Found: {name}")
                    break

            # Try any Mass Storage CD-ROM
            if not self.dev:
                for dev in usb.core.find(find_all=True, bDeviceClass=0):
                    try:
                        cfg = dev.get_active_configuration()
                        for intf in cfg:
                            if intf.bInterfaceClass == 8:  # Mass Storage
                                # Check if it's a CD-ROM by sending INQUIRY
                                self.dev = dev
                                break
                    except:
                        pass
                    if self.dev:
                        break

        if not self.dev:
            raise RuntimeError("No USB CD drive found. Is it shared with Linux?")

        self._setup_device()

    def _setup_device(self):
        """Set up USB device for communication."""
        # Detach kernel driver if active
        try:
            if self.dev.is_kernel_driver_active(0):
                print("Detaching kernel driver...")
                self.dev.detach_kernel_driver(0)
        except usb.core.USBError:
            pass  # May not be supported

        # Set configuration
        try:
            self.dev.set_configuration()
        except usb.core.USBError:
            pass  # May already be configured

        # Get endpoints
        cfg = self.dev.get_active_configuration()
        intf = cfg[(0, 0)]

        self.ep_out = usb.util.find_descriptor(
            intf,
            custom_match=lambda e: usb.util.endpoint_direction(e.bEndpointAddress)
            == usb.util.ENDPOINT_OUT,
        )
        self.ep_in = usb.util.find_descriptor(
            intf,
            custom_match=lambda e: usb.util.endpoint_direction(e.bEndpointAddress)
            == usb.util.ENDPOINT_IN,
        )

        if not self.ep_out or not self.ep_in:
            raise RuntimeError("Could not find USB endpoints")

        print(f"Endpoints: OUT=0x{self.ep_out.bEndpointAddress:02x}, "
              f"IN=0x{self.ep_in.bEndpointAddress:02x}")

    def _send_command(self, cdb, data_len=0, direction="in", timeout=30000):
        """
        Send SCSI command via USB Mass Storage Bulk-Only protocol.

        Args:
            cdb: Command Descriptor Block (SCSI command bytes)
            data_len: Expected data length
            direction: 'in' for reading, 'out' for writing
            timeout: USB timeout in milliseconds

        Returns:
            tuple: (data, status) where status 0 = success
        """
        flags = 0x80 if direction == "in" else 0x00
        cbw = struct.pack(
            "<IIIBBB", CBW_SIGNATURE, self.tag, data_len, flags, 0, len(cdb)
        )
        cbw += cdb + bytes(16 - len(cdb))
        self.tag += 1

        # Send CBW
        try:
            self.dev.write(self.ep_out, cbw, timeout=timeout)
        except usb.core.USBError as e:
            print(f"CBW write error: {e}")
            return None, -1

        # Read data if expected
        data = None
        if data_len > 0 and direction == "in":
            try:
                data = bytes(self.dev.read(self.ep_in, data_len, timeout=timeout))
            except usb.core.USBError as e:
                print(f"Data read error: {e}")
                # Try to recover by reading CSW anyway

        # Read CSW (Command Status Wrapper)
        try:
            csw = bytes(self.dev.read(self.ep_in, 13, timeout=timeout))
            sig, csw_tag, residue, status = struct.unpack("<IIIB", csw)
            if sig != CSW_SIGNATURE:
                print(f"Invalid CSW signature: 0x{sig:08x}")
                return data, -1
        except usb.core.USBError as e:
            print(f"CSW read error: {e}")
            return data, -1

        return data, status

    def inquiry(self):
        """Send SCSI INQUIRY command to identify the device."""
        cdb = bytes([0x12, 0, 0, 0, 36, 0])
        data, status = self._send_command(cdb, 36)

        if status == 0 and data:
            device_type = data[0] & 0x1F
            vendor = bytes(data[8:16]).decode("ascii", errors="ignore").strip()
            product = bytes(data[16:32]).decode("ascii", errors="ignore").strip()
            revision = bytes(data[32:36]).decode("ascii", errors="ignore").strip()
            return {
                "type": device_type,
                "vendor": vendor,
                "product": product,
                "revision": revision,
            }
        return None

    def test_unit_ready(self):
        """Check if drive is ready (disc loaded)."""
        cdb = bytes([0x00, 0, 0, 0, 0, 0])
        _, status = self._send_command(cdb, 0)
        return status == 0

    def read_toc(self):
        """
        Read Table of Contents from audio CD.

        Returns:
            tuple: (first_track, last_track, tracks) where tracks is list of
                   {'track': N, 'lba': start_address, 'type': 'audio'|'data'}
        """
        # READ TOC command - format 0 (standard TOC), starting track 0
        cdb = bytes([0x43, 0x02, 0, 0, 0, 0, 0, 0x03, 0xFC, 0])
        data, status = self._send_command(cdb, 1020)

        if status != 0 or not data:
            raise RuntimeError("Failed to read TOC - is there a disc in the drive?")

        toc_len = (data[0] << 8) | data[1]
        first_track = data[2]
        last_track = data[3]

        tracks = []
        offset = 4
        while offset + 8 <= len(data) and offset < toc_len + 2:
            control_adr = data[offset + 1]
            track_num = data[offset + 2]

            # LBA is big-endian in TOC response
            lba = struct.unpack(">I", data[offset + 4 : offset + 8])[0]

            # Control field: bit 2 = data track
            is_data = bool(control_adr & 0x04)

            tracks.append({
                "track": track_num,
                "lba": lba,
                "type": "data" if is_data else "audio",
            })
            offset += 8

        return first_track, last_track, tracks

    def read_cd_frames(self, start_lba, num_frames):
        """
        Read raw audio frames using SCSI READ CD command (0xBE).

        Args:
            start_lba: Starting Logical Block Address
            num_frames: Number of frames to read

        Returns:
            tuple: (audio_data, status)
        """
        # READ CD command (0xBE)
        # Byte 1: Expected sector type (0x04 = CD-DA audio)
        # Bytes 2-5: Starting LBA (big-endian)
        # Bytes 6-8: Transfer length (big-endian)
        # Byte 9: 0x10 = include user data (2352 bytes)
        cdb = bytes([
            0xBE,  # READ CD
            0x04,  # Expected sector type: CD-DA
            (start_lba >> 24) & 0xFF,
            (start_lba >> 16) & 0xFF,
            (start_lba >> 8) & 0xFF,
            start_lba & 0xFF,
            (num_frames >> 16) & 0xFF,
            (num_frames >> 8) & 0xFF,
            num_frames & 0xFF,
            0x10,  # Include user data
            0,
            0,
        ])

        data_len = num_frames * CD_FRAMESIZE_RAW
        return self._send_command(cdb, data_len, timeout=60000)

    def rip_track(self, track_num, start_lba, end_lba, output_dir="."):
        """
        Rip a single audio track to WAV file.

        Args:
            track_num: Track number
            start_lba: Starting LBA
            end_lba: Ending LBA (exclusive)
            output_dir: Output directory

        Returns:
            str: Path to output WAV file
        """
        total_frames = end_lba - start_lba
        duration_sec = total_frames / FRAMES_PER_SECOND

        print(f"Track {track_num}: {total_frames} frames "
              f"({duration_sec:.1f}s / {duration_sec/60:.1f}m)")

        filename = os.path.join(output_dir, f"track{track_num:02d}.wav")
        audio_data = bytearray()

        current_lba = start_lba
        start_time = time.time()
        errors = 0

        while current_lba < end_lba:
            frames_to_read = min(self.chunk_size, end_lba - current_lba)
            data, status = self.read_cd_frames(current_lba, frames_to_read)

            if status != 0 or not data:
                errors += 1
                if errors > 10:
                    print(f"\n  Too many errors at LBA {current_lba}, aborting track")
                    break
                print(f"\n  Error at LBA {current_lba}, retrying...")
                time.sleep(0.1)
                continue

            audio_data.extend(data)
            current_lba += frames_to_read
            errors = 0  # Reset error count on success

            # Progress
            done = current_lba - start_lba
            progress = done * 100 // total_frames
            elapsed = time.time() - start_time
            speed = done / elapsed if elapsed > 0 else 0
            eta = (total_frames - done) / speed if speed > 0 else 0

            print(f"\r  {progress:3d}% | {done}/{total_frames} frames | "
                  f"{speed:.0f} frames/s | ETA: {eta:.0f}s   ", end="", flush=True)

        print()  # Newline after progress

        # Write WAV file
        with wave.open(filename, "wb") as wav:
            wav.setnchannels(2)  # Stereo
            wav.setsampwidth(2)  # 16-bit
            wav.setframerate(44100)  # 44.1kHz
            wav.writeframes(bytes(audio_data))

        file_size = os.path.getsize(filename)
        print(f"  Saved: {filename} ({file_size / 1024 / 1024:.1f} MB)")

        return filename

    def rip_all(self, output_dir=".", tracks_to_rip=None):
        """
        Rip all (or specified) audio tracks.

        Args:
            output_dir: Output directory
            tracks_to_rip: List of track numbers to rip, or None for all

        Returns:
            list: Paths to output WAV files
        """
        first, last, tracks = self.read_toc()
        print(f"CD has tracks {first}-{last}")

        os.makedirs(output_dir, exist_ok=True)

        wav_files = []
        for i, track in enumerate(tracks[:-1]):  # Last entry is lead-out
            if track["track"] == 0xAA:  # Lead-out marker
                continue

            track_num = track["track"]

            # Skip if not in requested tracks
            if tracks_to_rip and track_num not in tracks_to_rip:
                continue

            # Skip data tracks
            if track["type"] == "data":
                print(f"Track {track_num}: Skipping (data track)")
                continue

            start_lba = track["lba"]
            end_lba = tracks[i + 1]["lba"]

            wav_file = self.rip_track(track_num, start_lba, end_lba, output_dir)
            wav_files.append(wav_file)

        return wav_files


def print_toc(ripper):
    """Print Table of Contents."""
    info = ripper.inquiry()
    if info:
        print(f"\nDevice: {info['vendor']} {info['product']} (rev {info['revision']})")
        device_type = info['type']
        type_str = 'CD-ROM' if device_type == 5 else f'Unknown ({device_type})'
        print(f"Type: {type_str}")

    if not ripper.test_unit_ready():
        print("\nNo disc in drive or drive not ready")
        return

    first, last, tracks = ripper.read_toc()

    print(f"\nTable of Contents:")
    print(f"{'Track':>6} {'Type':>8} {'LBA':>10} {'Length':>10} {'Duration':>10}")
    print("-" * 50)

    for i, track in enumerate(tracks):
        if track["track"] == 0xAA:
            print(f"{'Lead-out':>6} {'-':>8} {track['lba']:>10}")
        else:
            length = tracks[i + 1]["lba"] - track["lba"] if i + 1 < len(tracks) else 0
            duration = length / FRAMES_PER_SECOND
            print(f"{track['track']:>6} {track['type']:>8} {track['lba']:>10} "
                  f"{length:>10} {duration:>9.1f}s")


def main():
    parser = argparse.ArgumentParser(
        description="Rip audio CDs on ChromeOS/Crostini without kernel driver support"
    )
    parser.add_argument(
        "-o", "--output",
        default="/tmp/cd-rip",
        help="Output directory (default: /tmp/cd-rip)"
    )
    parser.add_argument(
        "-t", "--tracks",
        help="Tracks to rip (comma-separated, e.g., '1,3,5')"
    )
    parser.add_argument(
        "--toc",
        action="store_true",
        help="Show Table of Contents only, don't rip"
    )
    parser.add_argument(
        "--chunk-size",
        type=int,
        default=75,
        help="Frames per USB transfer (default: 75, optimal range: 50-150)"
    )
    parser.add_argument(
        "--vendor-id",
        type=lambda x: int(x, 0),
        help="USB vendor ID (hex, e.g., 0x0e8d)"
    )
    parser.add_argument(
        "--product-id",
        type=lambda x: int(x, 0),
        help="USB product ID (hex, e.g., 0x1887)"
    )

    args = parser.parse_args()

    print("crostini-cd-rip - USB CD Ripper for ChromeOS/Crostini")
    print("=" * 50)

    try:
        ripper = USBCDRipper(
            vendor_id=args.vendor_id,
            product_id=args.product_id,
            chunk_size=args.chunk_size,
        )

        if args.toc:
            print_toc(ripper)
            return

        print_toc(ripper)

        tracks_to_rip = None
        if args.tracks:
            tracks_to_rip = [int(t) for t in args.tracks.split(",")]
            print(f"\nRipping tracks: {tracks_to_rip}")

        print(f"\nRipping to: {args.output}")
        print(f"Chunk size: {args.chunk_size} frames\n")

        wav_files = ripper.rip_all(args.output, tracks_to_rip)

        print(f"\n{'=' * 50}")
        print(f"Done! Ripped {len(wav_files)} tracks to {args.output}")
        print("\nTo convert to MP3:")
        print(f"  cd {args.output}")
        print("  for f in *.wav; do lame -V 2 \"$f\" \"${f%.wav}.mp3\"; done")

    except RuntimeError as e:
        print(f"Error: {e}")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n\nAborted by user")
        sys.exit(1)


if __name__ == "__main__":
    main()
