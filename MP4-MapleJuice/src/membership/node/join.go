package node

// This contains functions to handle and send join messages
import (
	"net"
	"strconv"
	"time"
)

// HandleJoinMsg processes new joins from other nodes
func (node *ThisNode) HandleJoinMsg(msg *JoinMessage) error {
	var newNode OtherNode
	var err error

	newNode.NodeID = msg.NodeID
	newNode.TCPPort = msg.TCPPort
	newNode.UDPPort = msg.UDPPort
	newNode.Hostname = msg.Hostname
	newNode.LastHeartbeat = uint64(time.Now().UnixNano() / 1000000) // Create new node from message fields

	node.Logger.Printf("Node: %s just joined the network\n", newNode.Hostname)

	joinAddr := newNode.Hostname + ":" + strconv.Itoa(newNode.UDPPort)

	newNode.UDPAddr, err = net.ResolveUDPAddr("udp", joinAddr) // Resolve the UDP address of the new node
	if err != nil {
		node.Logger.Print("Couldn't resolve addres of new node: ")
		node.Logger.Println(err)
		return err
	}

	if node.Members.Contains(newNode.NodeID) < 0 { // If we haven't already added this node to our list, add it
		node.Members.Add(newNode)
		node.Neighbors.Update(node.Members, node.NodeID) // Update our membership list
		node.SendJoin(msg)                               // Forward the join to our neighbors
		node.Logger.Printf("Added new member: (%d)\n", newNode.NodeID)
		for i := 0; i < len(node.Members.Members); i++ {
			node.Logger.Printf("Current member list: Node %d (ID=%d)\n", node.Members.Members[i].TCPPort-10000, node.Members.Members[i].NodeID)
		}
	}

	return nil
}

// SendJoin sends a join request to our neighbors
func (node *ThisNode) SendJoin(msg *JoinMessage) error {
	node.Neighbors.Update(node.Members, node.NodeID)                                                               // Update our neighbor list just to be sure before we send/forward a join
	message := "JOIN," + strconv.FormatUint(msg.NodeID, 10) + "," + msg.Hostname + "," + strconv.Itoa(msg.TCPPort) // Generate message

	for i := 0; i < len(node.Neighbors.Neighbors); i++ {
		connection, err := net.DialUDP("udp", nil, node.Neighbors.Neighbors[i].UDPAddr) // Connect to each neighbor
		defer connection.Close()
		if err != nil {
			node.Logger.Print("Could not dial destination of JOIN message: ")
			node.Logger.Println(err)
			return err
		}
		node.Logger.Print("Sent JOIN message: " + message + " to ")
		_, err = connection.Write([]byte(message)) // Send the join message
		node.Logger.Println(node.Neighbors.Neighbors[i].UDPAddr)
		if err != nil {
			node.Logger.Print("Failed to send JOIN message to neighbor: ")
			node.Logger.Println(err)
			return err
		}
	}
	return nil
}
