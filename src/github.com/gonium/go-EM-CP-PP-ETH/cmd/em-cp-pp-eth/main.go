package main

import (
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
	url     = app.Flag("url", "Host:Port to connect to, i.e."+
		"10.0.0.1:502").Short('u').String()
	slaveid = app.Flag("slave", "slave id i.e. "+
		"180").Short('s').Default("180").Uint8()
	status = app.Command("status",
		"query the charge controller state").Default()
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

	if *url == "" {
		log.Fatal("Please specify the URL to connect to, i.e." +
			" em-cp-pp-eth -u 10.0.0.1:502")
	}

	// Build a Modbus TCP connection to the controller
	handler := modbus.NewTCPClientHandler(*url)
	handler.Timeout = 3 * time.Second
	handler.SlaveId = *slaveid
	handler.Logger = log.New(os.Stdout, "DEBUG ", log.LstdFlags)
	err := handler.Connect()
	if err != nil {
		log.Fatalf("Failed to connect: %s", err.Error())
	}
	defer handler.Close()
	client := modbus.NewClient(handler)
	statusCache := EM_CP_PP_ETH.NewStatusCache(client)

	switch cmd {
	case status.FullCommand():
		err := statusCache.Refresh()
		if err != nil {
			log.Fatal("Failed to get status: %s", err.Error())
		}
		statusCache.WriteFormattedStatus(os.Stdout)

	case get.FullCommand():
		result, err := statusCache.ReadActualChargingCurrent()
		if err != nil {
			log.Fatalf("Failed to read charging current: %s", err.Error())
		} else {
			log.Printf("Actual charging current: %d A", result)
		}
	case set.FullCommand():
		result, err := statusCache.WriteActualChargingCurrent(*chargecurrent)
		if err != nil {
			log.Fatalf("Failed to write charging current: %s", err.Error())
		} else {
			log.Printf("New charging current: %v A", result)
		}
	}
}
