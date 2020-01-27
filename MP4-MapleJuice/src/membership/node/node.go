package node

// This contains all the members and functions that define a node
import (
	//"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"logQuery"
)

// MeNode is a copy of our node
var MeNode *ThisNode

// InitNode will setup this node
func InitNode(conn io.Reader) (*ThisNode, error) {
	// Define Input Flags
	var tcpPort int
	flag.IntVar(&tcpPort, "p", 18080, "Specify the tcp port (>1024) to bind to (default 18080)")

	// Parse Input Flags
	flag.Parse()
	if !flag.Parsed() {
		// panic("Could not parse input")
	}
	rand.Seed(time.Now().UTC().UnixNano())
	NodeID := uint64(rand.Uint64()) // Generate our own NodeID randomly

	host, _ := os.Hostname()
	var rpcID int
	if len(host) > 17 {
		rpcID, _ = strconv.Atoi(host[15:17])
	} else {
		rpcID = -1
	}
	// Open log file
	logFile, err := os.OpenFile(
		fmt.Sprintf(LogLocation, rpcID),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644)
	if err != nil {
		return nil, err
	}

	tcpPort = 10000 + rpcID

	// Create Logger with format [DATE] [TIME] ???.go line ?? (Node ?) message
	logger := log.New(
		logFile,
		fmt.Sprintf("(Node %d) ", rpcID),
		(log.LstdFlags | log.Lshortfile))

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	introAddr := hostname + ":" + strconv.Itoa(UDPPort)

	remote, err := net.ResolveUDPAddr("udp", introAddr) // Resolve our own UDP address (ip:port)
	if err != nil {
		return nil, err
	}

	return &ThisNode{ // Return a completed node once everything is initialized
		conn: conn,
		OtherNode: &OtherNode{
			NodeID:   NodeID,
			Hostname: hostname,
			TCPPort:  tcpPort,
			UDPPort:  UDPPort,
			UDPAddr:  remote,
		},
		Neighbors: &NeighborList{
			L:         &sync.Mutex{},
			Neighbors: []OtherNode{},
		},
		Members: &MembershipList{
			L:       &sync.Mutex{},
			Members: []OtherNode{},
		},
		MessageCache: &RecentMessageCache{
			L:              &sync.Mutex{},
			RecentMessages: []RecentMessage{},
		},
		Files: &GlobalFileList{
			L:     &sync.Mutex{},
			Files: []FileEntry{},
		},
		GetsInProgress: &OngoingNeeds{
			L:      &sync.Mutex{},
			Labels: []OngoingNeedLabel{},
			Needs:  [][]PastHaveMessage{},
		},
		Logger: logger,
	}, nil
}

