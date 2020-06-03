package microchipboot

import (
	"fmt"
	"math"
	"os"

	"github.com/marcinbor85/gohex"
)

// Programmer reprsents the high level interface that allows devices to be programmed.
type Programmer interface {
	Connect() error
	Disconnect()
	GetVersionInfo() VersionInfo
	LoadHexFile(file string) error
	Program() error
	Verify() error
	Reset() error
}

func loadHexFile(fileName string) (*gohex.Memory, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	mem := gohex.NewMemory()
	err = mem.ParseIntelHex(file)
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
	for _, segment := range segments {
		offset := 0
		for addr := segment.Address; addr-segment.Address < uint32(len(segment.Data)); addr, offset = addr+uint32(writeRowSize), offset+writeRowSize {
			chunk := segment.Data[offset:]
			if len(chunk) > writeRowSize {
				chunk = segment.Data[offset : offset+writeRowSize]
			}
			err := writeFunc(addr, chunk)
			if err != nil {
				return &progError{Address: addr, Err: err}
			}
		}
	}
	return nil
}

func eraseSegments(segments []gohex.DataSegment, eraseRowSize int, eraseFunc func(uint32, uint16) error) error {
	for _, segment := range segments {
		start := segment.Address & ^uint32(eraseRowSize-1)
		num := uint16(math.Ceil(float64(len(segment.Data)) / float64(eraseRowSize)))

		err := eraseFunc(start, num)
		if err != nil {
			return &progError{Address: start, Err: err}
		}
	}
	return nil
}

func verifySegments(segments []gohex.DataSegment, writeRowSize int, readFunc func(uint32, uint16) ([]byte, error)) error {
	for _, segment := range segments {
		offset := 0
		for addr := segment.Address; addr-segment.Address < uint32(len(segment.Data)); addr, offset = addr+uint32(writeRowSize), offset+writeRowSize {
			chunk := segment.Data[offset:]
			if len(chunk) > writeRowSize {
				chunk = segment.Data[offset : offset+writeRowSize]
			}

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
