# microchipboot

Go implementation of the [Microchip Unified Bootloader](https://www.microchip.com/promo/unified-bootloaders) protocol.

[![GoDoc](https://godoc.org/github.com/amrbekhit/microchipboot?status.svg)](https://godoc.org/github.com/amrbekhit/microchipboot)

This package contains the following components:
 - `Bootloader`: low-level interface to the bootloader.
 - `Programmer`: high-level programming interface.
 - `microchipboot`: command line programming tool.

Supported transports:
 - Serial

## Installation
To install the package and the command line tool to your `GOPATH`:
```bash
go get github.com/amrbekhit/microchipboot/...
```

**Note:** Remove the `/...` at the end if you only want to install the library, but not the command line tool.

## Command Line Tool
The `cmd/microchipboot` directory contains the code for a command line tool that serves as both an example on how to use the library and a fully functional host program to allow HEX files to be uploaded to devices. The tool currently supports programming 8-bit PICs.

### Usage
Before using the host tool, a profile file must be created that describes the memory layout of the device to be programmed. An example for the PIC18F45K20 is shown below:

```yaml
profile:
  bootloaderoffset: 0x800
  flashsize: 0x8000
  eepromoffset: 0xF00000
  eepromsize: 256
  configoffset: 0x300000
  configsize: 14
  idoffset: 0x200000
  idsize: 8
options:
  programeeprom: true
  programconfig: false
  programid: false
  verifybyreading: true
```

To program a HEX file, run the following command:

```bash
microchipboot -port /dev/ttyUSB0 -profile profile.yaml program.hex
```

Individual bootloader commands can be run using the `-cmd` flag. See the help text for more information.

## Library
Programming functionality can be integrated into exisitng programs using the `Bootloader` and `Programmer` interfaces.

The `Bootloader` interface provides direct access to the individual bootloader commands. It abstracts away the communication transport (serial, ethernet, i2c, USB etc) and provides a unified way of interacting with the bootloader.

The `Programmer` interface implements the actual algorithms for loading a HEX file, erasing, programming and verifying the device. It uses a `Bootloader` to then send the necessary commands to the device.

The following example demonstrates how to use these two interfaces to program a device:

```go
// First create a bootloader using the necessary transport
bootloader, err := microchipboot.NewSerialBootloader("/dev/ttyUSB0", 115200)
if err != nil {
    log.Fatalf("failed to initialise bootloader: %v", err)
}
// Create a programmer that uses that bootloader
programmer := microchipboot.NewPIC8Programmer(bootloader, profile, options)

log.Printf("connecting to device...")
if err := programmer.Connect(); err != nil {
    log.Fatal(err)
}
defer programmer.Disconnect()
log.Printf("connected")

if err := programmer.LoadHexFile(flag.Args()[0]); err != nil {
    log.Fatal(err)
}
log.Printf("hex file loaded")

log.Printf("programming...")
if err := programmer.Program(); err != nil {
    log.Fatal(err)
}

log.Printf("verifying...")
if err := programmer.Verify(); err != nil {
    log.Fatal(err)
}

log.Printf("resetting...")
if err := programmer.Reset(); err != nil {
    log.Fatal(err)
}
log.Printf("complete")
```