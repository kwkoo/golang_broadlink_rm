package rmweb

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/kwkoo/broadlinkrm"
)

// cmdType represents the different types of RemoteCommands.
type cmdType int

// All the different enumerations of CmdType.
const (
	SendCommand cmdType = iota
	Pause               // in milliseconds
	shutdown
)

// remoteCommand represents a single remote code sent to a single IR blaster.
type remoteCommand struct {
	commandType cmdType
	target      string
	data        string
}

// RemoteCommandMessage represents a series of RemoteCommands that need to be
// executed as a group. This is needed to ensure that all commands generated
// from a macro are executed sequentially.
type RemoteCommandMessage struct {
	commands []remoteCommand
}

// MessageFromSingleCommand is a convenience function that lets you generate a
// message with a single RemoteCommand.
func MessageFromSingleCommand(cmdtype cmdType, target, data string) RemoteCommandMessage {
	return RemoteCommandMessage{commands: []remoteCommand{{commandType: cmdtype, target: target, data: data}}}
}

// ShutdownMessage is a convenience function that generates a message with a
// shutdown command.
func ShutdownMessage() RemoteCommandMessage {
	return MessageFromSingleCommand(shutdown, "", "")
}

func (msg *RemoteCommandMessage) appendMessage(cmdtype cmdType, target, data string) {
	msg.commands = append(msg.commands, remoteCommand{commandType: cmdtype, target: target, data: data})
}

// SendWorker pulls RemoteCommandMessages off the channel and sends them to the
// IR blasters. Its purpose is to avoid multiple entities sending commands
// to the same IR blaster at the same time.
func SendWorker(ch chan RemoteCommandMessage, broadlink broadlinkrm.Broadlink, wg *sync.WaitGroup) {
	for msg := range ch {
		for _, cmd := range msg.commands {
			switch cmd.commandType {
			case SendCommand:
				err := broadlink.Execute(cmd.target, cmd.data)
				if err != nil {
					log.Printf("Error executing command: %v", err)
				}
			case Pause:
				interval, err := strconv.Atoi(cmd.data)
				if err != nil {
					log.Printf("Error processing pause interval (%v): %v", cmd.data, err)
					continue
				}
				time.Sleep(time.Duration(interval) * time.Millisecond)
			case shutdown:
				wg.Done()
				log.Print("SendWorker terminated")
				return
			}
		}
	}
}
