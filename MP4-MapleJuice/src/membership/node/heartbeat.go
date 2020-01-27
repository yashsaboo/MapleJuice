package node

// This contains functions to send and handle heartbeats
import (
	"math/rand"
	"net"
	"strconv"
	"time"
)

// SendHeartbeats sends heartbeats to our neighbors
func (node *ThisNode) SendHeartbeats() error {
	message := "HEART," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname // Generate the heartbeat message
	for i := 0; i < len(node.Neighbors.Neighbors); i++ {
		if (len(node.Neighbors.Neighbors) == 4) && (i == 3) { // We only need to send it to 3 neighbors to maintain completeness up to 3 failures. Choose 0-2
			continue
		}
		connection, err := net.DialUDP("udp", nil, node.Neighbors.Neighbors[i].UDPAddr)
		defer connection.Close()
		if err != nil {
			node.Logger.Print("Could not dial destination of HEART message: ")
			node.Logger.Println(err)
			return err
		}

		// Can be used to simulate network errors for testing false postitives in failure detection
		simulatedNetworkErrorRate := 0
		if float64(simulatedNetworkErrorRate) < rand.Float64() {
			_, err = connection.Write([]byte(message)) // Send heartbeat to our neighbors
			if err != nil {
				node.Logger.Print("Error writing heartbeat message: ")
				node.Logger.Println(err)
				return err
			}
		}

	}
	return nil
}

// HandleHeartbeat processes incoming heartbeats to our node
func (node *ThisNode) HandleHeartbeat(msg *HeartMessage) error {
	sent := false
	curTime := time.Now()
	for i := 0; i < len(node.Neighbors.Neighbors); i++ {
		if node.Neighbors.Neighbors[i].NodeID == msg.NodeID { // Find which neighbor it was from
			node.Neighbors.Neighbors[i].LastHeartbeat = uint64(curTime.UnixNano() / 1000000) // Update their LastHeartbeat time
			sent = true                                                                      // Check to make sure the heartbeat is applied to at least one neighbor
		}
	}
	for i := 0; i < len(node.Members.Members); i++ {
		if node.Members.Members[i].NodeID == msg.NodeID {
			node.Members.Members[i].LastHeartbeat = uint64(curTime.UnixNano() / 1000000) // Also update the heartbeat in our membership list if the node is later promoted to neighbor
		}
	}

	if sent == false { // Someone sent us a heartbeat but we aren't neighbors with them. This shouldn't happen.
		node.Logger.Printf("Couldn't find neighbor to update heartbeat: %s\n", strconv.FormatUint(msg.NodeID, 10))
		node.Logger.Print("Current neighbor list is: ")
		for i := 0; i < len(node.Neighbors.Neighbors); i++ {
			node.Logger.Print(strconv.Itoa(node.Neighbors.Neighbors[i].TCPPort-10000) + " / " + strconv.FormatUint(node.Neighbors.Neighbors[i].NodeID, 10) + ", ")
		}
		node.Logger.Print("Current member list is: ")
		for i := 0; i < len(node.Members.Members); i++ {
			node.Logger.Print(strconv.Itoa(node.Members.Members[i].TCPPort-10000) + " / " + strconv.FormatUint(node.Members.Members[i].NodeID, 10) + ", ")
		}
		node.Logger.Println()
		node.Logger.Println()
	}

	return nil

}
