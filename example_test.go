package microchipboot

import (
	"flag"
	"log"
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
}
