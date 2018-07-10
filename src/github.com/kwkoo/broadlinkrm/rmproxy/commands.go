package rmproxy

import (
	"encoding/json"
	"fmt"
	"io"
)

// Command represents a remote command code.
type Command struct {
	Group   string `json:"group"`
	Command string `json:"command"`
	Data    string `json:"data"`
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
