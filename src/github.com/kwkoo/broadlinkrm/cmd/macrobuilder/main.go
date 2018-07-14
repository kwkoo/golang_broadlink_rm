package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// SimplifiedRoom is a slimmed-down struct solely for macrobuilder.
type SimplifiedRoom struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
}

// SimplifiedGroup is slimmed-down.
type SimplifiedGroup struct {
	Group    string   `json:"group"`
	Commands []string `json:"commands"`
}

var deployment string

func main() {
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
	mandatoryParameter("rooms", roomsPath)
	mandatoryParameter("commands", commandsPath)

	deployment = generateJSON(rooms(roomsPath), groups(commandsPath))

	http.HandleFunc("/", handlerIndex)
	http.HandleFunc("/index.html", handlerIndex)
	http.HandleFunc("/api", handlerJSON)
	log.Printf("Web server listening on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func mandatoryParameter(key, value string) {
	if len(value) == 0 {
		fmt.Fprintf(os.Stderr, "Mandatory parameter %v missing - set it as a command line option or as an environment variable (%v).\n", key, strings.ToUpper(key))
		flag.Usage()
		os.Exit(1)
	}
}

func rooms(roomsPath string) []SimplifiedRoom {
	roomsFile, err := os.Open(roomsPath)
	if err != nil {
		log.Fatalf("Could not open rooms JSON file %v: %v", roomsFile, err)
	}
	defer roomsFile.Close()
	dec := json.NewDecoder(roomsFile)
	rooms := []SimplifiedRoom{}
	err = dec.Decode(&rooms)
	if err != nil {
		log.Fatalf("Error decoding rooms JSON: %v", err)
	}
	return rooms
}

func groups(commandsPath string) []SimplifiedGroup {
	commandsFile, err := os.Open(commandsPath)
	if err != nil {
		log.Fatalf("Could not open commands JSON file %v: %v", commandsPath, err)
	}
	defer commandsFile.Close()
	dec := json.NewDecoder(commandsFile)
	rawc := []struct {
		Group   string `json:"group"`
		Command string `json:"command"`
	}{}
	err = dec.Decode(&rawc)
	if err != nil {
		log.Fatalf("Error decoding commands JSON: %v", err)
	}
	cmap := make(map[string][]string)
	for _, record := range rawc {
		_, ok := cmap[record.Group]
		if !ok {
			cmap[record.Group] = []string{}
		}
		cmap[record.Group] = append(cmap[record.Group], record.Command)
	}

	groups := []SimplifiedGroup{}
	for key, value := range cmap {
		groups = append(groups, SimplifiedGroup{Group: key, Commands: value})
	}
	return groups
}

func generateJSON(rooms []SimplifiedRoom, groups []SimplifiedGroup) string {
	deployment := struct {
		Rooms  []SimplifiedRoom  `json:"rooms"`
		Groups []SimplifiedGroup `json:"groups"`
	}{
		Rooms:  rooms,
		Groups: groups,
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	err := enc.Encode(deployment)
	if err != nil {
		log.Fatalf("Error encoding to JSON: %v", err)
	}
	return buf.String()
}

func handlerJSON(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, deployment)
}

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `<!DOCTYPE html>
	<html>
		<head>
			<link rel="icon" href="icon.png">
			<link rel="apple-touch-icon" href="icon.png">
			<link rel="apple-touch-startup-image" href="icon.png">
			<title>Macro Builder</title>
			<script type="text/javascript">
				var deployment;
				function initPage() {
					var xmlhttp = new XMLHttpRequest();
					xmlhttp.onreadystatechange = function() {
						if (this.readyState == 4) {
							if (this.status == 200) {
								try {
									deployment = JSON.parse(this.responseText);
									showContainer(0);
									return;
								} catch (e) {
									alert("Got an unexpected response while fetching JSON: " + e);
									return;
								}
							} else {
									alert("Got an unexpected response while fetching JSON: " + this.status);
									return;
							}
						}
					}
					xmlhttp.open("GET", "/api", true);
					xmlhttp.send();
	
				}
	
				function showContainer(index) {
					var containers = ["sendcontainer", "pausecontainer"];
					for (var i=0; i<containers.length; i++) {
						var el = getElement(containers[i]);
						if (i == index) {
							el.style.visibility = "visible";
							//el.style.display = "";
							initSendContainer();
						} else {
							el.style.visibility = "hidden";
							//el.style.display = "none";
						}
					}
				}
				
				function initSendContainer() {
					var rooms = getElement("rooms");
					clearElement(rooms);
	
					deployment.rooms.forEach(function(room) {
						rooms.appendChild(createOption(room.name));
					});
	
					initGroup();
				}
	
				function initGroup() {
					var room = getElement("rooms").value;
					var group = getElement("group");
					clearElement(group);
	
					var length = deployment.rooms.length;
					var groups = null;
	
					for (i=0; i<length; i++) {
						if (deployment.rooms[i].name == room) {
							groups = deployment.rooms[i].groups;
							break;
						}
					}
	
					if (groups == null) return;
					groups.forEach(function(g) {
						group.appendChild(createOption(g));
					});
	
					initCommand();
				}
	
				function initCommand() {
					var group = getElement("group").value;
					var command = getElement("command");
					clearElement(command);
	
					var length = deployment.groups.length;
					var commands = null;
	
					for (i=0; i<length; i++) {
						if (deployment.groups[i].group == group) {
							commands = deployment.groups[i].commands;
							break;
						}
					}
	
					if (commands == null) return;
					commands.forEach(function(c) {
						command.appendChild(createOption(c));
					});
				}
	
				function processAdd() {
					if (getElement("typesend").checked) {
						addInstruction("sendcommand " + getElement("rooms").value + " " + getElement("command").value);
					} else {
						var interval = getElement("interval").value.trim();
						if (!isNaN(parseInt(interval))) addInstruction("pause " + interval);
					}
				}
	
				function addInstruction(text) {
					var span = document.createElement("span");
					span.className = "insttext";
					span.innerText = text;
					
					var space = document.createElement("span");
					space.innerHTML = "&nbsp;"
	
					var close = document.createElement("button");
					close.className = "delbutton";
					close.addEventListener("click", function(event){delInst(event);});
					close.innerHTML = "&#9747;";
	
					var item = document.createElement("div");
					item.className = "institem";
					item.appendChild(span);
					item.appendChild(space);
					item.appendChild(close);
					item.addEventListener("dragstart", function(event){drag(event);});
					item.setAttribute("draggable", "true");
					item.addEventListener("drop", function(event){drop(event);});
					item.addEventListener("dragover", function(event){allowDrop(event);});
	
					getElement("instlist").appendChild(item);
				}
	
				function delInst(e) {
					list = getElement("instlist");
					list.removeChild(e.target.parentElement);
				}
	
				function drag(e) {
					var item = e.srcElement;
					var index = Array.from(getElement("instlist").children).indexOf(item);
	
					if (index != -1) {
						e.dataTransfer.setData("srcindex", index);
					}
				}
	
				function allowDrop(e) {
					e.preventDefault();
				}
	
				function drop(e) {
					var list = getElement("instlist");
					var item = e.target;
					if (item.tagName == "SPAN") item = item.parentElement;
	
					var targetIndex = Array.from(list.children).indexOf(item);
					var srcIndex = e.dataTransfer.getData("srcindex");
	
					if (targetIndex == null || srcIndex == null || isNaN(targetIndex) || isNaN(srcIndex) || targetIndex < 0 || srcIndex < 0 || srcIndex == targetIndex) return;
					srcEl = list.children[srcIndex];
					targetEl = list.children[targetIndex];
	
					srcEl.parentNode.removeChild(srcEl);
	
					if (srcIndex < targetIndex) {
						targetIndex--;
					}
					targetEl.parentNode.insertBefore(srcEl, targetEl);
				}
	
				function clearElement(el) {
					var length = el.options.length;
					for (i=length-1; i>=0; i--) el.remove(i);
				}
	
				function createOption(value) {
					var option = document.createElement("option");
					option.value = value;
					option.text = value;
	
					return option;
				}
	
				function getElement(id) {
					return document.getElementById(id);
				}
	
				function exportMacro() {
					var macro = {};
					macro.name = getElement("macroname").value;
					macro.instructions = [];
					getElement("instlist").childNodes.forEach(function(item) {
						macro.instructions.push(item.firstChild.innerText);
					});
					getElement("exporttext").innerText = JSON.stringify(macro);
					getElement("overlay").style.display = "flex";
				}
	
				function hideOverlay() {
					getElement("overlay").style.display = "none";
				}
			</script>
			<style>
				.container {
					width: 100%;
					height: 100%;
					display: flex;
					flex-direction: row;
					justify-content: space-around;
				}
				.leftcontainer {
					width: 30%;
				}
				.rightcontainer {
					width: 60%;
					background: lightgray;
					padding: 10px;
					overflow-y: scroll;
				}
				.typecontainer {
					position: relative;
				}
				#sendcontainer, #pausecontainer {
					width: 100%;
					height: 100%;
					position: absolute;
					top: 0;
					left: 0;
				}
				#pausecontainer {
					z-index: 10;
				}
				.instheader {
					font-size: 2em;
				}
				.label {
					font-size: 1em;
				}
				.inputtext {
					font-size: 1em;
				}
				.delbutton {
					color: white;
					background: red;
					float: right;
				}
				.instlist {
					border: 1px solid #555;
					list-style-type: none;
				}
				.insttext {
					font-family: monospace;
					float: left;
				}
				.institem {
					flex-direction: column;
					border-bottom: 1px solid #555;
					background: white;
					padding: 5px;
				}
				#overlay {
					position: fixed;
					display: none;
					width: 100%;
					height: 100%;
					top: 0; 
					left: 0;
					right: 0;
					bottom: 0;
					background-color: rgba(0,0,0,0.5);
					z-index: 50;
					flex-direction: column;
					padding: 20px;
					align-items: center;
					justify-content: center;
				}
			</style>
		</head>
		<body onload="initPage()">
			<div class="container">
				<div class="leftcontainer">
					<div class="label">Macro Name</div>
					<input id="macroname" class="inputtext" type="text" size="30" value="macro_test">
					<br><br>
					<div class="label">Command Type</div>
					<input id="typesend" name="cmdtype" type="radio" value="sendcommand" onclick="showContainer(0)" checked>Send Command</input>
					<input id="typepause" name="cmdtype" type="radio" value="pause" onclick="showContainer(1)">Pause</input>
					<br><br>
					<div class="typecontainer">
						<div id="sendcontainer">
							<div class="label">Rooms</div>
							<select id="rooms" name="rooms" onchange="initGroup()"></select>
							<br><br>
							<div class="label">Group</div>
							<select id="group" name="group" onchange="initCommand()"></select>
							<br><br>
							<div class="label">Command</div>
							<select id="command" name="command"></select>
							<br><br>
							<button onclick="processAdd()">Add</button>
						</div>
						<div id="pausecontainer">
								<div class="label">Interval</div>
								<input id="interval" name="interval" type="text" size="5">
								milliseconds
								<br><br>
								<button onclick="processAdd()">Add</button>
						</div>
					</div>
				</div>
				<div class="rightcontainer">
					<div class="instheader">Instructions</div>
					<br>
					<div id="instlist" class="instlist"></div>
					<br><br>
					<button onclick="exportMacro()">Export</button>
				</div>
			</div>
			<div id="overlay">
				<textarea id="exporttext" rows="10" cols="80"></textarea>
				<br><br>
				<button onclick="hideOverlay()">OK</button>
			</div>
		</body>
	</html>`)
}
