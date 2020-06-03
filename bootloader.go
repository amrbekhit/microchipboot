// Package microchipboot implements the Microchip Unified Bootloader protocol
// (https://www.microchip.com/promo/unified-bootloaders).
//
// The package contains two main components: Bootloader and Programmer.
// Bootloader provides a transport-agnostic way of interacting with the individual
// bootloader commands. Programmer provides a high-level programming interface,
// allowing HEX files to be loaded, programmed and verified. It uses a provided
// Bootloader interface to communicate with the device.
//
// Also included is a command line tool, found in the cmd/microchipboot directory,
// that serves as both an example on how to use the library and a fully functional
// host program to allow HEX files to be uploaded to devices.
package microchipboot

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
)

// The Bootloader interface allows low-level interaction with the bootloader in a transport-agnostic fashion.
// For higher level programming operations, use the Programmer interface.
type Bootloader interface {
	Connect() error
	Disconnect()
	GetVersion() (VersionInfo, error)
	ReadFlash(address uint32, length uint16) ([]byte, error)
	WriteFlash(address uint32, data []byte) error
	EraseFlash(address uint32, numRows uint16) error
	ReadEE(address uint32, length uint16) ([]byte, error)
	WriteEE(address uint32, data []byte) error
	ReadConfig(address uint32, length uint16) ([]byte, error)
	WriteConfig(address uint32, data []byte) error
	CalculateChecksum(address uint32, length uint16) (uint16, error)
	Reset() error
}

// VersionInfo holds the results of the Request Version command.
type VersionInfo struct {
	VersionMinor, VersionMajor int
	MaxPacketSize              int
	DeviceID                   int
	EraseRowSize               int
	WriteRowSize               int
	ConfigWords                [4]byte
}

const (
	commandGetVersion        = 0x00
	commandReadFlash         = 0x01
	commandWriteFlash        = 0x02
	commandEraseFlash        = 0x03
	commandReadEE            = 0x04
	commandWriteEE           = 0x05
	commandReadConfig        = 0x06
	commandWriteConfig       = 0x07
	commandCalculateChecksum = 0x08
	commandReset             = 0x09
)

const (
	respLengthGetVersion = 16
	respLengthEraseFlash = 1
)

// Command result codes.
const (
	ResultSuccess      = 0x01
	ResultUnsupported  = 0xFF
	ResultAddressError = 0xFE
)

// GetResponseCodeString returns the string representation of a bootloader response code.
func GetResponseCodeString(code int) string {
	switch code {
	case ResultSuccess:
		return "success"
	case ResultUnsupported:
		return "unsupported"
	case ResultAddressError:
		return "address error"
	default:
		return "invalid response code"
	}
}

// Command represents a bootloader command.
type Command struct {
	Command        uint8
	UnlockSequence [2]byte
	Address        uint32
	Length         uint16
	Data           []byte
	// Response length, excluding the success code.
	responseLength     int
	expectsSuccessCode bool
}

// GetBytes returns a byte slice containing the data for the command.
func (c Command) GetBytes() []byte {
	b := []byte{c.Command}
	buf := new(bytes.Buffer)

	if len(c.Data) > 0 {
		c.Length = uint16(len(c.Data))
	}
	binary.Write(buf, binary.LittleEndian, c.Length)
	b = append(b, buf.Bytes()...)

	b = append(b, c.UnlockSequence[0], c.UnlockSequence[1])

	buf.Reset()
	binary.Write(buf, binary.LittleEndian, c.Address)
	b = append(b, buf.Bytes()...)

	b = append(b, c.Data...)
	return b
}

// GetResponseLength returns the expected number of response bytes.
func (c Command) GetResponseLength() int {
	return c.responseLength
}

// ExpectsSuccessCode returns true if the command expects a success code to be returned.
func (c Command) ExpectsSuccessCode() bool {
	return c.expectsSuccessCode
}

// NewGetVersionCommand returns the representation of the GetVersion command.
func NewGetVersionCommand() Command {
	c := Command{
		responseLength: respLengthGetVersion,
	}
	return c
}

// ParseGetVersionResponse parses the response of the GetVersionInfo command.
func ParseGetVersionResponse(data []byte) (VersionInfo, error) {
	if len(data) != respLengthGetVersion {
		return VersionInfo{}, errors.New("invalid response length")
	}

	resp := VersionInfo{
		VersionMinor:  int(data[0]),
		VersionMajor:  int(data[1]),
		MaxPacketSize: int(binary.LittleEndian.Uint16(data[2:])),
		DeviceID:      int(binary.LittleEndian.Uint16(data[6:])),
		EraseRowSize:  int(data[10]),
		WriteRowSize:  int(data[11]),
	}

	copy(resp.ConfigWords[:], data[12:])
	return resp, nil
}

// NewReadFlashCommand returns the representation of the ReadFlash command.
func NewReadFlashCommand(address uint32, length uint16) Command {
	c := Command{
		Command:        commandReadFlash,
		Address:        address,
		Length:         length,
		responseLength: int(length),
	}
	return c
}

// NewWriteFlashCommand returns the representation of the WriteFlash command.
func NewWriteFlashCommand(address uint32, data []byte) Command {
	c := Command{
		Command:            commandWriteFlash,
		Address:            address,
		Length:             uint16(len(data)),
		Data:               data,
		UnlockSequence:     [2]byte{0x55, 0xAA},
		expectsSuccessCode: true,
	}
	return c
}

// NewEraseFlashCommand returns the representation of the EraseFlash command.
func NewEraseFlashCommand(address uint32, numRows uint16) Command {
	c := Command{
		Command:            commandEraseFlash,
		Address:            address,
		Length:             numRows,
		UnlockSequence:     [2]byte{0x55, 0xAA},
		expectsSuccessCode: true,
	}
	return c
}

// NewReadEECommand returns the representation of the ReadEEPROM command.
func NewReadEECommand(address uint32, length uint16) Command {
	c := Command{
		Command:        commandReadEE,
		Address:        address,
		Length:         length,
		responseLength: int(length),
	}
	return c
}

// NewWriteEECommand returns the representation of the WriteEEPROM command.
func NewWriteEECommand(address uint32, data []byte) Command {
	c := Command{
		Command:            commandWriteEE,
		Address:            address,
		Length:             uint16(len(data)),
		Data:               data,
		UnlockSequence:     [2]byte{0x55, 0xAA},
		expectsSuccessCode: true,
	}
	return c
}

// NewReadConfigCommand returns the representation of the ReadConfig command.
func NewReadConfigCommand(address uint32, length uint16) Command {
	c := Command{
		Command:        commandReadConfig,
		Address:        address,
		Length:         length,
		responseLength: int(length),
	}
	return c
}

// NewWriteConfigCommand returns the representation of the WriteConfig command.
func NewWriteConfigCommand(address uint32, data []byte) Command {
	c := Command{
		Command:            commandWriteConfig,
		Address:            address,
		Length:             uint16(len(data)),
		Data:               data,
		UnlockSequence:     [2]byte{0x55, 0xAA},
		expectsSuccessCode: true,
	}
	return c
}

// NewCalculateChecksumCommand returns the representation of the CalculateChecksum command.
func NewCalculateChecksumCommand(address uint32, length uint16) Command {
	c := Command{
		Command:        commandCalculateChecksum,
		Address:        address,
		Length:         length,
		responseLength: 2,
	}
	return c
}

// NewResetCommand returns the representation of the Reset command.
func NewResetCommand() Command {
	c := Command{
		Command: commandReset,
	}
	return c
}