// Start will start the node and all its services
func (node *ThisNode) Start(ctx context.Context, uiChan chan string) error {
	node.ctx = ctx
	MeNode = node
	fmt.Println("Starting main loop")
	//reader := bufio.NewReader(node.conn)
	node.Active = true

	err := node.DeleteAllFiles()

	if err != nil {
		fmt.Println("Error deleting files " + err.Error())
	}

	//uiChan := make(chan string)
	//confirmChan := make(chan bool)

	// Spawn Heartbeat Thread
	go func() {
		fmt.Println("Started heartbeats")
		for {
			select {
			case <-node.ctx.Done():
				return
			case <-time.After(time.Millisecond * HeartbeatFrequencyMilliseconds):
				if node.Active {
					node.Neighbors.L.Lock()
					node.SendHeartbeats() // Send heartbeats to our neighbors
					node.Neighbors.L.Unlock()
				}
			}
		}
	}()

	// Spawn Recent Message Cache Flusher Thread
	go func() {
		fmt.Println("Started cache flusher")
		for {
			select {
			case <-node.ctx.Done():
				return
			case <-time.After(time.Second * MessageCacheFlushIntervalSeconds):
				if node.Active {
					node.MessageCache.FlushMessageCache(MessageCacheMaxAge) // Flush the message cache every so often so it doesn't overflow
				}

			}
		}
	}()

	// Spawn thread for stabilization
	go func() {
		fmt.Println("Started file monitor")
		for {
			select {
			case <-node.ctx.Done():
				return
			case <-time.After(time.Second * FileListRefreshSeconds):
				if node.Active {
					node.CheckFileStabilization()
				}
			}
		}
	}()

	// Spawn FailureDectection Thread
	go func() {
		fmt.Println("Started failure detection")
		for {
			select {
			case <-node.ctx.Done():
				return
			case <-time.After(time.Millisecond * FailureCheckFrequencyMilliseconds):
				if node.Active {
					node.CheckForFailures(FailureTimeoutSeconds) // Check for failures of our neighbors
				}
			}
		}
	}()

	/*go func() {
		fmt.Println("Get User Input")
		for {
			select {
			case <-node.ctx.Done():
				return
			case <-time.After(time.Millisecond):
				text, _ := reader.ReadString('\n') // Get user input from stdin for monitoring nodes
				if err != nil {
					node.Logger.Println("Error getting user input " + err.Error())
				}
				text = strings.TrimSuffix(text, "\n")
				if strings.Compare(text, "confirm") == 0 {
					confirmChan <- true
				} else if strings.Compare(text, "deny") == 0 {
					confirmChan <- false

				} else {
					uiChan <- text
				}

			}
		}
	}()*/

	go func() {
		// Main node controller loop
		fmt.Println("Started node controller")
		for {
			select {
			case <-node.ctx.Done():
				node.Logger.Printf("@ Node Terminated Normally @")
				os.Exit(0)
			case text := <-uiChan:

				fmt.Println("GETTING A REQUEST FROM THE UICHAN IN NODE.go!!!!!!!")
				fmt.Println(text+"\n")

				// User Interaction
				if strings.Contains(text, "grep") == true {

					// Load list of peers
					nodeList, err := logQuery.ReadNodeAddrFile("../logQuery/nodes.txt")
					if err != nil {
						fmt.Println("Error reading node file", err)
					}

					err = logQuery.DistributedGrep(strings.Split(text[:len(text)-1], " ")[1:], nodeList, "./out/machine.%d.log")
					if err != nil {
						fmt.Println("Error processing grep command: ", err)
					}
				}

				if (strings.Contains(text, "neighbor") == true) && (strings.Contains(text, "grep") == false) { // Print out our current neighbors
					fmt.Println("Current neighbors: ")
					j := -2
					for i := 0; i < len(node.Neighbors.Neighbors); i++ {
						fmt.Printf("Neighbor %d is Node %d (ID=%d)\n", j, node.Neighbors.Neighbors[i].TCPPort-10000, node.Neighbors.Neighbors[i].NodeID)
						j++
						if j == 0 {
							j++
						}
					}
					fmt.Println()
				}

				if (strings.Contains(text, "member") == true) && (strings.Contains(text, "grep") == false) { // Print our membership list
					fmt.Println("Current Membership List: ")
					for i := 0; i < len(node.Members.Members); i++ {
						fmt.Printf("Node %d (ID=%d)\n", node.Members.Members[i].TCPPort-10000, node.Members.Members[i].NodeID)
					}
					fmt.Println()
				}

				if (strings.Contains(text, "put") == true) && (strings.Contains(text, "grep") == false) { // Put a File
					parts := strings.Split(text, " ")
					if len(parts) < 3 {
						fmt.Println("Usage: put <path_to_local_file> <SDFS_file_name>")
						continue
					}
					err := node.PutFile(parts[1], strings.ReplaceAll(strings.TrimSuffix(parts[2], "\n"), "/", "^"))
					if err != nil {
						fmt.Println("Error adding new file " + err.Error())
					}
				}

				if (strings.Contains(text, "get") == true) && (strings.Contains(text, "grep") == false) { // Get a File
					parts := strings.Split(text, " ")
					if len(parts) < 3 {
						fmt.Println("Usage: get <SDFS_fie_name> <local_output_path>")
						continue
					}
					node.GetFile(strings.ReplaceAll(parts[1], "/", "^"), parts[2])
				}

				if (strings.Contains(text, "delete") == true) && (strings.Contains(text, "grep") == false) { // Print our membership list
					parts := strings.Split(text, " ")
					if len(parts) < 2 {
						fmt.Println("Usage: delete <SDFS_file_name>")
						continue
					}
					err := node.AnnounceDelete(strings.ReplaceAll(parts[1], "/", "^"))
					if err != nil {
						fmt.Println("Error deleting file " + err.Error())
					}
				}

				if (strings.Contains(text, "store") == true) && (strings.Contains(text, "grep") == false) { // Print our membership list
					localFiles := node.ListLocalFiles()
					fmt.Println("=== Local Store ===")
					for _, localFile := range localFiles {
						fmt.Println("- " + strings.ReplaceAll(localFile.SDFSName, "^", "/"))
					}
				}

				if (strings.Contains(text, "ls") == true) && (strings.Contains(text, "grep") == false) { // Print our membership list
					parts := strings.Split(text, " ")
					if len(parts) < 2 {
						fmt.Println("Please enter the file you want to find hosts for")
						continue
					}
					locations := node.ListFileLocations(strings.ReplaceAll(parts[1], "/", "^"))
					fmt.Println("Nodes with \"" + strings.ReplaceAll(parts[1], "^", "/") + "\" (" + strconv.Itoa(len(locations)) + ")")
					for _, location := range locations {
						fmt.Println("- " + location.Hostname)
					}
				}

				if (strings.Contains(text, "leave") == true) && (strings.Contains(text, "grep") == false) { // Command to voluntarily leave the network
					baseMsg := &Message{
						NodeID:   node.NodeID,
						Hostname: node.Hostname,
						UDPPort:  UDPPort,
						Orig:     "LEAVE," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname,
					}
					leaveMsg := &LeaveMessage{
						Message: baseMsg,
					}
					node.SendLeave(leaveMsg)
					node.Active = false
					node.Members.Members = nil
					node.Neighbors.Neighbors = nil
					fmt.Println("Leaving")
				}

				if (strings.Contains(text, "id") == true) && (strings.Contains(text, "grep") == false) { // Command to print my own NodeID
					fmt.Println("My ID is: " + strconv.FormatUint(node.NodeID, 10))
					node.Logger.Println("My ID is: " + strconv.FormatUint(node.NodeID, 10))
				}

				if (strings.Contains(text, "join") == true) && (strings.Contains(text, "grep") == false) { // Command to rejoin the network after you have left
					if len(node.Neighbors.Neighbors) != 0 {
						fmt.Println("Trying to join while already joined")
						break
					}
					// node.NodeID = uint64(rand.Intn(HashRingSize)) // Generate a new ID each time we rejoin
					node.NodeID = uint64(rand.Uint64())
					me := OtherNode{
						NodeID:   node.NodeID,
						Hostname: node.Hostname,
						TCPPort:  node.TCPPort,
						UDPPort:  node.UDPPort,
						UDPAddr:  node.UDPAddr,
					}

					node.Members.Add(me) // Add myself back since we clear the members list on a leave

					rpcID, _ := strconv.Atoi(node.Hostname[15:17])
					node.Neighbors.L.Lock()
					for i := 1; i < 11; i++ {
						err := node.AskForIntroduction(IntroPort, i) // Ask for introductions from all nodes again
						if err != nil {
							node.Logger.Print("Error asking for introduction ")
							node.Logger.Println(err)
							continue
						}
					}

					node.Active = true
					baseMsg := &Message{
						NodeID:   node.NodeID,
						Hostname: node.Hostname,
						UDPPort:  31337,
						Orig:     "JOIN," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname,
					}
					joinMsg := &JoinMessage{
						Message: baseMsg,
						TCPPort: 10000 + rpcID,
					}
					time.Sleep(500 * time.Millisecond) // Wait for member lists to come in before sending joins
					node.SendJoin(joinMsg)             // Send the join
					node.Neighbors.L.Unlock()
					node.Logger.Printf("Joining with ID: %d\n", node.NodeID)
				}
			}

		}
	}()
	return nil
}

// Stop kills the service
func (node *ThisNode) Stop() error {
	ctx, cancel := context.WithCancel(node.ctx)
	node.ctx = ctx
	cancel()
	return nil
}
