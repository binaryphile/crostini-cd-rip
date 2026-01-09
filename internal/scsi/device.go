package scsi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/gousb"
)

// CD audio constants
const (
	FrameSize       = 2352 // Raw CD-DA frame size in bytes
	FramesPerSecond = 75   // CD audio frames per second
)

// Known USB CD drive IDs
var KnownDevices = []struct {
	VendorID  gousb.ID
	ProductID gousb.ID
	Name      string
}{
	{0x0e8d, 0x1887, "Hitachi-LG/MediaTek Slim Portable DVD Writer"},
	{0x152d, 0x2339, "JMicron USB CD/DVD"},
	{0x13fd, 0x0840, "Initio USB CD/DVD"},
	{0x1c6b, 0xa223, "Philips USB CD/DVD"},
}

// Device represents a USB CD/DVD drive
type Device struct {
	ctx    *gousb.Context
	dev    *gousb.Device
	config *gousb.Config
	intf   *gousb.Interface
	epIn   *gousb.InEndpoint
	epOut  *gousb.OutEndpoint
	tag    uint32
}

// OpenDevice opens a USB CD drive.
// If vendorID and productID are 0, it will auto-detect.
func OpenDevice(vendorID, productID gousb.ID) (*Device, error) {
	ctx := gousb.NewContext()

	var dev *gousb.Device
	var err error
	var deviceName string

	if vendorID != 0 && productID != 0 {
		// Open specific device
		dev, err = ctx.OpenDeviceWithVIDPID(vendorID, productID)
		if err != nil {
			ctx.Close()
			return nil, fmt.Errorf("open device: %w", err)
		}
		if dev == nil {
			ctx.Close()
			return nil, errors.New("device not found")
		}
		deviceName = fmt.Sprintf("0x%04x:0x%04x", vendorID, productID)
	} else {
		// Try known devices
		for _, known := range KnownDevices {
			dev, err = ctx.OpenDeviceWithVIDPID(known.VendorID, known.ProductID)
			if err == nil && dev != nil {
				deviceName = known.Name
				break
			}
		}
		if dev == nil {
			ctx.Close()
			return nil, errors.New("no USB CD drive found")
		}
	}

	// Set auto-detach for kernel driver
	if err := dev.SetAutoDetach(true); err != nil {
		// Not fatal - may not be supported
	}

	// Get configuration
	config, err := dev.Config(1)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("get config: %w", err)
	}

	// Find Mass Storage interface (class 8) or fallback to first interface with bulk endpoints
	var intf *gousb.Interface
	for _, iface := range config.Desc.Interfaces {
		for _, alt := range iface.AltSettings {
			if alt.Class == gousb.ClassMassStorage {
				intf, err = config.Interface(iface.Number, alt.Alternate)
				if err != nil {
					continue
				}
				break
			}
		}
		if intf != nil {
			break
		}
	}

	// Fallback: try first interface (some drives use vendor-specific class)
	if intf == nil {
		for _, iface := range config.Desc.Interfaces {
			intf, err = config.Interface(iface.Number, 0)
			if err == nil {
				break
			}
		}
	}

	if intf == nil {
		config.Close()
		dev.Close()
		ctx.Close()
		return nil, errors.New("no suitable interface found")
	}

	// Find IN and OUT endpoints
	var epIn *gousb.InEndpoint
	var epOut *gousb.OutEndpoint

	for _, ep := range intf.Setting.Endpoints {
		if ep.Direction == gousb.EndpointDirectionIn {
			epIn, err = intf.InEndpoint(ep.Number)
			if err != nil {
				continue
			}
		} else {
			epOut, err = intf.OutEndpoint(ep.Number)
			if err != nil {
				continue
			}
		}
	}

	if epIn == nil || epOut == nil {
		intf.Close()
		config.Close()
		dev.Close()
		ctx.Close()
		return nil, errors.New("could not find USB endpoints")
	}

	fmt.Printf("Found: %s\n", deviceName)
	fmt.Printf("Endpoints: OUT=0x%02x, IN=0x%02x\n",
		uint8(epOut.Desc.Address), uint8(epIn.Desc.Address))

	return &Device{
		ctx:    ctx,
		dev:    dev,
		config: config,
		intf:   intf,
		epIn:   epIn,
		epOut:  epOut,
		tag:    1,
	}, nil
}

