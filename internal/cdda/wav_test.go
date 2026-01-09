package cdda

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWriteWAV_Header(t *testing.T) {
	// 1 second of silence (44100 samples × 2 channels × 2 bytes = 176400 bytes)
	samples := make([]byte, 176400)

	wav := WriteWAV(samples)

	// WAV file should be header (44 bytes) + data
	expectedSize := 44 + len(samples)
	if len(wav) != expectedSize {
		t.Errorf("WAV size = %d, want %d", len(wav), expectedSize)
	}

	// Check RIFF header
	if string(wav[0:4]) != "RIFF" {
		t.Errorf("RIFF magic = %q, want \"RIFF\"", string(wav[0:4]))
	}

	// Check file size (total - 8 bytes for RIFF header)
	fileSize := binary.LittleEndian.Uint32(wav[4:8])
	if fileSize != uint32(len(wav)-8) {
		t.Errorf("File size = %d, want %d", fileSize, len(wav)-8)
	}

	// Check WAVE format
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("WAVE format = %q, want \"WAVE\"", string(wav[8:12]))
	}

	// Check fmt chunk
	if string(wav[12:16]) != "fmt " {
		t.Errorf("fmt chunk = %q, want \"fmt \"", string(wav[12:16]))
	}

	// fmt chunk size (16 for PCM)
	fmtSize := binary.LittleEndian.Uint32(wav[16:20])
	if fmtSize != 16 {
		t.Errorf("fmt size = %d, want 16", fmtSize)
	}

	// Audio format (1 = PCM)
	audioFormat := binary.LittleEndian.Uint16(wav[20:22])
	if audioFormat != 1 {
		t.Errorf("Audio format = %d, want 1 (PCM)", audioFormat)
	}

	// Channels (2 = stereo)
	channels := binary.LittleEndian.Uint16(wav[22:24])
	if channels != 2 {
		t.Errorf("Channels = %d, want 2", channels)
	}

	// Sample rate (44100 Hz)
	sampleRate := binary.LittleEndian.Uint32(wav[24:28])
	if sampleRate != 44100 {
		t.Errorf("Sample rate = %d, want 44100", sampleRate)
	}

	// Byte rate (44100 × 2 × 2 = 176400)
	byteRate := binary.LittleEndian.Uint32(wav[28:32])
	if byteRate != 176400 {
		t.Errorf("Byte rate = %d, want 176400", byteRate)
	}

	// Block align (2 × 2 = 4)
	blockAlign := binary.LittleEndian.Uint16(wav[32:34])
	if blockAlign != 4 {
		t.Errorf("Block align = %d, want 4", blockAlign)
	}

	// Bits per sample (16)
	bitsPerSample := binary.LittleEndian.Uint16(wav[34:36])
	if bitsPerSample != 16 {
		t.Errorf("Bits per sample = %d, want 16", bitsPerSample)
	}

	// Check data chunk
	if string(wav[36:40]) != "data" {
		t.Errorf("data chunk = %q, want \"data\"", string(wav[36:40]))
	}

	// Data size
	dataSize := binary.LittleEndian.Uint32(wav[40:44])
	if dataSize != uint32(len(samples)) {
		t.Errorf("Data size = %d, want %d", dataSize, len(samples))
	}
}

func TestWriteWAV_DataIntegrity(t *testing.T) {
	// Create known audio data pattern
	samples := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	wav := WriteWAV(samples)

	// Data should appear after 44-byte header
	data := wav[44:]
	if !bytes.Equal(data, samples) {
		t.Errorf("Data mismatch: got %v, want %v", data, samples)
	}
}

func TestWriteWAV_EmptyData(t *testing.T) {
	wav := WriteWAV(nil)

	// Should still produce valid header
	if len(wav) != 44 {
		t.Errorf("Empty WAV size = %d, want 44", len(wav))
	}

	// Data size should be 0
	dataSize := binary.LittleEndian.Uint32(wav[40:44])
	if dataSize != 0 {
		t.Errorf("Data size = %d, want 0", dataSize)
	}
}
