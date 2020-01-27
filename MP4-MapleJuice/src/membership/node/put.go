package node

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// PutFile will add a new file to the system
func (node *ThisNode) PutFile(Source, SDFSName string) error {

	SDFSName = strings.ReplaceAll(SDFSName, "/", "^")

	_, err := os.Stat(Source)
	if os.IsNotExist(err) {
		return errors.New("file " + Source + " does not exist")
	}

	// Check for a conflict
	/*if node.IsConflicted(SDFSName) {
		// If so, wait for confirmation or timeout
		fmt.Println("Write conflict detected for " + strings.ReplaceAll(SDFSName, "^", "/") + "!")
		fmt.Println("This write will be ignored unless you type 'confirm' within 60 seconds. You can also type 'deny' to cancel it.")
		select {
		case doit := <-confirmChan:
			if !doit {
				return errors.New("User canceled the put to guard against conflicts")
			}

		case <-time.After(time.Second * 30):
			return errors.New("timout overriding write-write conflict. Aborting put")
		}
	}*/

	fmt.Println(time.Now().Format("15:04:05.000"))

	// Figure out which nodes should hold it
	responsibleNodes := node.GetResponsibleNodes(SDFSName)
	//fmt.Print("Nodes responsible for file " + strings.ReplaceAll(SDFSName, "^", "/") + " are:")
	//fmt.Println(responsibleNodes)

	// Copy to each node
	ackChan := make(chan bool, 0)
	acksReceived := 0
	sendFailures := 0

	quorum := node.GetQuorumSize()

	for _, responsibleNode := range responsibleNodes {
		go node.RSyncSend(Source, SDFSName, responsibleNode, ackChan)
	}

	// Wait for a quorum
	for {
		select {
		case ack := <-ackChan:
			if ack {
				fmt.Println("Got an ack ")
				acksReceived++
			} else {
				sendFailures++
			}

			// Do we have a quorum of success responses?
			if acksReceived >= quorum {
				goto nextPart
			}

			// Have so many failed that we can never get a quorum?
			if len(responsibleNodes)-sendFailures < quorum {
				return errors.New("Too many failures! Could not put file to a quorum")
			}

		// Time out if 3/4 copies never finish in 30 seconds
		case <-time.After(time.Second * 300):
			return errors.New("file adding timed out after 5 minutes")
		}
	}

nextPart:

	h := sha1.New()
	h.Write([]byte(SDFSName))
	hash := binary.BigEndian.Uint64(h.Sum(nil))

	// Announce the new file
	node.AnnounceNewFile(SDFSName, time.Now().UnixNano()/1000000, hash)

	return nil
}

// AnnounceNewFile after we know other nodes have it
func (node *ThisNode) AnnounceNewFile(name string, timestamp int64, hash uint64) error {

	// Create Message
	message := "NEWFILE," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname + "," + name + "," + strconv.FormatInt(timestamp, 10) + "," + strconv.FormatUint(hash, 10)

	// Send Message to all other nodes
	for _, member := range node.Members.Members {
		connection, err := net.DialUDP("udp", nil, member.UDPAddr) // Connect to and send the message
		defer connection.Close()
		if err != nil {
			node.Logger.Print("Could not dial destination of NEWFILE message: ")
			node.Logger.Println(err)
			return err
		}
		connection.Write([]byte(message))
	}
	return nil
}

// HandleNewFile appends a new file to our file list
func (node *ThisNode) HandleNewFile(msg *FileMessage) {
	node.Files.L.Lock()
	defer node.Files.L.Unlock()

	NewFileEntry := FileEntry{
		SDFSName:  msg.SDFSName,
		Hostname:  msg.Hostname,
		TimeAdded: msg.Updated,
		Hash:      msg.Hash,
	}

	for i := 0; i < len(node.Files.Files); i++ {
		if strings.Compare(node.Files.Files[i].SDFSName, msg.SDFSName) == 0 { // Have we already seen this file?
			node.Files.Files[i].Hash = msg.Hash
			node.Files.Files[i].TimeAdded = msg.Updated
			return
		}
	}
	node.Files.Files = append(node.Files.Files, NewFileEntry)

}
