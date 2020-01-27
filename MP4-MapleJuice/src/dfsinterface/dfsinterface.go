package dfsinterface

// This file contains the main function for the program. The node is initialized, the two UDP monitors are started,
// and the RPC server is started for MP1 querying

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"log"
	"net/http"
	"net/rpc"
	"os"
	"time"

	"logQuery/rpc_export"
	"membership"
	"membership/node"
)

// UniversalLog for logging events
var UniversalLog *log.Logger

// UDPPort for all nodes
var UDPPort = 31337

// IntroPort is used for introduction
var IntroPort = 33333

// HashRingSize is how big our hashring is
const HashRingSize = 4294967296

func FileSystemRun(uiChan chan string) {

	//uiChan := make(chan string)

	myNode, err := node.InitNode(os.Stdin) // Initialize the current node
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	if err := myNode.Start(ctx, uiChan); err != nil { // Start the node's helper functions and monitors
		myNode.Logger.Println(err)
	}

	UniversalLog = myNode.Logger // Start logger

	// Generate a chan to send kill signal
	killSignal := make(chan bool, 1)
	//interrupt := make(chan os.Signal, 1)
	//signal.Notify(interrupt, os.Interrupt)
	// Contact introduction node to get initial membership list

	host, err := os.Hostname()
	if err != nil {
		UniversalLog.Fatal("Couldn't get hostname")
	}

	rpcID := -1
	if len(host) > 17 {
		rpcID, _ = strconv.Atoi(host[15:17])
	}

	me := node.OtherNode{
		NodeID:   myNode.NodeID,
		Hostname: myNode.Hostname,
		TCPPort:  myNode.TCPPort,
		UDPPort:  myNode.UDPPort,
		UDPAddr:  myNode.UDPAddr,
	}

	myNode.Members.Add(me) // Add ourself to our membership list

	membership.StartNode(myNode)
	// Set pointers in RPC Export

	rpc_export.SetKill(&killSignal)
	rpc_export.SetLogger(UniversalLog)
	rpc_export.SetNodeID(rpcID)

	// Make node for RPC Export
	rpcNode := new(rpc_export.Node)

	// Export all RPC Calls
	rpc.Register(rpcNode)
	// rpc.HandleHTTP()
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", 10000+rpcID))
	if e != nil {
		UniversalLog.Fatal("TCP Server Listen Error:", e)
	}
	// Handle RPC Calls
	go http.Serve(l, nil)

	rpcNode2 := new(node.RPCNode)

	// Export all RPC Calls
	rpc.Register(rpcNode2)
	rpc.HandleHTTP()
	l2, e := net.Listen("tcp", fmt.Sprintf(":%d", 20000+rpcID))
	if e != nil {
		UniversalLog.Fatal("TCP Server Listen Error:", e)
	}
	// Handle RPC Calls
	go http.Serve(l2, nil)

	// Send Join
	baseMsg := &node.Message{
		NodeID:   myNode.NodeID,
		Hostname: host,
		UDPPort:  31337,
		Orig:     "JOIN," + strconv.FormatUint(myNode.NodeID, 10) + "," + host,
	}
	joinMsg := &node.JoinMessage{
		Message: baseMsg,
		TCPPort: 10000 + rpcID,
	}
	myNode.Neighbors.L.Lock()
	for i := 1; i < 11; i++ { // Ask for introductions from all available nodes
		err := myNode.AskForIntroduction(IntroPort, i)
		if err != nil {
			fmt.Print("Error asking for introduction ")
			fmt.Println(err)
			continue
		}
	}
	time.Sleep(500 * time.Millisecond) // Wait for member lists to come in before sending joins

	err = myNode.SendJoin(joinMsg) // Tell other nodes in the network that we want to join
	myNode.Neighbors.L.Unlock()    // Release lock and start failure detection

	if err != nil {
		UniversalLog.Fatal("Failed to send 'Join' to neighbors!", err)
	}

	/*select { // Catches interrupts so we can end the program
	case <-interrupt:
		UniversalLog.Printf("@ Node Terminated Normally @")
		os.Exit(0)
	}*/
}
