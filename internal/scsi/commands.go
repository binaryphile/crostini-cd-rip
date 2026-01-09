package scsi

// SCSI command opcodes
const (
	OpTestUnitReady = 0x00
	OpInquiry       = 0x12
	OpReadTOC       = 0x43
	OpReadCD        = 0xBE
)

// BuildTestUnitReady creates the CDB for TEST UNIT READY command.
// Returns 6-byte CDB.
func BuildTestUnitReady() []byte {
	return []byte{OpTestUnitReady, 0, 0, 0, 0, 0}
}

// BuildInquiry creates the CDB for INQUIRY command.
// Returns 6-byte CDB requesting 36 bytes of response.
func BuildInquiry() []byte {
	return []byte{OpInquiry, 0, 0, 0, 36, 0}
}

// BuildReadTOC creates the CDB for READ TOC command (LBA format).
// This uses LBA format (byte 1 = 0x00), not MSF format (0x02).
// Returns 10-byte CDB.
//
// NOTE: The Python version had a bug where it sent MSF format (0x02)
// but parsed the response as LBA. This caused track boundaries to be
// computed incorrectly (track 1 = entire CD).
func BuildReadTOC() []byte {
	// Byte 0: Opcode (0x43)
	// Byte 1: 0x00 = LBA format (NOT 0x02 = MSF format)
	// Byte 2-5: Reserved
	// Byte 6: Starting track (0 = all tracks)
	// Byte 7-8: Allocation length (1020 bytes max)
	// Byte 9: Control
	return []byte{
		OpReadTOC,
		0x00, // LBA format - IMPORTANT: was 0x02 (MSF) in buggy Python version
		0, 0, 0, 0, 0,
		0x03, 0xFC, // Allocation length = 1020
		0,
	}
}

// BuildReadCD creates the CDB for READ CD command (audio extraction).
// startLBA: Starting Logical Block Address
// numFrames: Number of 2352-byte frames to read
// Returns 12-byte CDB.
func BuildReadCD(startLBA, numFrames int) []byte {
	return []byte{
		OpReadCD,
		0x04, // Expected sector type: CD-DA audio
		byte(startLBA >> 24),
		byte(startLBA >> 16),
		byte(startLBA >> 8),
		byte(startLBA),
		byte(numFrames >> 16),
		byte(numFrames >> 8),
		byte(numFrames),
		0x10, // Include user data (2352 bytes)
		0, 0,
	}
}

// InquiryData represents parsed INQUIRY response
type InquiryData struct {
	DeviceType byte   // Peripheral device type (5 = CD-ROM)
	Vendor     string // 8 chars
	Product    string // 16 chars
	Revision   string // 4 chars
}

// ParseInquiry parses a 36-byte INQUIRY response.
// This is a pure function.
func ParseInquiry(data []byte) InquiryData {
	if len(data) < 36 {
		return InquiryData{}
	}

	return InquiryData{
		DeviceType: data[0] & 0x1F,
		Vendor:     trimString(data[8:16]),
		Product:    trimString(data[16:32]),
		Revision:   trimString(data[32:36]),
	}
}

// trimString trims trailing spaces from ASCII bytes
func trimString(b []byte) string {
	s := string(b)
	// Trim trailing spaces
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
