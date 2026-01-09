package scsi

import (
	"encoding/binary"
	"testing"
)

func TestBuildCBW(t *testing.T) {
	// Test INQUIRY command
	cdb := []byte{0x12, 0x00, 0x00, 0x00, 0x24, 0x00} // INQUIRY, 36 bytes
	cbw := BuildCBW(1, 36, DirectionIn, cdb)

	if len(cbw) != CBWSize {
		t.Errorf("CBW size = %d, want %d", len(cbw), CBWSize)
	}

	// Check signature
	sig := binary.LittleEndian.Uint32(cbw[0:4])
	if sig != CBWSignature {
		t.Errorf("CBW signature = 0x%08x, want 0x%08x", sig, CBWSignature)
	}

	// Check tag
	tag := binary.LittleEndian.Uint32(cbw[4:8])
	if tag != 1 {
		t.Errorf("CBW tag = %d, want 1", tag)
	}

	// Check data length
	dataLen := binary.LittleEndian.Uint32(cbw[8:12])
	if dataLen != 36 {
		t.Errorf("CBW data length = %d, want 36", dataLen)
	}

	// Check direction
	if cbw[12] != DirectionIn {
		t.Errorf("CBW direction = 0x%02x, want 0x%02x", cbw[12], DirectionIn)
	}

	// Check command length
	if cbw[14] != 6 {
		t.Errorf("CBW command length = %d, want 6", cbw[14])
	}

	// Check CDB copied correctly
	for i, b := range cdb {
		if cbw[15+i] != b {
			t.Errorf("CBW command[%d] = 0x%02x, want 0x%02x", i, cbw[15+i], b)
		}
	}
}

func TestBuildCBW_LongCommand(t *testing.T) {
	// 12-byte READ CD command
	cdb := []byte{
		0xBE, 0x04, // READ CD, CD-DA
		0x00, 0x00, 0x00, 0x96, // LBA = 150
		0x00, 0x00, 0x01, // Transfer length = 1
		0x10, 0x00, 0x00, // Include user data
	}
	cbw := BuildCBW(42, 2352, DirectionIn, cdb)

	if cbw[14] != 12 {
		t.Errorf("CBW command length = %d, want 12", cbw[14])
	}

	// Verify CDB
	for i, b := range cdb {
		if cbw[15+i] != b {
			t.Errorf("CBW command[%d] = 0x%02x, want 0x%02x", i, cbw[15+i], b)
		}
	}
}

func TestParseCSW_Valid(t *testing.T) {
	// Build a valid CSW
	data := make([]byte, CSWSize)
	binary.LittleEndian.PutUint32(data[0:4], CSWSignature)
	binary.LittleEndian.PutUint32(data[4:8], 42)  // tag
	binary.LittleEndian.PutUint32(data[8:12], 0)  // residue
	data[12] = StatusPassed

	csw, err := ParseCSW(data)
	if err != nil {
		t.Fatalf("ParseCSW error: %v", err)
	}

	if csw.Tag != 42 {
		t.Errorf("CSW tag = %d, want 42", csw.Tag)
	}
	if csw.Residue != 0 {
		t.Errorf("CSW residue = %d, want 0", csw.Residue)
	}
	if csw.Status != StatusPassed {
		t.Errorf("CSW status = %d, want %d", csw.Status, StatusPassed)
	}
}

func TestParseCSW_WithResidue(t *testing.T) {
	data := make([]byte, CSWSize)
	binary.LittleEndian.PutUint32(data[0:4], CSWSignature)
	binary.LittleEndian.PutUint32(data[4:8], 1)
	binary.LittleEndian.PutUint32(data[8:12], 100) // residue = 100
	data[12] = StatusPassed

	csw, err := ParseCSW(data)
	if err != nil {
		t.Fatalf("ParseCSW error: %v", err)
	}

	if csw.Residue != 100 {
		t.Errorf("CSW residue = %d, want 100", csw.Residue)
	}
}

func TestParseCSW_Failed(t *testing.T) {
	data := make([]byte, CSWSize)
	binary.LittleEndian.PutUint32(data[0:4], CSWSignature)
	binary.LittleEndian.PutUint32(data[4:8], 1)
	binary.LittleEndian.PutUint32(data[8:12], 0)
	data[12] = StatusFailed

	csw, err := ParseCSW(data)
	if err != nil {
		t.Fatalf("ParseCSW error: %v", err)
	}

	if csw.Status != StatusFailed {
		t.Errorf("CSW status = %d, want %d", csw.Status, StatusFailed)
	}
}

func TestParseCSW_InvalidSignature(t *testing.T) {
	data := make([]byte, CSWSize)
	binary.LittleEndian.PutUint32(data[0:4], 0xDEADBEEF) // wrong signature

	_, err := ParseCSW(data)
	if err == nil {
		t.Error("ParseCSW should fail with invalid signature")
	}
}

func TestParseCSW_TooShort(t *testing.T) {
	data := make([]byte, 5) // too short

	_, err := ParseCSW(data)
	if err == nil {
		t.Error("ParseCSW should fail with short data")
	}
}
