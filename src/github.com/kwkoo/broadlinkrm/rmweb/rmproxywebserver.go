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
	broadlink broadlinkrm.Broadlink
	key       string
	rooms     Rooms
}

// NewRMProxyWebServer instantiates a new RMProxyWebServer struct.
func NewRMProxyWebServer(broadlink broadlinkrm.Broadlink, key string, rooms Rooms) RMProxyWebServer {
	return RMProxyWebServer{
		broadlink: broadlink,
		key:       key,
		rooms:     rooms,
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

func (proxy *RMProxyWebServer) handleExecute(w http.ResponseWriter, r *http.Request, room, command string) {
	w.Header().Set("Content-type", "text/plain")
	log.Printf("Execute %v in %v", command, room)
	host, data, err := proxy.rooms.RemoteCode(room, command)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		log.Printf("Error: %v", err)
		return
	}

	err = proxy.broadlink.Execute(host, data)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		log.Printf("Error: %v", err)
		return
	}

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
