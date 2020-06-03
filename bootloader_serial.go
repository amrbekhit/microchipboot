package microchipboot

import (
	"fmt"
	"time"

	"github.com/tarm/serial"
)

type serialBootloader struct {
	portConfig serial.Config
	port       *serial.Port
}

// NewSerialBootloader creates a new bootloader using the serial transport.
func NewSerialBootloader(port string, baud int) (Bootloader, error) {
	b := new(serialBootloader)

	b.portConfig.Baud = baud
	b.portConfig.Name = port
	b.portConfig.ReadTimeout = time.Second

	return b, nil
}

func (b *serialBootloader) Connect() error {
	var err error
	b.port, err = serial.OpenPort(&b.portConfig)
	if err != nil {
		return err
	}
	// On Linux with USB serial ports, in order for flush to work properly
	// we need to delay a little before flushing to make sure that any
	// received data has made its way up the driver stack.
	// See https://stackoverflow.com/questions/13013387/clearing-the-serial-ports-buffer
	time.Sleep(time.Millisecond * 100)
	b.port.Flush()
	return nil
}

func (b *serialBootloader) Disconnect() {
	b.port.Close()
}

func (b *serialBootloader) recv(count int) ([]byte, error) {
	resp := make([]byte, 0, count)
	for len(resp) < cap(resp) {
		buf := make([]byte, cap(resp))
		n, err := b.port.Read(buf)
		if err != nil {
			return nil, err
		}
		resp = append(resp, buf[:n]...)
	}
	return resp, nil
}

func (b *serialBootloader) send(cmd Command) ([]byte, error) {
	tx := append([]byte{0x55}, cmd.GetBytes()...)
	b.port.Write(tx)
	// Wait for the echoed command
	echoLen := len(tx) - len(cmd.Data)
	echo, err := b.recv(echoLen)
	if err != nil {
		return nil, err
	}

	// Check that the echoed data matches the sent data
	for i := 0; i < echoLen; i++ {
		if i != 4 && i != 5 && tx[i] != echo[i] {
			return nil, fmt.Errorf("echo mismatch at position %v", i)
		}
	}

	// Now receive the actual response
	if cmd.ExpectsSuccessCode() {
		code, err := b.recv(1)
		if err != nil {
			return nil, err
		}
		if code[0] != ResultSuccess {
			return nil, fmt.Errorf("command returned code %v: %v", code[0], GetResponseCodeString(int(code[0])))
		}
	}
	resp := []byte{}
	if cmd.GetResponseLength() > 0 {
		resp, err = b.recv(cmd.GetResponseLength())
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (b *serialBootloader) GetVersion() (VersionInfo, error) {
	resp, err := b.send(NewGetVersionCommand())
	if err != nil {
		return VersionInfo{}, err
	}

	info, err := ParseGetVersionResponse(resp)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("failed to parse GetVersion response: %v", err)
	}
	return info, nil
}

func (b *serialBootloader) ReadFlash(address uint32, length uint16) ([]byte, error) {
	resp, err := b.send(NewReadFlashCommand(address, length))
	if err != nil {
		return nil, fmt.Errorf("read flash failed: %v", err)
	}
	return resp, nil
}

func (b *serialBootloader) WriteFlash(address uint32, data []byte) error {
	_, err := b.send(NewWriteFlashCommand(address, data))
	if err != nil {
		return fmt.Errorf("write flash failed: %v", err)
	}
	return nil
}

func (b *serialBootloader) EraseFlash(address uint32, numRows uint16) error {
	_, err := b.send(NewEraseFlashCommand(address, numRows))
	if err != nil {
		return fmt.Errorf("erase flash failed: %v", err)
	}
	return nil
}

func (b *serialBootloader) ReadEE(address uint32, length uint16) ([]byte, error) {
	resp, err := b.send(NewReadEECommand(address, length))
	if err != nil {
		return nil, fmt.Errorf("read eeprom failed: %v", err)
	}
	return resp, nil
}

func (b *serialBootloader) WriteEE(address uint32, data []byte) error {
	_, err := b.send(NewWriteEECommand(address, data))
	if err != nil {
		return fmt.Errorf("write eeprom failed: %v", err)
	}
	return nil
}

func (b *serialBootloader) ReadConfig(address uint32, length uint16) ([]byte, error) {
	resp, err := b.send(NewReadConfigCommand(address, length))
	if err != nil {
		return nil, fmt.Errorf("read config failed: %v", err)
	}
	return resp, nil
}

func (b *serialBootloader) WriteConfig(address uint32, data []byte) error {
	_, err := b.send(NewWriteConfigCommand(address, data))
	if err != nil {
		return fmt.Errorf("write config failed: %v", err)
	}
	return nil
}

func (b *serialBootloader) CalculateChecksum(address uint32, length uint16) (uint16, error) {
	resp, err := b.send(NewCalculateChecksumCommand(address, length))
	if err != nil {
		return 0, fmt.Errorf("calculate checksum failed: %v", err)
	}
	checksum := uint16(resp[0]) + 256*uint16(resp[1])
	return checksum, nil
}

func (b *serialBootloader) Reset() error {
	_, err := b.send(NewResetCommand())
	if err != nil {
		return fmt.Errorf("reset failed: %v", err)
	}
	return nil
}