// Close releases all USB resources
func (d *Device) Close() {
	if d.intf != nil {
		d.intf.Close()
	}
	if d.config != nil {
		d.config.Close()
	}
	if d.dev != nil {
		d.dev.Close()
	}
	if d.ctx != nil {
		d.ctx.Close()
	}
}

// SendCommand sends a SCSI command and receives response.
// Returns (data, status, error)
func (d *Device) SendCommand(cdb []byte, dataLen int, timeout time.Duration) ([]byte, byte, error) {
	// Build CBW
	direction := DirectionIn
	if dataLen == 0 {
		direction = DirectionOut
	}
	cbw := BuildCBW(d.tag, uint32(dataLen), byte(direction), cdb)
	d.tag++

	// Send CBW with timeout
	writeCtx, writeCancel := context.WithTimeout(context.Background(), timeout)
	defer writeCancel()

	n, err := d.epOut.WriteContext(writeCtx, cbw)
	if err != nil {
		return nil, 0xFF, fmt.Errorf("CBW write: %w", err)
	}
	if n != len(cbw) {
		return nil, 0xFF, fmt.Errorf("CBW short write: %d/%d bytes", n, len(cbw))
	}

	// Read data if expected
	var data []byte
	if dataLen > 0 {
		readCtx, readCancel := context.WithTimeout(context.Background(), timeout)
		defer readCancel()

		data = make([]byte, dataLen)
		n, err := d.epIn.ReadContext(readCtx, data)
		if err != nil {
			// Try to recover by reading CSW anyway
			data = nil
		} else {
			data = data[:n]
		}
	}

	// Read CSW
	cswCtx, cswCancel := context.WithTimeout(context.Background(), timeout)
	defer cswCancel()

	cswBuf := make([]byte, CSWSize)
	_, err = d.epIn.ReadContext(cswCtx, cswBuf)
	if err != nil {
		return data, 0xFF, fmt.Errorf("CSW read: %w", err)
	}

	csw, err := ParseCSW(cswBuf)
	if err != nil {
		return data, 0xFF, err
	}

	return data, csw.Status, nil
}

// Inquiry sends INQUIRY command and returns device info
func (d *Device) Inquiry() (*InquiryData, error) {
	cdb := BuildInquiry()
	data, status, err := d.SendCommand(cdb, 36, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if status != StatusPassed {
		return nil, fmt.Errorf("INQUIRY failed with status %d", status)
	}

	info := ParseInquiry(data)
	return &info, nil
}

// TestUnitReady checks if drive is ready (disc loaded)
func (d *Device) TestUnitReady() bool {
	cdb := BuildTestUnitReady()
	_, status, err := d.SendCommand(cdb, 0, 5*time.Second)
	return err == nil && status == StatusPassed
}

// ReadTOCRaw reads raw TOC data from CD
func (d *Device) ReadTOCRaw() ([]byte, error) {
	cdb := BuildReadTOC()
	data, status, err := d.SendCommand(cdb, 1020, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("READ TOC: %w", err)
	}
	if status != StatusPassed {
		return nil, fmt.Errorf("READ TOC failed with status %d", status)
	}
	return data, nil
}

// ReadCDFrames reads raw audio frames
func (d *Device) ReadCDFrames(startLBA, numFrames int) ([]byte, error) {
	cdb := BuildReadCD(startLBA, numFrames)
	dataLen := numFrames * FrameSize
	data, status, err := d.SendCommand(cdb, dataLen, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("READ CD: %w", err)
	}
	if status != StatusPassed {
		return nil, fmt.Errorf("READ CD failed with status %d at LBA %d", status, startLBA)
	}
	return data, nil
}
