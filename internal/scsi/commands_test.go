package scsi

import (
	"testing"
)

func TestBuildTestUnitReady(t *testing.T) {
	cdb := BuildTestUnitReady()

	if len(cdb) != 6 {
		t.Errorf("CDB length = %d, want 6", len(cdb))
	}
	if cdb[0] != OpTestUnitReady {
		t.Errorf("Opcode = 0x%02x, want 0x%02x", cdb[0], OpTestUnitReady)
	}
}

func TestBuildInquiry(t *testing.T) {
	cdb := BuildInquiry()

	if len(cdb) != 6 {
		t.Errorf("CDB length = %d, want 6", len(cdb))
	}
	if cdb[0] != OpInquiry {
		t.Errorf("Opcode = 0x%02x, want 0x%02x", cdb[0], OpInquiry)
	}
	if cdb[4] != 36 {
		t.Errorf("Allocation length = %d, want 36", cdb[4])
	}
}

func TestBuildReadTOC(t *testing.T) {
	cdb := BuildReadTOC()

	if len(cdb) != 10 {
		t.Errorf("CDB length = %d, want 10", len(cdb))
	}
	if cdb[0] != OpReadTOC {
		t.Errorf("Opcode = 0x%02x, want 0x%02x", cdb[0], OpReadTOC)
	}
	// CRITICAL: Verify LBA format (0x00), not MSF format (0x02)
	if cdb[1] != 0x00 {
		t.Errorf("Format byte = 0x%02x, want 0x00 (LBA format)", cdb[1])
	}
	// Allocation length should be 1020 (0x03FC)
	allocLen := int(cdb[7])<<8 | int(cdb[8])
	if allocLen != 1020 {
		t.Errorf("Allocation length = %d, want 1020", allocLen)
	}
}

func TestBuildReadCD(t *testing.T) {
	// Test reading 1 frame at LBA 150 (typical track 1 start)
	cdb := BuildReadCD(150, 1)

	if len(cdb) != 12 {
		t.Errorf("CDB length = %d, want 12", len(cdb))
	}
	if cdb[0] != OpReadCD {
		t.Errorf("Opcode = 0x%02x, want 0x%02x", cdb[0], OpReadCD)
	}
	if cdb[1] != 0x04 {
		t.Errorf("Sector type = 0x%02x, want 0x04 (CD-DA)", cdb[1])
	}

	// Check LBA (big-endian in bytes 2-5)
	lba := int(cdb[2])<<24 | int(cdb[3])<<16 | int(cdb[4])<<8 | int(cdb[5])
	if lba != 150 {
		t.Errorf("LBA = %d, want 150", lba)
	}

	// Check transfer length (big-endian in bytes 6-8)
	frames := int(cdb[6])<<16 | int(cdb[7])<<8 | int(cdb[8])
	if frames != 1 {
		t.Errorf("Transfer length = %d, want 1", frames)
	}

	if cdb[9] != 0x10 {
		t.Errorf("Subchannel = 0x%02x, want 0x10 (user data)", cdb[9])
	}
}

func TestBuildReadCD_LargeValues(t *testing.T) {
	// Test with larger LBA and frame count
	cdb := BuildReadCD(265288, 75)

	lba := int(cdb[2])<<24 | int(cdb[3])<<16 | int(cdb[4])<<8 | int(cdb[5])
	if lba != 265288 {
		t.Errorf("LBA = %d, want 265288", lba)
	}

	frames := int(cdb[6])<<16 | int(cdb[7])<<8 | int(cdb[8])
	if frames != 75 {
		t.Errorf("Transfer length = %d, want 75", frames)
	}
}

func TestParseInquiry(t *testing.T) {
	// Build mock INQUIRY response
	data := make([]byte, 36)
	data[0] = 0x05 // CD-ROM device type

	// Vendor at offset 8-15 (8 bytes)
	copy(data[8:16], "HL-DT-ST")

	// Product at offset 16-31 (16 bytes)
	copy(data[16:32], "DVDRAM GP65NB60 ")

	// Revision at offset 32-35 (4 bytes)
	copy(data[32:36], "1.00")

	info := ParseInquiry(data)

	if info.DeviceType != 5 {
		t.Errorf("DeviceType = %d, want 5 (CD-ROM)", info.DeviceType)
	}
	if info.Vendor != "HL-DT-ST" {
		t.Errorf("Vendor = %q, want %q", info.Vendor, "HL-DT-ST")
	}
	if info.Product != "DVDRAM GP65NB60" {
		t.Errorf("Product = %q, want %q", info.Product, "DVDRAM GP65NB60")
	}
	if info.Revision != "1.00" {
		t.Errorf("Revision = %q, want %q", info.Revision, "1.00")
	}
}

func TestParseInquiry_TooShort(t *testing.T) {
	data := make([]byte, 10) // too short

	info := ParseInquiry(data)

	// Should return empty struct, not panic
	if info.DeviceType != 0 {
		t.Errorf("DeviceType = %d, want 0", info.DeviceType)
	}
	if info.Vendor != "" {
		t.Errorf("Vendor = %q, want empty", info.Vendor)
	}
}
