package scsi

import (
	"encoding/binary"
	"errors"
)

// USB Mass Storage Bulk-Only protocol constants
const (
	CBWSignature = 0x43425355 // "USBC" little-endian
	CSWSignature = 0x53425355 // "USBS" little-endian
	CBWSize      = 31
	CSWSize      = 13
)

// Direction constants for CBW
const (
	DirectionOut = 0x00 // Host to device
	DirectionIn  = 0x80 // Device to host
)

// CSW status values
const (
	StatusPassed     = 0x00
	StatusFailed     = 0x01
	StatusPhaseError = 0x02
)

// CBW represents a Command Block Wrapper
type CBW struct {
	Tag           uint32
	DataLength    uint32
	Direction     byte
	LUN           byte
	CommandLength byte
	Command       [16]byte
}

// CSW represents a Command Status Wrapper
type CSW struct {
	Tag     uint32
	Residue uint32
	Status  byte
}

// BuildCBW creates a CBW from a SCSI CDB.
// This is a pure function: (tag, dataLen, direction, cdb) → 31 bytes
func BuildCBW(tag uint32, dataLen uint32, direction byte, cdb []byte) []byte {
	cbw := make([]byte, CBWSize)

	binary.LittleEndian.PutUint32(cbw[0:4], CBWSignature)
	binary.LittleEndian.PutUint32(cbw[4:8], tag)
	binary.LittleEndian.PutUint32(cbw[8:12], dataLen)
	cbw[12] = direction
	cbw[13] = 0 // LUN
	cbw[14] = byte(len(cdb))

	// Copy CDB into command field (max 16 bytes)
	for i := 0; i < len(cdb) && i < 16; i++ {
		cbw[15+i] = cdb[i]
	}

	return cbw
}

// ParseCSW parses a 13-byte CSW response.
// This is a pure function: bytes → (CSW, error)
func ParseCSW(data []byte) (CSW, error) {
	if len(data) < CSWSize {
		return CSW{}, errors.New("CSW too short")
	}

	sig := binary.LittleEndian.Uint32(data[0:4])
	if sig != CSWSignature {
		return CSW{}, errors.New("invalid CSW signature")
	}

	return CSW{
		Tag:     binary.LittleEndian.Uint32(data[4:8]),
		Residue: binary.LittleEndian.Uint32(data[8:12]),
		Status:  data[12],
	}, nil
}
