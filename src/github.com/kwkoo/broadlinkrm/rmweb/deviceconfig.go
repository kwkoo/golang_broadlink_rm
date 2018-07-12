package rmweb

import (
	"encoding/json"
	"fmt"
	"io"
)

// DeviceConfig represents a manually configured device (versus a device that
// is discovered).
type DeviceConfig struct {
	IP         string `json:"ip"`
	Mac        string `json:"mac"`
	Key        string `json:"key"`
	ID         string `json:"id"`
	DeviceType int    `json:"type"`
}

// IngestDeviceConfig reads a JSON stream and returns a slice of DeviceConfig
// structs. You use this to manually add devices to the Broadlink struct,
// bypassing the need for the discovery process.
func IngestDeviceConfig(r io.Reader) ([]DeviceConfig, error) {
	dec := json.NewDecoder(r)

	d := []DeviceConfig{}
	err := dec.Decode(&d)
	if err != nil {
		return d, fmt.Errorf("error decoding device config JSON: %v", err)
	}

	return d, nil
}
