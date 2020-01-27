package node

// This contains the functions for sending and handling leave messages
import (
	"net"
	"strconv"
)

// HandleLeave processes leave messages from other nodes
func (node *ThisNode) HandleLeave(msg *LeaveMessage) error {
	if node.Members.Contains(msg.NodeID) >= 0 { // We only care if the node that left is in our membership list
		node.Members.Remove(msg.NodeID) // If it is, remove it and update our neighbors
		node.Neighbors.Update(node.Members, node.NodeID)
		node.SendLeave(msg) // Forward the leave to our neighbors
		node.Logger.Printf("Removed member: (%d)\n", msg.NodeID)
		for i := 0; i < len(node.Members.Members); i++ {
			node.Logger.Printf("Current member list: Node %d (ID=%d)\n", node.Members.Members[i].TCPPort-10000, node.Members.Members[i].NodeID)
		}
	}
	return nil
}

// SendLeave sends a leave message to our neighbors
func (node *ThisNode) SendLeave(msg *LeaveMessage) error {
	if node.MessageCache.Contains(msg.Orig) {
		return nil // Check if we've already seen this message
	}
	node.MessageCache.Add(msg.Orig)

	var sentTo []*net.UDPAddr
	message := "LEAVE," + strconv.FormatUint(msg.NodeID, 10) + "," + msg.Hostname // Construct the leave message
	for i := 0; i < len(node.Neighbors.Neighbors); i++ {
		found := false
		for j := 0; j < len(sentTo); j++ {
			if sentTo[j] == node.Neighbors.Neighbors[i].UDPAddr { // Don't send the message to nodes we've already sent it to once
				found = true
			}
		}
		if found == false {
			sentTo = append(sentTo, node.Neighbors.Neighbors[i].UDPAddr)
			connection, err := net.DialUDP("udp", nil, node.Neighbors.Neighbors[i].UDPAddr) // Connect to our neighbors and send the message
			defer connection.Close()
			if err != nil {
				node.Logger.Print("Could not dial destination of LEAVE message: ")
				node.Logger.Println(err)
				return err
			}
			connection.Write([]byte(message))
		}
	}
	return nil
}
