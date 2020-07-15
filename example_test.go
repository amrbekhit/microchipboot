package microchipboot

import (
	"log"
	"os"
)

func Example() {
	// First create a bootloader using the necessary transport
	bootloader, err := NewSerialBootloader("/dev/ttyUSB0", 115200)
	if err != nil {
		log.Fatalf("failed to initialise bootloader: %v", err)
	}
	// Populate the profile with the device memory map details
	profile := PIC8Profile{}
	// Specify programming options (such as whether EEPROM or configuration bits should be programmed)
	options := PIC8Options{}

	// Create a programmer that uses the previously created bootloader
	programmer := NewPIC8Programmer(bootloader, profile, options)

	log.Print("connecting to device...")
	if err := programmer.Connect(); err != nil {
		log.Fatal(err)
	}
	defer programmer.Disconnect()
	log.Print("connected")

	file, err := os.Open("firmware.hex")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if err := programmer.LoadHex(file); err != nil {
		log.Fatal(err)
	}
	log.Print("hex file loaded")

	log.Print("programming...")
	if err := programmer.Program(); err != nil {
		log.Fatal(err)
	}

	log.Print("verifying...")
	if err := programmer.Verify(); err != nil {
		log.Fatal(err)
	}

	log.Print("resetting...")
	if err := programmer.Reset(); err != nil {
		log.Fatal(err)
	}
	log.Print("complete")
}
