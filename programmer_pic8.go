package microchipboot

import (
	"fmt"

	"github.com/marcinbor85/gohex"
)

// PIC8 is a programmer for 8-bit PICs.
type PIC8 struct {
	bootloader Bootloader
	memory     *gohex.Memory
	profile    PIC8Profile
	options    PIC8Options
	info       VersionInfo

	flash  []gohex.DataSegment
	config []gohex.DataSegment
	eeprom []gohex.DataSegment
	id     []gohex.DataSegment
}

// PIC8Profile defines the memory structure for 8-bit PICs.
type PIC8Profile struct {
	BootloaderOffset uint32
	FlashSize        uint32
	EEPROMOffset     uint32
	EEPROMSize       uint32
	ConfigOffset     uint32
	ConfigSize       uint32
	IDOffset         uint32
	IDSize           uint32
}

// PIC8Options holds programming options.
type PIC8Options struct {
	ProgramEEPROM bool
	ProgramConfig bool
	ProgramID     bool
}

// NewPIC8Programmer creates a new programmer for 8-bit PICs.
func NewPIC8Programmer(bootloader Bootloader, profile PIC8Profile, options PIC8Options) Programmer {
	prog := new(PIC8)

	prog.bootloader = bootloader
	prog.profile = profile
	prog.options = options

	return prog
}

// LoadHexFile loads and parses the specified hex file.
func (p *PIC8) LoadHexFile(fileName string) error {
	var err error
	p.memory, err = loadHexFile(fileName)
	if err != nil {
		return err
	}

	validSegment := func(s *gohex.DataSegment, start, length uint32) bool {
		if s.Address >= start && s.Address+uint32(len(s.Data)) <= start+length {
			return true
		}
		return false
	}

	// Extract the various segments
	for _, segment := range p.memory.GetDataSegments() {
		switch {
		case validSegment(&segment, p.profile.BootloaderOffset, p.profile.FlashSize-p.profile.BootloaderOffset):
			p.flash = append(p.flash, segment)

		case validSegment(&segment, p.profile.IDOffset, p.profile.IDSize):
			p.id = append(p.id, segment)

		case validSegment(&segment, p.profile.ConfigOffset, p.profile.ConfigSize):
			// Unused configuration bytes are saved as 0xFF in the hex file,
			// but are read as 0x00 by the PIC. Therefore, replace any 0xFF's with 0x00.
			for i := range segment.Data {
				if segment.Data[i] == 0xFF {
					segment.Data[i] = 0
				}
			}
			p.config = append(p.config, segment)

		case validSegment(&segment, p.profile.EEPROMOffset, p.profile.EEPROMSize):
			p.eeprom = append(p.eeprom, segment)

		default:
			return fmt.Errorf("invalid data segment at address %X", segment.Address)
		}

	}
	return nil
}

// Connect establishes a connection with the PIC and gets the device info.
func (p *PIC8) Connect() error {
	var err error
	if err = p.bootloader.Connect(); err != nil {
		return fmt.Errorf("failed to open bootloader: %v", err)
	}
	// Get the device info
	p.info, err = p.bootloader.GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get device info: %v", err)
	}
	return nil
}

// Disconnect closes the connection with the PIC.
func (p *PIC8) Disconnect() {
	p.bootloader.Disconnect()
}

// GetVersionInfo returns the current device info.
func (p *PIC8) GetVersionInfo() VersionInfo {
	return p.info
}

// Program erases and writes the program data previously loaded with LoadHexFile.
func (p *PIC8) Program() error {
	// Erase flash
	if err := eraseSegments(p.flash, p.info.EraseRowSize, p.bootloader.EraseFlash); err != nil {
		return fmt.Errorf("failed to erase segment at %X: %v", err.(*progError).Address, err.(*progError).Err)
	}

	// Program flash
	if err := writeSegments(p.flash, p.info.WriteRowSize, p.bootloader.WriteFlash); err != nil {
		return fmt.Errorf("failed to write flash at address: %X: %v", err.(*progError).Address, err.(*progError).Err)
	}

	// Program EEPROM
	if p.options.ProgramEEPROM {
		if err := writeSegments(p.eeprom, p.info.WriteRowSize, p.bootloader.WriteEE); err != nil {
			return fmt.Errorf("failed to write eeprom at address: %X: %v", err.(*progError).Address, err.(*progError).Err)
		}
	}

	// Write Config
	if p.options.ProgramConfig {
		// // Erase the config
		// if err := eraseSegments(p.config, p.info.EraseRowSize, p.bootloader.EraseFlash); err != nil {
		// 	return fmt.Errorf("failed to erase config segment at %X: %v", err.(*progError).Address, err.(*progError).Err)
		// }
		// Flash the new config
		if err := writeSegments(p.config, p.info.WriteRowSize, p.bootloader.WriteConfig); err != nil {
			return fmt.Errorf("failed to write config at address: %X: %v", err.(*progError).Address, err.(*progError).Err)
		}
	}

	// Write ID
	if p.options.ProgramID {
		// // Erase the ID
		if err := eraseSegments(p.id, p.info.EraseRowSize, p.bootloader.EraseFlash); err != nil {
			return fmt.Errorf("failed to erase id segment at %X: %v", err.(*progError).Address, err.(*progError).Err)
		}
		// Flash the new ID data
		if err := writeSegments(p.id, p.info.WriteRowSize, p.bootloader.WriteFlash); err != nil {
			return fmt.Errorf("failed to write id at address: %X: %v", err.(*progError).Address, err.(*progError).Err)
		}
	}

	return nil
}

// Verify reads back the program memory and compares it to the data in the hex file.
func (p *PIC8) Verify() error {
	// Verify flash
	err := verifySegments(p.flash, p.info.WriteRowSize, p.bootloader.ReadFlash)
	if err != nil {
		return fmt.Errorf("failed to verify flash: %v", err)
	}

	// Verify EEPROM
	if p.options.ProgramEEPROM {
		err = verifySegments(p.eeprom, p.info.WriteRowSize, p.bootloader.ReadEE)
		if err != nil {
			return fmt.Errorf("failed to verify eeprom: %v", err)
		}
	}

	// Verify config
	if p.options.ProgramConfig {
		err = verifySegments(p.config, p.info.WriteRowSize, p.bootloader.ReadConfig)
		if err != nil {
			return fmt.Errorf("failed to verify config: %v", err)
		}
	}

	// Verify ID
	if p.options.ProgramID {
		err = verifySegments(p.id, p.info.WriteRowSize, p.bootloader.ReadFlash)
		if err != nil {
			return fmt.Errorf("failed to verify eeprom: %v", err)
		}
	}

	return nil
}

// Reset resets the PIC.
func (p *PIC8) Reset() error {
	return p.bootloader.Reset()
}
