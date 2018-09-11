package rmweb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const clientTimeout = 10 // seconds

// CommandMapping maps a command string to information needed to invoke a Home
// Assistant REST API.
type CommandMapping struct {
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
	Payload  string `json:"payload"`
}

// HomeAssistantConfig contains the information necessary to connect to a Home
// Assistant server.
type HomeAssistantConfig struct {
	client   *http.Client
	Server   string                    `json:"server"`
	Password string                    `json:"password"`
	Commands map[string]CommandMapping `json:"commands"`
}

// IngestHomeAssistantConfig reads a JSON stream and returns a pointer to a new
// HomeAssistantConfig struct.
func IngestHomeAssistantConfig(r io.Reader) (*HomeAssistantConfig, error) {
	config := &HomeAssistantConfig{}

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	err := dec.Decode(config)
	if err != nil {
		return config, fmt.Errorf("error decoding Home Assistant configuration JSON: %v", err)
	}

	if len(config.Server) == 0 {
		return config, errors.New("Home Assistant server is not defined")
	}

	if !strings.HasSuffix(config.Server, "/") {
		config.Server = config.Server + "/"
	}

	if len(config.Password) == 0 {
		return config, errors.New("Home Assistant password is not defined")
	}

	config.client = &http.Client{Timeout: clientTimeout * time.Second}

	return config, nil
}

// Execute invokes the specified command via the Home Assistant REST API.
func (config HomeAssistantConfig) Execute(command string) error {
	cmd, ok := config.Commands[command]
	if !ok {
		return fmt.Errorf(`"%v" is not a valid command`, command)
	}

	if len(cmd.Method) == 0 {
		cmd.Method = "GET"
	}

	var reqBody *bytes.Buffer
	if len(cmd.Payload) > 0 {
		reqBody = bytes.NewBufferString(cmd.Payload)
	}

	req, err := http.NewRequest(cmd.Method, config.Server+cmd.Endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("error while creating request to Home Assistant server: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := config.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request to Home Assistant server: %v", err)
	}
	resp.Body.Close()

	return nil
}
