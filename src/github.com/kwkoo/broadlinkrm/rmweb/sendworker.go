package rmweb

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/kwkoo/broadlinkrm"
)

// CmdType represents the different types of RemoteCommands.
type CmdType int

// All the different enumerations of CmdType.
const (
	SendCommand CmdType = iota
	Pause               // in milliseconds
	Shutdown
)

// RemoteCommand represents a single remote code sent to a single IR blaster.
type RemoteCommand struct {
	CommandType CmdType
	Target      string
	Data        string
}

// SendWorker pulls RemoteCommands off the channel and sends them to the IR
// blasters. Its purpose is to avoid multiple entities sending commands
// to the same IR blaster at the same time.
func SendWorker(ch chan RemoteCommand, broadlink broadlinkrm.Broadlink, wg *sync.WaitGroup) {
	for cmd := range ch {
		switch cmd.CommandType {
		case SendCommand:
			err := broadlink.Execute(cmd.Target, cmd.Data)
			if err != nil {
				log.Printf("Error executing command: %v", err)
			}
		case Pause:
			interval, err := strconv.Atoi(cmd.Data)
			if err != nil {
				log.Printf("Error processing pause interval (%v): %v", cmd.Data, err)
				continue
			}
			time.Sleep(time.Duration(interval) * time.Millisecond)
		case Shutdown:
			wg.Done()
			log.Print("SendWorker terminated")
			return
		}
	}
}
