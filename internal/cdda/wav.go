package cdda

import (
	"encoding/binary"
)

// CD audio constants
const (
	SampleRate    = 44100 // Hz
	Channels      = 2     // Stereo
	BitsPerSample = 16
	BytesPerFrame = 2352 // Raw CD-DA frame size
)

// WriteWAV creates a WAV file from raw CD audio samples.
// This is a pure function: raw audio bytes → complete WAV file bytes.
//
// Input: raw 16-bit stereo PCM samples at 44.1kHz (CD-DA format)
// Output: complete WAV file including header
func WriteWAV(samples []byte) []byte {
	if samples == nil {
		samples = []byte{}
	}

	dataSize := uint32(len(samples))
	fileSize := 36 + dataSize // Total - 8 bytes for RIFF header

	// WAV header is 44 bytes
	header := make([]byte, 44)

	// RIFF header
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], fileSize)
	copy(header[8:12], "WAVE")

	// fmt subchunk
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // Subchunk1Size (16 for PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)  // AudioFormat (1 = PCM)
	binary.LittleEndian.PutUint16(header[22:24], Channels)
	binary.LittleEndian.PutUint32(header[24:28], SampleRate)

	byteRate := SampleRate * Channels * (BitsPerSample / 8) // 176400
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))

	blockAlign := Channels * (BitsPerSample / 8) // 4
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], BitsPerSample)

	// data subchunk
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataSize)

	// Combine header and data
	wav := make([]byte, 44+len(samples))
	copy(wav[0:44], header)
	copy(wav[44:], samples)

	return wav
}
