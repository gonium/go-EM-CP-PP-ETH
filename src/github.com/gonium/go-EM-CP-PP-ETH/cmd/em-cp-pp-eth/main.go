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
	get = app.Command("get",
		"get the actual charging current")
	set = app.Command("set",
		"set the charging current")
	chargecurrent = set.Arg("current", "Charge current to set"+
		" (amps)").Required().Uint16()
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

	case get.FullCommand():
		result, err := commander.ReadActualChargingCurrent()
		if err != nil {
			log.Fatalf("Failed to read charging current: %s", err.Error())
		} else {
			log.Printf("Actual charging current: %d A", result)
		}

	case set.FullCommand():
		result, err := commander.WriteActualChargingCurrent(*chargecurrent)
		if err != nil {
			log.Fatalf("Failed to write charging current: %s", err.Error())
		} else {
			log.Printf("New charging current: %v A", result)
		}
	}
}
