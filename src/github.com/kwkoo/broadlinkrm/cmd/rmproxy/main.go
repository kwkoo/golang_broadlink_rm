package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kwkoo/broadlinkrm"
	"github.com/kwkoo/broadlinkrm/rmproxy"
)

var (
	broadlink broadlinkrm.Broadlink
	key       string
	rooms     rmproxy.Rooms
)

func main() {
	key = os.Getenv("KEY")
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
	flag.Parse()
	mandatoryParameter("key", key)
	mandatoryParameter("rooms", roomsPath)
	mandatoryParameter("commands", commandsPath)

	initializeRooms(roomsPath, commandsPath)
	initalizeBroadlink()
	setupWebServer(port)
}

func mandatoryParameter(key, value string) {
	if len(value) == 0 {
		fmt.Fprintf(os.Stderr, "Mandatory parameter %v missing - set it as a command line option or as an environment variable (%v).\n", key, strings.ToUpper(key))
		flag.Usage()
		os.Exit(1)
	}
}

func initializeRooms(roomsPath, commandsPath string) {
	commandFile, err := os.Open(commandsPath)
	if err != nil {
		log.Fatalf("Could not open commands JSON file %v: %v", commandsPath, err)
	}
	commands, err := rmproxy.IngestCommands(commandFile)
	commandFile.Close()
	if err != nil {
		log.Fatalf("Error while processing commands JSON: %v", err)
	}

	log.Printf("Processed %d commands", len(commands))

	roomsFile, err := os.Open(roomsPath)
	if err != nil {
		log.Fatalf("Could not open rooms JSON file %v: %v", roomsFile, err)
	}
	rooms, err = rmproxy.NewRooms(roomsFile, commands)
	roomsFile.Close()
	if err != nil {
		log.Fatalf("Error while processing rooms JSON: %v", err)
	}

	log.Printf("Processed %d rooms", rooms.Count())
}

func initalizeBroadlink() {
	broadlink = broadlinkrm.NewBroadlink()
	err := broadlink.Discover()
	if err != nil {
		log.Fatal(err)
	}
	count := broadlink.Count()
	if count == 0 {
		log.Fatal("Did not discover any devices")
	}
	log.Printf("Discovered %d devices", count)
}

func setupWebServer(port int) {
	log.Print("Web server listening on port ", port)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Printf("Received request: %v", path)
	w.Header().Set("Content-type", "text/plain")
	if strings.HasPrefix(path, "/learn/") {
		components, authorized := processURI("/learn/", path)
		if !authorized {
			unauthorized(w, r)
			return
		}
		if len(components) != 1 {
			notfound(w, r, "Invalid command")
		}
		handleLearn(w, r, components[0])
		return
	}
	if strings.HasPrefix(path, "/execute/") {
		components, authorized := processURI("/execute/", path)
		if !authorized {
			unauthorized(w, r)
			return
		}
		if len(components) != 2 {
			notfound(w, r, "Invalid command")
		}
		handleExecute(w, r, components[0], components[1])
		return
	}

	notfound(w, r, fmt.Sprintf("%v is not a valid command", path))
}

// Strips the prefix off the URI, checks the first argument to ensure it
// matches the key, then returns the rest of the arguments in a slice of
// strings. It returns true if the key is valid.
func processURI(prefix, uri string) ([]string, bool) {
	uri = uri[len(prefix):]
	components := strings.Split(uri, "/")
	if (len(components) == 0) || (components[0] != key) {
		return components, false
	}
	return components[1:], true
}

func handleLearn(w http.ResponseWriter, r *http.Request, host string) {
	log.Printf("Learn %v", host)
	data, err := broadlink.Learn(host)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		log.Printf("Error: %v", err)
		return
	}
	fmt.Fprintln(w, data)
	return
}

func handleExecute(w http.ResponseWriter, r *http.Request, room, command string) {
	log.Printf("Execute %v in %v", command, room)
	host, data, err := rooms.RemoteCode(room, command)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		log.Printf("Error: %v", err)
		return
	}

	err = broadlink.Send(host, data)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		log.Printf("Error: %v", err)
		return
	}

	fmt.Fprintln(w, "OK")
	return
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintln(w, "Unauthorized")
	log.Print("Unauthorized")
}

func notfound(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintln(w, message)
	log.Print(message)
}