package rmweb

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/kwkoo/broadlinkrm"
)

// RMProxyWebServer is a consolidation of all web server logic.
type RMProxyWebServer struct {
	broadlink   broadlinkrm.Broadlink
	key         string
	rooms       Rooms
	macros      map[string]RemoteCommandMessage
	haconfig    *HomeAssistantConfig
	sendChannel chan RemoteCommandMessage
}

// NewRMProxyWebServer instantiates a new RMProxyWebServer struct.
func NewRMProxyWebServer(broadlink broadlinkrm.Broadlink, key string, rooms Rooms, macros map[string]RemoteCommandMessage, haconfig *HomeAssistantConfig, ch chan RemoteCommandMessage) RMProxyWebServer {
	return RMProxyWebServer{
		broadlink:   broadlink,
		key:         key,
		macros:      macros,
		rooms:       rooms,
		haconfig:    haconfig,
		sendChannel: ch,
	}
}

func (proxy RMProxyWebServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Printf("Received request: %v", path)
	if strings.HasPrefix(path, "/remote/") {
		components, authorized := proxy.processURI("/remote/", path)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handleRemote(w, r, components)
		return
	}

	if strings.HasPrefix(path, "/learn/") {
		components, authorized := proxy.processURI("/learn/", path)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(components) != 1 {
			http.Error(w, "Invalid command", http.StatusNotFound)
			return
		}
		proxy.handleLearn(w, r, components[0])
		return
	}
	if strings.HasPrefix(path, "/learnrf/") {
		components, authorized := proxy.processURI("/learnrf/", path)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(components) != 1 {
			http.Error(w, "Invalid command", http.StatusNotFound)
			return
		}
		proxy.handleLearnRF(w, r, components[0])
		return
	}
	if strings.HasPrefix(path, "/execute/") {
		components, authorized := proxy.processURI("/execute/", path)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(components) != 2 {
			http.Error(w, "Invalid command", http.StatusNotFound)
			return
		}
		proxy.handleExecute(w, r, components[0], components[1])
		return
	}
	if strings.HasPrefix(path, "/macro/") {
		components, authorized := proxy.processURI("/macro/", path)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(components) != 1 {
			http.Error(w, "Invalid command", http.StatusNotFound)
			return
		}
		proxy.handleMacro(w, r, components[0])
		return
	}
	if strings.HasPrefix(path, "/query/") {
		components, authorized := proxy.processURI("/query/", path)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(components) != 1 {
			http.Error(w, "Invalid command", http.StatusNotFound)
			return
		}
		proxy.handleQuery(w, r, components[0])
		return
	}
	if strings.HasPrefix(path, "/homeassistant/") {
		components, authorized := proxy.processURI("/homeassistant/", path)
		if !authorized {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(components) != 1 {
			http.Error(w, "Invalid command", http.StatusNotFound)
			return
		}
		proxy.handleHomeAssistant(w, r, components[0])
		return
	}

	http.Error(w, fmt.Sprintf("%v is not a valid command", path), http.StatusNotFound)
}

func (proxy *RMProxyWebServer) handleLearn(w http.ResponseWriter, r *http.Request, host string) {
	w.Header().Set("Content-type", "text/plain")
	log.Printf("Learn %v", host)
	data, err := proxy.broadlink.Learn(host)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		log.Printf("Error: %v", err)
		return
	}
	fmt.Fprintln(w, data)
	return
}

func (proxy *RMProxyWebServer) handleLearnRF(w http.ResponseWriter, r *http.Request, host string) {
	w.Header().Set("Content-type", "text/plain")
	log.Printf("Learn RF %v", host)
	data, err := proxy.broadlink.LearnRF(host)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		log.Printf("Error: %v", err)
		return
	}
	fmt.Fprintln(w, data)
	return
}

func (proxy *RMProxyWebServer) handleExecute(w http.ResponseWriter, r *http.Request, room, command string) {
	w.Header().Set("Content-type", "text/plain")
	log.Printf("Execute %v in %v", command, room)
	host, data, err := proxy.rooms.RemoteCode(room, command)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		log.Printf("Error: %v", err)
		return
	}

	proxy.sendChannel <- MessageFromSingleCommand(SendCommand, host, data)
	fmt.Fprintln(w, "OK")
	return
}

func (proxy *RMProxyWebServer) handleMacro(w http.ResponseWriter, r *http.Request, macroname string) {
	w.Header().Set("Content-type", "text/plain")
	log.Printf("Execute macro %v", macroname)

	msg, ok := proxy.macros[macroname]
	if !ok {
		errmsg := fmt.Sprintf("Error: %v is not a valid macro", macroname)
		fmt.Fprintln(w, errmsg)
		log.Print(errmsg)
		return
	}

	proxy.sendChannel <- msg
	fmt.Fprintln(w, "OK")
	return
}

func (proxy *RMProxyWebServer) handleQuery(w http.ResponseWriter, r *http.Request, host string) {
	w.Header().Set("Content-type", "text/plain")
	log.Printf("Query %v", host)
	state, err := proxy.broadlink.GetPowerState(host)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		log.Printf("Error: %v", err)
		return
	}
	fmt.Fprintln(w, state)
	return
}

func (proxy *RMProxyWebServer) handleHomeAssistant(w http.ResponseWriter, r *http.Request, command string) {
	w.Header().Set("Content-type", "text/plain")
	log.Printf("Execute Home Assistant command %v", command)

	if proxy.haconfig == nil {
		errmsg := "Not configured for Home Assistant"
		fmt.Fprintln(w, errmsg)
		log.Print(errmsg)
		return
	}

	err := proxy.haconfig.Execute(command)
	if err != nil {
		errmsg := fmt.Sprintf("Error making API call to Home Assistant: %v", err)
		fmt.Fprintln(w, errmsg)
		log.Print(errmsg)
		return
	}

	fmt.Fprintln(w, "OK")
	return
}

func handleRemote(w http.ResponseWriter, r *http.Request, components []string) {
	if len(components) < 1 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	path := components[0]
	if path == "" || path == "index.html" {
		w.Header().Set("Content-type", "text/html")
		fmt.Fprint(w, IndexHTML())
		return
	}
	if path == "icon.png" {
		w.Header().Set("Content-type", "image/png")
		w.Write(Icon())
		return
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

// Strips the prefix off the URI, checks the first argument to ensure it
// matches the key, then returns the rest of the arguments in a slice of
// strings. It returns true if the key is valid.
func (proxy *RMProxyWebServer) processURI(prefix, uri string) ([]string, bool) {
	uri = uri[len(prefix):]
	components := strings.Split(uri, "/")
	if (len(components) == 0) || (components[0] != proxy.key) {
		return components, false
	}
	return components[1:], true
}
