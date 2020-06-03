package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/amrbekhit/microchipboot"
	log "github.com/sirupsen/logrus"
)

func processGetVersion(bootloader microchipboot.Bootloader, args []string) {
	ver, err := bootloader.GetVersion()
	if err != nil {
		log.Fatalf("failed to read version: %v", err)
	}

	log.Infof("version info: %+v", ver)
}

func getAddrAndLen(args []string) (uint32, uint16) {
	if len(args) != 2 {
		log.Fatalf("expected: addr len")
	}
	addr, err := strconv.ParseUint(args[0], 0, 32)
	if err != nil {
		log.Fatalf("invalid address: %v", err)
	}
	len, err := strconv.ParseUint(args[1], 0, 16)
	if err != nil {
		log.Fatalf("invalid length: %v", err)
	}
	return uint32(addr), uint16(len)
}

func processReadFlash(bootloader microchipboot.Bootloader, args []string) {
	addr, len := getAddrAndLen(args)
	data, err := bootloader.ReadFlash(uint32(addr), uint16(len))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(hex.Dump(data))
}

func getAddrAndData(args []string) (uint32, []byte) {
	if len(args) != 2 {
		log.Fatalf("expected: addr datafile")
	}
	addr, err := strconv.ParseUint(args[0], 0, 32)
	if err != nil {
		log.Fatalf("invalid address: %v", err)
	}
	data, err := ioutil.ReadFile(args[1])
	if err != nil {
		log.Fatalf("failed to read data file: %v", err)
	}
	return uint32(addr), data
}

func processWriteFlash(bootloader microchipboot.Bootloader, args []string) {
	addr, data := getAddrAndData(args)
	err := bootloader.WriteFlash(addr, data)
	if err != nil {
		log.Fatalf("failed to write flash: %v", err)
	}
}

func processEraseFlash(bootloader microchipboot.Bootloader, args []string) {
	addr, blocks := getAddrAndLen(args)
	err := bootloader.EraseFlash(addr, blocks)
	if err != nil {
		log.Fatalf("failed to erase flash: %v", err)
	}
}

func processReadEE(bootloader microchipboot.Bootloader, args []string) {
	addr, len := getAddrAndLen(args)
	data, err := bootloader.ReadEE(addr, len)
	if err != nil {
		log.Fatalf("failed to read eeprom: %v", err)
	}
	fmt.Print(hex.Dump(data))
}

func processWriteEE(bootloader microchipboot.Bootloader, args []string) {
	addr, data := getAddrAndData(args)
	err := bootloader.WriteEE(addr, data)
	if err != nil {
		log.Fatalf("failed to write eeprom: %v", err)
	}
}

func processReadConfig(bootloader microchipboot.Bootloader, args []string) {
	addr, len := getAddrAndLen(args)
	data, err := bootloader.ReadConfig(addr, len)
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	fmt.Print(hex.Dump(data))
}

func processWriteConfig(bootloader microchipboot.Bootloader, args []string) {
	addr, data := getAddrAndData(args)
	err := bootloader.WriteConfig(addr, data)
	if err != nil {
		log.Fatalf("failed to write config: %v", err)
	}
}

func processCalculateChecksum(bootloader microchipboot.Bootloader, args []string) {
	addr, len := getAddrAndLen(args)
	checksum, err := bootloader.CalculateChecksum(addr, len)
	if err != nil {
		log.Fatal("failed to calculate checksum: %v", err)
	}
	fmt.Printf("checksum: %X\n", checksum)
}

func processReset(bootloader microchipboot.Bootloader, args []string) {
	err := bootloader.Reset()
	if err != nil {
		log.Fatalf("failed to reset: %v", err)
	}
}
