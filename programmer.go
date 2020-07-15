package microchipboot

import (
	"fmt"
	"io"
	"math"

	"github.com/marcinbor85/gohex"
)

// Programmer reprsents the high level interface that allows devices to be programmed.
type Programmer interface {
	Connect() error
	Disconnect()
	GetVersionInfo() VersionInfo
	LoadHex(data io.Reader) error
	Program() error
	Verify() error
	Reset() error
}

func loadHex(data io.Reader) (*gohex.Memory, error) {
	mem := gohex.NewMemory()
	err := mem.ParseIntelHex(data)
	if err != nil {
		return nil, err
	}
	return mem, nil
}

type progError struct {
	Address uint32
	Err     error
}

func (e *progError) Error() string {
	return fmt.Sprintf("error at %X: %v", e.Address, e.Err)
}

func (e *progError) Unwrap() error { return e.Err }

func writeSegments(segments []gohex.DataSegment, writeRowSize int, writeFunc func(uint32, []byte) error) error {
	// Convert the segments into row-aligned blocks of length writeRowSize
	blocks := make(map[uint32][]byte)
	for _, segment := range segments {
		for i, data := range segment.Data {
			byteAddress := segment.Address + uint32(i)
			rowAlignedAddress := byteAddress & ^uint32(writeRowSize-1)
			b, ok := blocks[rowAlignedAddress]
			if !ok {
				// Create a blank block
				b = make([]byte, writeRowSize)
				for i := range b {
					b[i] = 0xFF
				}
				blocks[rowAlignedAddress] = b
			}
			// Copy the data into the block
			b[byteAddress-rowAlignedAddress] = data
		}
	}
	// Now write the blocks to flash
	for addr, block := range blocks {
		pkgLog.Debugf("writing %v bytes at %X", len(block), addr)
		err := writeFunc(addr, block)
		if err != nil {
			return &progError{Address: addr, Err: err}
		}
	}
	return nil
}

func eraseSegments(segments []gohex.DataSegment, eraseRowSize int, eraseFunc func(uint32, uint16) error) error {
	for _, segment := range segments {
		start := segment.Address & ^uint32(eraseRowSize-1)
		num := uint16(math.Ceil(
			float64((segment.Address+uint32(len(segment.Data)))-start) /
				float64(eraseRowSize)))

		pkgLog.Debugf("erasing %v rows at %X", num, start)
		err := eraseFunc(start, num)
		if err != nil {
			return &progError{Address: start, Err: err}
		}
	}
	return nil
}

func verifySegmentsByReading(segments []gohex.DataSegment, writeRowSize int, readFunc func(uint32, uint16) ([]byte, error)) error {
	for _, segment := range segments {
		offset := 0
		for addr := segment.Address; addr-segment.Address < uint32(len(segment.Data)); addr, offset = addr+uint32(writeRowSize), offset+writeRowSize {
			chunk := segment.Data[offset:]
			if len(chunk) > writeRowSize {
				chunk = segment.Data[offset : offset+writeRowSize]
			}

			pkgLog.Debugf("verifying data at %X length %v", addr, len(chunk))
			data, err := readFunc(addr, uint16(len(chunk)))
			if err != nil {
				return fmt.Errorf("failed to read flash at address %X: %v", addr, err)
			}
			// Compare the bytes
			for i := range data {
				if data[i] != chunk[i] {
					return fmt.Errorf("mismatch at %X, expected %X read %X", addr+uint32(i), chunk[i], data[i])
				}
			}
		}
	}
	return nil
}

func verifySegmentsByChecksum(segments []gohex.DataSegment, checksumFunc func(uint32, uint16) (uint16, error)) error {
	// The maximum length to checksum needs to fit inside 16-bits and be an even number
	const maxChecksumChunk = math.MaxUint16 - 1
	for _, segment := range segments {
		offset := 0
		for addr := segment.Address; addr-segment.Address < uint32(len(segment.Data)); addr, offset = addr+uint32(maxChecksumChunk), offset+maxChecksumChunk {
			chunk := segment.Data[offset:]
			if len(chunk) > maxChecksumChunk {
				chunk = segment.Data[offset : offset+maxChecksumChunk]
			}

			pkgLog.Debugf("verifying checksum at %X length %v", addr, len(chunk))
			picsum, err := checksumFunc(addr, uint16(len(chunk)))
			if err != nil {
				return fmt.Errorf("failed to calculate checksum at address %X: %v", addr, err)
			}
			// Calculate the local checksum
			var localsum uint16
			for i := 0; i < len(chunk); i += 2 {
				localsum += uint16(chunk[i]) + (uint16(chunk[i+1]) << 8)
			}
			if picsum != localsum {
				return fmt.Errorf("checksum mismatch at %X, PIC: %X, local: %X", addr, picsum, localsum)
			}
		}
	}
	return nil
}
