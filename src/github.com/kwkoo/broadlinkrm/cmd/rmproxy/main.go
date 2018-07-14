package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kwkoo/broadlinkrm"
	"github.com/kwkoo/broadlinkrm/rmweb"
)

const sendChannelSize = 20

func main() {
	skipDiscovery := false
	if len(os.Getenv("SKIPDISCOVERY")) > 0 {
		skipDiscovery = true
	} else {
		flag.BoolVar(&skipDiscovery, "skipdiscovery", false, "Skip the device discovery process")
	}
	key := os.Getenv("KEY")
	if len(key) == 0 {
		flag.StringVar(&key, "key", "", "A key that's used to authenticate incoming requests. This is a required part of all incoming URLs.")
	}
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		flag.IntVar(&port, "port", 8080, "HTTP listener port.")
	}
	roomsPath := os.Getenv("ROOMS")
	if len(roomsPath) == 0 {
		flag.StringVar(&roomsPath, "rooms", "", "Path to the JSON file specifying a room configuration.")
	}
	commandsPath := os.Getenv("COMMANDS")
	if len(commandsPath) == 0 {
		flag.StringVar(&commandsPath, "commands", "", "Path to the JSON file listing all remote commands.")
	}
	deviceConfigPath := os.Getenv("DEVICECONFIG")
	if len(deviceConfigPath) == 0 {
		flag.StringVar(&deviceConfigPath, "deviceconfig", "", "Path to the JSON file specifying device configurations.")
	}
	macrosPath := os.Getenv("MACROS")
	if len(macrosPath) == 0 {
		flag.StringVar(&macrosPath, "macros", "", "Path to the JSON file specifying macros.")
	}
	flag.Parse()
	mandatoryParameter("key", key)
	mandatoryParameter("rooms", roomsPath)
	mandatoryParameter("commands", commandsPath)

	rooms := initializeRooms(roomsPath, commandsPath)
	macros := initializeMacros(macrosPath, rooms)
	broadlink := initalizeBroadlink(deviceConfigPath, skipDiscovery)

	// Setup signal handling.
	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt)

	var wg sync.WaitGroup

	commandChannel := make(chan rmweb.RemoteCommandMessage, sendChannelSize)
	wg.Add(1)
	server := setupWebServer(port, broadlink, key, rooms, macros, commandChannel, &wg)

	wg.Add(1)
	go rmweb.SendWorker(commandChannel, broadlink, &wg)

	<-shutdown
	log.Print("Interrupt signal received, initiating shutdown process...")
	signal.Reset(os.Interrupt)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	commandChannel <- rmweb.ShutdownMessage()
	wg.Wait()
	close(commandChannel)

	log.Print("Shutdown successful")
}

func mandatoryParameter(key, value string) {
	if len(value) == 0 {
		fmt.Fprintf(os.Stderr, "Mandatory parameter %v missing - set it as a command line option or as an environment variable (%v).\n", key, strings.ToUpper(key))
		flag.Usage()
		os.Exit(1)
	}
}

func initializeRooms(roomsPath, commandsPath string) rmweb.Rooms {
	commandsFile, err := os.Open(commandsPath)
	if err != nil {
		log.Fatalf("Could not open commands JSON file %v: %v", commandsPath, err)
	}
	commands, err := rmweb.IngestCommands(commandsFile)
	commandsFile.Close()
	if err != nil {
		log.Fatalf("Error while processing commands JSON: %v", err)
	}

	log.Printf("Processed %d commands", len(commands))

	roomsFile, err := os.Open(roomsPath)
	if err != nil {
		log.Fatalf("Could not open rooms JSON file %v: %v", roomsFile, err)
	}
	rooms, err := rmweb.NewRooms(roomsFile, commands)
	roomsFile.Close()
	if err != nil {
		log.Fatalf("Error while processing rooms JSON: %v", err)
	}

	log.Printf("Processed %d rooms", rooms.Count())
	return rooms
}

func initializeMacros(macrosPath string, rooms rmweb.Rooms) map[string]rmweb.RemoteCommandMessage {
	empty := make(map[string]rmweb.RemoteCommandMessage)
	if len(macrosPath) == 0 {
		log.Print("No macros")
		return empty
	}

	macrosFile, err := os.Open(macrosPath)
	if err != nil {
		log.Fatalf("Could not open macros JSON file %v: %v", macrosPath, err)
	}
	macros, err := rmweb.IngestMacros(macrosFile, rooms)
	macrosFile.Close()
	if err != nil {
		log.Fatalf("Error while processing macros JSON: %v", err)
	}

	log.Printf("Processed %d macros", len(macros))
	return macros
}

func initalizeBroadlink(deviceConfigPath string, skipDiscovery bool) broadlinkrm.Broadlink {
	broadlink := broadlinkrm.NewBroadlink()

	if len(deviceConfigPath) > 0 {
		deviceConfigFile, err := os.Open(deviceConfigPath)
		if err != nil {
			log.Fatalf("Could not open device configurations JSON file %v: %v", deviceConfigPath, err)
		}
		dc, err := rmweb.IngestDeviceConfig(deviceConfigFile)
		deviceConfigFile.Close()
		if err != nil {
			log.Fatalf("Error while processing device configurations JSON: %v", err)
		}
		for _, d := range dc {
			err := broadlink.AddManualDevice(d.IP, d.Mac, d.Key, d.ID, d.DeviceType)
			if err != nil {
				log.Fatalf("Error adding manual device configuration: %v", err)
			}
		}
		log.Printf("Added %v devices manually", broadlink.Count())
		return broadlink
	}

	if !skipDiscovery {
		err := broadlink.Discover()
		if err != nil {
			log.Fatal(err)
		}
	}

	count := broadlink.Count()
	if count == 0 {
		log.Fatal("Did not discover any devices")
	}
	log.Printf("Discovered %d devices", count)
	return broadlink
}

func setupWebServer(port int, broadlink broadlinkrm.Broadlink, key string, rooms rmweb.Rooms, macros map[string]rmweb.RemoteCommandMessage, ch chan rmweb.RemoteCommandMessage, wg *sync.WaitGroup) *http.Server {
	proxy := rmweb.NewRMProxyWebServer(broadlink, key, rooms, macros, ch)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: proxy,
	}

	go func() {
		log.Print("Web server listening on port ", port)
		if err := server.ListenAndServe(); err != nil {
			wg.Done()
			if err == http.ErrServerClosed {
				log.Print("Web server graceful shutdown")
				return
			}
			log.Fatal(err)
		}
	}()

	return server
}
