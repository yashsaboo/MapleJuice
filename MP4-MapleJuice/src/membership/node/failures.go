package node

// Contains functions to check for failures, send failure messages, and handle incoming failure messages
import (
	"net"
	"strconv"
	"time"
)

// CheckForFailures checks to see if any of our neighbors have failed
func (node *ThisNode) CheckForFailures(timoutSeconds uint64) error {
	for i := 0; i < len(node.Neighbors.Neighbors); i++ {
		if (len(node.Neighbors.Neighbors) == 4) && (i == 0) { // Only need to check for failures from 3 neighbors. Choose 1-3
			continue
		}
		curTime := time.Now()
		millis := uint64(curTime.UnixNano() / 1000000)
		if node.Neighbors.Neighbors[i].LastHeartbeat+(timoutSeconds*1000) < millis { // Check when the last time we heard from our neighbor was
			// If it's passed the timeout, we mark them as having failed
			baseMsg := &Message{
				NodeID:   node.Neighbors.Neighbors[i].NodeID,
				Hostname: node.Neighbors.Neighbors[i].Hostname,
				Orig:     "FAIL," + strconv.FormatUint(node.Neighbors.Neighbors[i].NodeID, 10) + "," + node.Neighbors.Neighbors[i].Hostname, // Generate failure message
			}
			failMsg := &FailMessage{
				Message: baseMsg,
			}
			toRemove := node.Neighbors.Neighbors[i].NodeID

			node.SendFailure(failMsg) // Send failure message to our neighbors including the node we thought failed. This gives them one last chance to respond.
			node.Logger.Printf("Failure Detected in Node %d (ID=%d) Letting my neighbors know...\n", node.Neighbors.Neighbors[i].TCPPort-10000, node.Neighbors.Neighbors[i].NodeID)
			node.Members.Remove(node.Neighbors.Neighbors[i].NodeID) // Remove them from the membership list
			node.Neighbors.Update(node.Members, node.NodeID)        // Update our neighbor list
			node.Logger.Printf("Removed member: (%d)\n", toRemove)
			for i := 0; i < len(node.Members.Members); i++ {
				node.Logger.Printf("Current member list: Node %d (ID=%d)\n", node.Members.Members[i].TCPPort-10000, node.Members.Members[i].NodeID)
			}

		}
	}
	return nil
}

// SendFailure sends a failure message to our neighbors
func (node *ThisNode) SendFailure(msg *FailMessage) error {
	var sentTo []*net.UDPAddr // List of who we have already sent this message to so we reduce duplicates
	found := false
	message := "FAIL," + strconv.FormatUint(msg.NodeID, 10) + "," + msg.Hostname // Generate failure message
	for i := 0; i < len(node.Neighbors.Neighbors); i++ {
		for j := 0; j < len(sentTo); j++ {
			if sentTo[j] == node.Neighbors.Neighbors[i].UDPAddr {
				found = true
			}
		}
		if found == false {
			sentTo = append(sentTo, node.Neighbors.Neighbors[i].UDPAddr)
			connection, err := net.DialUDP("udp", nil, node.Neighbors.Neighbors[i].UDPAddr) // Connect to and send the message
			defer connection.Close()
			if err != nil {
				node.Logger.Print("Could not dial destination of FAIL message: ")
				node.Logger.Println(err)
				return err
			}
			connection.Write([]byte(message))
		}
	}
	return nil
}

// HandleFailure processes failure messages from other nodes
func (node *ThisNode) HandleFailure(msg *FailMessage) error {

	node.Logger.Printf("Handling a failure. My ID is %d and the node that failed is %d\n", node.NodeID, msg.NodeID)
	node.Logger.Println(time.Now())

	if msg.NodeID == node.NodeID { // Someone thought I failed! I will send a counteracting JOIN message.
		baseMsg := &Message{
			NodeID:   node.NodeID,
			Hostname: node.Hostname,
			UDPPort:  31337,
			Orig:     "JOIN," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname,
		}
		joinMsg := &JoinMessage{
			Message: baseMsg,
			TCPPort: node.TCPPort,
		}
		err := node.SendJoin(joinMsg)
		node.Logger.Println("Someone thought I failed. Resending join")
		if err != nil {
			node.Logger.Print("Error adding myself back when someone thought I failed: ")
			node.Logger.Println(err)
		}
		return nil
	}

	if node.MessageCache.Contains(msg.Orig) {
		return nil //We've aleady seen this
	}
	node.MessageCache.Add(msg.Orig)

	if node.Members.Contains(msg.NodeID) >= 0 { // We only care if the failure is for a node in our membership list
		node.Members.Remove(msg.NodeID)                  // Remove it from our list
		node.Neighbors.Update(node.Members, node.NodeID) // Update our neighbors
		node.SendFailure(msg)                            // Forward the failure to our neighbors
	} else {
		node.Logger.Printf("Ignoring extra failure message for node %d\n", msg.NodeID)
	}
	node.Logger.Print("Node failure from: ")
	node.Logger.Println(msg.Hostname)
	return nil
}
