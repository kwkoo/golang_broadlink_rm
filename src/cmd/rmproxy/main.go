package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/kwkoo/broadlinkrm"
	"github.com/kwkoo/broadlinkrm/rmweb"
	"github.com/kwkoo/configparser"
)

const sendChannelSize = 20

func main() {
	config := struct {
		Skipdiscovery    bool   `usage:"Skip the device discovery process."`
		Key              string `mandatory:"true" usage:"A key that's used to authenticate incoming requests. This is a required part of all incoming URLs."`
		Port             int    `usage:"HTTP listener port." default:"8080"`
		Roomspath        string `mandatory:"true" env:"ROOMS" flag:"rooms" usage:"Path to the JSON file specifying a room configuration."`
		Commandspath     string `mandatory:"true" env:"COMMANDS" flag:"commands" usage:"Path to the JSON file listing all remote commands."`
		Deviceconfigpath string `env:"DEVICECONFIG" flag:"deviceconfig" usage:"Path to the JSON file specifying device configurations."`
		Macrospath       string `env:"MACROS" flag:"macros" usage:"Path to the JSON file specifying macros."`
		Hapath           string `env:"HOMEASSISTANT" flag:"homeassistant" usage:"Path to the JSON file specifying the connection details to the Home Assistant server."`
	}{}

	if err := configparser.Parse(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing configuration: %v\n", err)
		os.Exit(1)
	}

	var haconfig *rmweb.HomeAssistantConfig
	if len(config.Hapath) > 0 {
		haconfig = initializeHomeAssistantConfig(config.Hapath)
		log.Print("Successfully imported Home Assistant configuration")
	} else {
		log.Print("No Home Assistant config")
	}

	rooms := initializeRooms(config.Roomspath, config.Commandspath)
	macros := initializeMacros(config.Macrospath, rooms)
	broadlink := initalizeBroadlink(config.Deviceconfigpath, config.Skipdiscovery)

	// Setup signal handling.
	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, os.Interrupt)

	var wg sync.WaitGroup

	commandChannel := make(chan rmweb.RemoteCommandMessage, sendChannelSize)
	wg.Add(1)
	server := setupWebServer(config.Port, broadlink, config.Key, rooms, macros, haconfig, commandChannel, &wg)

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

func initializeHomeAssistantConfig(haPath string) *rmweb.HomeAssistantConfig {
	haFile, err := os.Open(haPath)
	if err != nil {
		log.Fatalf("Could not open Home Assistant JSON config file %v: %v", haPath, err)
	}
	config, err := rmweb.IngestHomeAssistantConfig(haFile)
	haFile.Close()
	if err != nil {
		log.Fatalf("Error while processing Home Assistant JSON config file %v: %v", haPath, err)
	}

	return config
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

func setupWebServer(port int, broadlink broadlinkrm.Broadlink, key string, rooms rmweb.Rooms, macros map[string]rmweb.RemoteCommandMessage, haconfig *rmweb.HomeAssistantConfig, ch chan rmweb.RemoteCommandMessage, wg *sync.WaitGroup) *http.Server {
	proxy := rmweb.NewRMProxyWebServer(broadlink, key, rooms, macros, haconfig, ch)
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
