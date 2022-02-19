package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/amrbekhit/microchipboot"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var commands = map[string]func(microchipboot.Bootloader, []string){
	"ver":         processGetVersion,
	"readflash":   processReadFlash,
	"writeflash":  processWriteFlash,
	"eraseflash":  processEraseFlash,
	"readee":      processReadEE,
	"writeee":     processWriteEE,
	"readconfig":  processReadConfig,
	"writeconfig": processWriteConfig,
	"checksum":    processCalculateChecksum,
	"reset":       processReset,
}

type pic8ProfileOptions struct {
	Profile microchipboot.PIC8Profile
	Options microchipboot.PIC8Options
}

const appVersion = "0.2.2"

func main() {
	version := flag.Bool("version", false, "Prints the program version.")
	port := flag.String("port", "", "Serial port name.")
	baud := flag.Int("baud", 115200, "Baud rate.")
	verbose := flag.Bool("v", false, "Enable verbose logging.")
	before := flag.String("before", "", "Command to run before programming.")
	after := flag.String("after", "", "Command to run after programming has been completed successfully.")

	// Format an empty pic8ProfileOptions struct in YAML format as an example.
	buf := new(bytes.Buffer)
	enc := yaml.NewEncoder(buf)
	enc.Encode(pic8ProfileOptions{})
	profile := flag.String("profile", "", "Device profile yaml file. Example:\n\n"+buf.String())

	cmdList := []string{}
	for key := range commands {
		cmdList = append(cmdList, key)
	}
	command := flag.String("cmd", "", fmt.Sprintf("Command to run, one of: %+v\n"+
		"Memory read commands have the following usage: cmdname addr length, e.g. readflash 0x1000 32\n"+
		"Memory write commands have the following usage: cmdname addr datafile, e.g. writeflash 0x1000 datafile",
		cmdList))

	flag.Parse()

	if *version {
		fmt.Println(appVersion)
		return
	}

	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	microchipboot.SetLogger(log.StandardLogger())

	if *port == "" {
		log.Fatal("must specify port")
	}

	bootloader, err := microchipboot.NewSerialBootloader(*port, *baud)
	if err != nil {
		log.Fatalf("failed to initialise bootloader: %v", err)
	}

	switch {
	case *command != "":
		// Run a single command
		f, ok := commands[*command]
		if !ok {
			log.Fatalf("invalid command %v", *command)
		}
		if err = bootloader.Connect(); err != nil {
			log.Fatalf("failed to open bootloader: %v", err)
		}
		defer bootloader.Disconnect()
		f(bootloader, flag.Args())

	default:
		// Try and program a hex file
		if len(flag.Args()) != 1 {
			log.Fatalf("must specify hex file to program")
		}

		if *profile == "" {
			log.Fatalf("must specify a profile file")
		}

		f, err := ioutil.ReadFile(*profile)
		if err != nil {
			log.Fatalf("failed to open profile file: %v", err)
		}
		pic := new(pic8ProfileOptions)
		if err := yaml.Unmarshal(f, pic); err != nil {
			log.Fatalf("failed to parse profile file: %v", err)
		}

		// Run the before command
		if *before != "" {
			log.Infof("running before command...")
			if err := exec.Command(*before).Run(); err != nil {
				log.Fatalf("failed to run before command: %v", err)
			}
		}

		prog := microchipboot.NewPIC8Programmer(bootloader, pic.Profile, pic.Options)
		log.Infof("connecting to device...")
		if err := prog.Connect(); err != nil {
			log.Fatal(err)
		}
		defer prog.Disconnect()
		log.Infof("connected")

		file, err := os.Open(flag.Args()[0])
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		if err := prog.LoadHex(file); err != nil {
			log.Fatal(err)
		}
		log.Infof("hex file loaded")

		log.Infof("programming...")
		if err := prog.Program(); err != nil {
			log.Fatal(err)
		}

		log.Infof("verifying...")
		if err := prog.Verify(); err != nil {
			log.Fatal(err)
		}

		log.Infof("resetting...")
		if err := prog.Reset(); err != nil {
			log.Fatal(err)
		}
		log.Infof("complete")

		// Run the after command
		if *after != "" {
			log.Infof("running after command...")
			if err := exec.Command(*after).Run(); err != nil {
				log.Fatalf("failed to run after command: %v", err)
			}
		}
	}
}
