package rmweb

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Rooms contains a map of rooms.
type Rooms struct {
	rooms  map[string]Room
	groups map[string]map[string]Command
}

// Room maps groups to devices.
type Room struct {
	Name   string   `json:"name"`
	Host   string   `json:"host"`
	Groups []string `json:"groups"`
}

// NewRooms reads a JSON stream and returns a Rooms type.
func NewRooms(r io.Reader, commands []Command) (Rooms, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	s := []Room{}
	rms := Rooms{}
	rms.rooms = make(map[string]Room)
	err := dec.Decode(&s)
	if err != nil {
		return rms, fmt.Errorf("error decoding rooms JSON: %v", err)
	}

	for _, rm := range s {
		rms.addRoom(rm)
	}

	rms.groups = make(map[string]map[string]Command)
	for _, c := range commands {
		m, ok := rms.groups[c.Group]
		if !ok {
			m = make(map[string]Command)
			rms.groups[c.Group] = m
		}
		m[c.Command] = c
	}

	return rms, nil
}

// RemoteCode retrieves a particular host and remote code for a command in a room.
func (r Rooms) RemoteCode(roomName, commandName string) (string, string, error) {
	rm, ok := r.rooms[roomName]
	if !ok {
		return "", "", fmt.Errorf("room %v does not exist", roomName)
	}
	for _, g := range rm.Groups {
		group, ok := r.groups[g]
		if !ok {
			continue
		}
		command, ok := group[commandName]
		if !ok {
			continue
		}
		return rm.Host, command.Data, nil
	}
	return "", "", fmt.Errorf("command %v not found in room %v", commandName, roomName)
}

func (r *Rooms) addRoom(rm Room) {
	r.rooms[strings.ToLower(rm.Name)] = rm
}

// Count returns the number of rooms.
func (r Rooms) Count() int {
	return len(r.rooms)
}
