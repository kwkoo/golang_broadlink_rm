package rmweb

import (
	"encoding/json"
	"fmt"
	"io"
)

// CommandType represents the type of command - there are currently 2 types:
// SendCode which represents a command to send an IR or RF code and
// SendPowerState which switches a WiFi Power Outlet on or off.
type CommandType int

// Various enums representing different CmdTypes.
const (
	SendCode       CommandType = iota
	SendPowerState             // If a CmdType is set to this, Data is expected to be "0" or "1"
)

// Command represents a remote command code.
type Command struct {
	Group   string      `json:"group"`
	Command string      `json:"command"`
	CmdType CommandType `json:"type"`
	Data    string      `json:"data"`
}

// IngestCommands reads a JSON stream and returns a slice of Command structs.
func IngestCommands(r io.Reader) ([]Command, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	c := []Command{}
	err := dec.Decode(&c)
	if err != nil {
		return c, fmt.Errorf("error decoding commands JSON: %v", err)
	}

	return c, nil
}
