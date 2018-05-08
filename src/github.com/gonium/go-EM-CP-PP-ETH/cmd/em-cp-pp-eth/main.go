package main

import (
	"fmt"
	"github.com/goburrow/modbus"
	"github.com/gonium/go-EM-CP-PP-ETH"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"math/rand"
	"os"
	"time"
)

var (
	app = kingpin.New("em-cp-pp-eth", "An Interface for the Phoenix"+
		" Contact EM-CP-PP-ETH charge controller")
	verbose = app.Flag("verbose", "Verbose mode.").Short('v').Bool()
	host    = app.Flag("host", "Host to connect to, i.e."+
		"10.0.0.1").Short('h').String()
	port = app.Flag("port", "Port to connect to, i.e."+
		"502").Short('p').Default("502").Uint16()
	slaveid = app.Flag("slave", "slave id i.e. "+
		"180").Short('s').Default("180").Uint8()
	status = app.Command("status",
		"query the charge controller state").Default()
	reset = app.Command("reset",
		"reset the charge controller via HTTP")
	current = app.Command("current", "get and set the "+
		" actual charging current")
	getcurrent = current.Command("get",
		"get the charging current")
	setcurrent = current.Command("set",
		"set the charging current")
	chargecurrent = setcurrent.Arg("current", "Charge current to set"+
		" (amps)").Required().Uint16()

	avail = app.Command("avail", "make the charging station"+
		" (un)available")
	setavail = avail.Command("set", "make the charging station"+
		" (un)available")
	newavail = setavail.Arg("state",
		"true: available, false: unavailable").Required().Bool()
	getavail = avail.Command("get", "get the charging station"+
		" availability")

	digimode = app.Command("digimode", "make the charging station"+
		" (un)available")
	setdigimode = digimode.Command("set", "switch to digital "+
		"communication mode")
	newdigimode = setdigimode.Arg("state",
		"true: enabled, false: disabled").Required().Bool()
	getdigimode = digimode.Command("get", "get the digital communication"+
		" mode state")
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(
		"0.1.0").Author("Mathias Dalheimer")
	kingpin.CommandLine.Help = "An interface to the Phoenix Contact" +
		" EM-CP-PP-ETH charge controller"
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	if *host == "" {
		log.Fatal("Please specify the host to connect to, i.e." +
			" em-cp-pp-eth -h 10.0.0.1")
	}
	url := fmt.Sprintf("%s:%d", *host, *port)

	// Build a Modbus TCP connection to the controller
	handler := modbus.NewTCPClientHandler(url)
	handler.Timeout = 3 * time.Second
	handler.SlaveId = *slaveid
	handler.Logger = log.New(os.Stdout, "DEBUG ", log.LstdFlags)
	err := handler.Connect()
	if err != nil {
		log.Fatalf("Failed to connect: %s", err.Error())
	}
	defer handler.Close()
	modbusClient := modbus.NewClient(handler)

	// Initialize internal handlers
	// TODO: This might need to be refactored into a nice facade
	statusCache := EM_CP_PP_ETH.NewStatusCache(modbusClient)
	commander := EM_CP_PP_ETH.NewCommander(modbusClient)

	switch cmd {
	case status.FullCommand():
		err := statusCache.Refresh()
		if err != nil {
			log.Fatal("Failed to get status: %s", err.Error())
		}
		statusCache.WriteFormattedStatus(os.Stdout)

	case reset.FullCommand():
		log.Printf("Resetting host %s\n", *host)
		err := commander.HTTPHardReset(*host)
		if err != nil {
			log.Fatal("Failed to reset charge controller: %s", err.Error())
		}
		log.Printf("Reset sent")

	case getcurrent.FullCommand():
		result, err := commander.ReadActualChargingCurrent()
		if err != nil {
			log.Fatalf("Failed to read charging current: %s", err.Error())
		} else {
			log.Printf("Actual charging current: %d A", result)
		}

	case setcurrent.FullCommand():
		result, err := commander.WriteActualChargingCurrent(*chargecurrent)
		if err != nil {
			log.Fatalf("Failed to write charging current: %s", err.Error())
		} else {
			log.Printf("New charging current: %v A", result)
		}

	case setavail.FullCommand():
		err := commander.WriteChargingEnabled(*newavail)
		if err != nil {
			log.Fatalf("Failed to update availability: %s", err.Error())
		} else {
			log.Printf("New availability: %t", *newavail)
		}

	case getavail.FullCommand():
		result, err := commander.ReadChargingEnabled()
		if err != nil {
			log.Fatalf("Failed to read availability: %s", err.Error())
		} else {
			log.Printf("Charging station available is %t", result)
		}

	case setdigimode.FullCommand():
		err := commander.WriteDigimodeEnabled(*newdigimode)
		if err != nil {
			log.Fatalf("Failed to update availability: %s", err.Error())
		} else {
			log.Printf("Digital communication mode is %t", *newdigimode)
		}

	case getdigimode.FullCommand():
		result, err := commander.ReadDigimodeEnabled()
		if err != nil {
			log.Fatalf("Failed to read digital communication mode state: %s", err.Error())
		} else {
			log.Printf("Digital communication mode is %t", result)
		}

	}

}
