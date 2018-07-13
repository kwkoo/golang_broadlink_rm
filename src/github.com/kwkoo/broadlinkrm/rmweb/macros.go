package rmweb

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Macro represents a series of instructions to be executed as a group. An
// example of a macro JSON would be:
// [
//   {"name":"media_on", "instructions":["sendcommand livingroom tv_on", "pause 3000", "sendcommand livingroom amp_on"]},
//   {"name":"media_off", "instructions":["sendcommand livingroom tv_off", "sendcommand livingroom amp_off"]}
// ]
//
// There are 2 types of instructions - sendcommand and pause.
//
// The first type of instruction consists of the following format:
// sendcommand ROOM COMMAND
//
// The second type of instruction consists of the following format:
// pause INTERVAL
//
// INTERVAL is in milliseconds.
type Macro struct {
	Name         string   `json:"name"`
	Instructions []string `json:"instructions"`
}

// IngestMacros reads a JSON stream and returns a map of RemoteCommandMessages.
func IngestMacros(r io.Reader, rooms Rooms) (map[string]RemoteCommandMessage, error) {
	m := make(map[string]RemoteCommandMessage)

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	macros := []Macro{}
	err := dec.Decode(&macros)
	if err != nil {
		return m, fmt.Errorf("error decoding macros JSON: %v", err)
	}

	for _, macro := range macros {
		var msg RemoteCommandMessage
		for _, inst := range macro.Instructions {
			if strings.HasPrefix(inst, "sendcommand ") {
				args := strings.Split(inst[len("sendcommand "):], " ")
				if len(args) != 2 || len(args[0]) == 0 || len(args[1]) == 0 {
					return m, fmt.Errorf("\"%v\" is an invalid instruction", inst)
				}
				target, data, err := rooms.RemoteCode(args[0], args[1])
				if err != nil {
					return m, fmt.Errorf("could not convert \"%v\" to remote code: %v", inst, err)
				}
				msg.appendMessage(SendCommand, target, data)
			} else if strings.HasPrefix(inst, "pause ") {
				interval := inst[len("pause "):]

				// make sure it's a valid integer
				_, err := strconv.Atoi(interval)
				if err != nil {
					return m, fmt.Errorf("pause interval \"%v\" is not a valid number: %v", inst[len("pause "):], err)
				}
				msg.appendMessage(Pause, "", interval)
			} else {
				return m, fmt.Errorf("\"%v\" is an invalid instruction", inst)
			}
		}
		m[macro.Name] = msg
	}

	return m, nil
}
