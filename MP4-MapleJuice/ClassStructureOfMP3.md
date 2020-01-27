Class Structure of MP3:

# src/logQuery
## main.go
> NodeInfo struct {	Hostname string; IP string; Port string } --> NodeInfo Stores config information about nodes
> LogSearchReplyQueueItem struct { Reply rpc_export.LogSearchReply; Error string; ReplyCount int }

func main()
```
    Load list of peers
    Usage: main.go <command> [arguments]
    switch command {
    case "grep":
        err = DistributedGrep(os.Args[2:], nodeList, "./log/vm%d.log")
    default:
         Invalid Command
    }
```

func DistributedGrep(query []string, nodes []NodeInfo, location string) error --> Run grep across distributed logs. Corresponds to the "grep" command.
```
	Build the query object for RPC
	Query every other node all at once: go getRemoteLog(logReplyQueue, nodes[i], wg, DistributedQuery)
	Slice to track the number of logs received from each Node
	Print all replies from all nodes.
	Print total count summary
```

func ReadNodeAddrFile(path string) ([]NodeInfo, error) --> Read the config file of nodes

func getRemoteLog(logReplyQueue chan *LogSearchReplyQueueItem, node NodeInfo, wg *sync.WaitGroup, query rpc_export.LogSearchQuery) --> Log Fetch GoRoutine
```
	Alert main thread when this function returns
	Dial peer node
	Make RPC Log Search Call
	Insert reply into thread-safe channel ignoring trailing newline from grep
```


## nodes.txt

## RunGrep_test.go
> Testing MP1 - Might Require it since MP4 requires MP1 usage

## server_test.go
> For testing: sets up a local RPC server to test with - Might Require it since MP4 requires MP1 usage

## src/logQuery/rpc_export
### kill.go
func Kill(args *KillCommand, reply *KillResponse) error
> 	Kills channel and sends it to main node thread

### log_search.go
func (*Node) LogSearch(args *LogSearchQuery, reply *LogSearchReply) error
> 	Runs grep command MP1: Executes a Log Search.

### rpc_export.go
func SetKill(k *chan bool)
> 	Sends the kill signla through channel

func SetLogger(l *log.Logger)
> 	Logs it

func SetNodeID(i int)
> 	Sets GrepID

### structs.go
> LogSearchQuery struct {	GrepArgs string; Location string} --> The RPC Query Object for Log Queries. GrepArgs should be a list of arguments to grep like -n or -Po or "foo.*"
LogSearchReply struct {	GrepID int;	Logs []string} --> This is the reply to an RPC Log Query
type KillCommand struct { K bool } --> Send one of these to rpc.Kill to kill a node
type KillResponse struct { K bool } --> Reply to a kill command

# src/membership
## membership.go
> UDPPort for all nodes:	var UDPPort = 31337
> IntroPort is used for introduction:	var IntroPort = 33333
> HashRingSize is how big our hashring is:	const HashRingSize = 4294967296

func StartNode(myNode *node.ThisNode) --> StartNode starts the two listeners and handles membership messages
```
	Introducer loop with boilerplate code for implementing a UDP listener in Go
		baseMsg := &node.Message{
				NodeID:     incNodeID,
				T:          uint64(curTime.UnixNano() / 1000000),
				RemoteAddr: returnAddr,
				Hostname:   messageFields[2],
				UDPPort:    UDPPort,
				Orig:       message,
			}
		switch messageType {
				case "INTRO":node.IntroMessage
							myNode.HandleIntro(introMsg, returnAddr)
				case "MEMBER":myNode.HandleIntroList(message)
				case "FILELIST":myNode.HandleIntroFileList(message)
			}
	Spawn UDP listener thread for all other messages
		baseMsg := &node.Message{
				NodeID:     incNodeID,
				T:          uint64(curTime.UnixNano() / 1000000),
				RemoteAddr: returnAddr,
				Hostname:   messageFields[2],
				UDPPort:    UDPPort,
				Orig:       message,
			}
		switch messageType { // Messages are decoded and processed according to type
			case "INTRO": introMsg := &node.IntroMessage
				fmt.Println("Got an INTRO on the wrong port!")
			case "HEART":
				heartMsg := &node.HeartMessage{
			case "JOIN":
				tcpPort, _ := strconv.Atoi(messageFields[3])
				joinMsg := &node.JoinMessage
			case "LEAVE":
				leaveMsg := &node.LeaveMessage
			case "FAIL":
				failMsg := &node.FailMessage
			case "NEED":
				requestID, _ := strconv.ParseUint(messageFields[5], 10, 64)
				myNode.HandleNeed(messageFields[3], requestID, messageFields[2])
			case "DELETE":
				myNode.HandleDelete(messageFields[3])
			case "HAVE":
				timestamp, _ := strconv.ParseInt(messageFields[5], 10, 64)
				requestID, _ := strconv.ParseUint(messageFields[4], 10, 64)
				hostname := messageFields[2]
				filename := messageFields[3]
				myNode.HandleHave(hostname, filename, timestamp, requestID)
			case "NEWFILE":
				newTime, _ := strconv.ParseInt(messageFields[4], 10, 64)
				newfileMsg := &node.FileMessage{
					Message:  baseMsg,
					SDFSName: messageFields[3],
					Hash:     messageFields[5],
					Updated:  newTime,
				}
				fmt.Println(time.Now().Format("15:04:05.000"))
				myNode.HandleNewFile(newfileMsg)
			default:
				myNode.Logger.Printf("Unknown message type: " + messageType + " was ignored")
			}
		}
```


## node
### delete.go
func (node *ThisNode) AnnounceDelete(name string) error --> AnnounceDelete accounces that a file is deleted
```
    Construct the delete message
    Send Message to all other nodes including yourself using UDP
```
    
func (node *ThisNode) HandleDelete(name string) error --> HandleDelete handles a delete message that someone sent us
```
    exec.Command("rm", "/shared/"+name).Output() // Try to delete.
    Delete from global file list
    for i, file := range node.Files.Files {
		if file.SDFSName == name {
			// Remove file from global list
			node.Files.L.Lock()
			node.Files.Files = append(node.Files.Files[:i], node.Files.Files[i+1:]...)
			node.Files.L.Unlock()
		}
	}
```

func (node *ThisNode) DeleteAllFiles() error --> DeleteAllFiles deletes all files in the /shared directory upon startup/rejoin
```
    directory := "/shared/"
    read, err := os.Open(directory)
    files, err := read.Readdir(0)
    for idx := range files {
		curFile := files[idx]
		path := directory + curFile.Name()
		err = os.Remove(path)
    }
```

### failures.go
**Contains functions to check for failures, send failure messages, and handle incoming failure messages**

func (node *ThisNode) CheckForFailures(timoutSeconds uint64) error
> CheckForFailures checks to see if any of our neighbors have failed
    
func (node *ThisNode) SendFailure(msg *FailMessage) error
> SendFailure sends a failure message to our neighbors
    
func (node *ThisNode) HandleFailure(msg *FailMessage) error
> HandleFailure processes failure messages from other nodes

### get.go

func (node *ThisNode) GetFile(name, localPath string)
> GetFile will get a file

func (node *ThisNode) SendNeed(host, name string, requestID uint64) error
> SendNeed tells another nods that we need a file

func (node *ThisNode) HandleNeed(name string, requestID uint64, neederHost string) error
> HandleNeed handles a need message

func (node *ThisNode) HandleHave(hostname, name string, timestamp int64, requestID uint64)
> HandleHave handles a have message

### heartbeat.go
**This contains functions to send and handle heartbeats**

func (node *ThisNode) SendHeartbeats() error
> SendHeartbeats sends heartbeats to our neighbors

func (node *ThisNode) HandleHeartbeat(msg *HeartMessage) error
> HandleHeartbeat processes incoming heartbeats to our node

### introduction.go
**This contains functions for handling the introduction process. Every node in the network can be an introducer.**

func (node *ThisNode) AskForIntroduction(IntroducerPort int, idx int) error
> AskForIntroduction gets the membership list from the introducer

func (node *ThisNode) HandleIntroList(msg string) error
> HandleIntroList processes introductions if someone asked this node for an introduction

func (node *ThisNode) HandleIntro(msg *IntroMessage, retAddr *net.UDPAddr) error
> HandleIntro processes introductions if this node is the introducer

func (node *ThisNode) HandleIntroFileList(msg string) error
> HandleIntroFileList processes an incoming file list

### join.go
**This contains functions to handle and send join messages**

func (node *ThisNode) HandleJoinMsg(msg *JoinMessage) error
> HandleJoinMsg processes new joins from other nodes

func (node *ThisNode) SendJoin(msg *JoinMessage) error 
> SendJoin sends a join request to our neighbors

### leave.go

**This contains the functions for sending and handling leave messages**

func (node *ThisNode) HandleLeave(msg *LeaveMessage) error
> HandleLeave processes leave messages from other nodes

func (node *ThisNode) SendLeave(msg *LeaveMessage) error
> SendLeave sends a leave message to our neighbors

### messagecache.go

func (r *RecentMessageCache) FlushMessageCache(MaxAgeSeconds uint64) error
> FlushMessageCache will remove all messages older than the max age from the cache

func (r *RecentMessageCache) Add(message string) error
> Add will add a new message to the recent message cache

func (r *RecentMessageCache) Contains(message string) bool 
> Contains will check if a message is in the recent message cache

### messages.go
**Contains data structurs for each message format. They are all based on a common Message format.**

> Message struct
> IntroMessage struct
> MemberMessage struct
> HeartMessage struct
> LeaveMessage struct
> FailMessage struct
> FileMessage struct

### neighbors.go
**This file contains the data structures and helper functions for maintaining the neighbor and membership lists**

// MembershipList struct with lock
> type MembershipList struct {
> 	L       *sync.Mutex
> 	Members []OtherNode
> }

// NeighborList struct with lock
> type NeighborList struct {
> 	L         *sync.Mutex
> 	Neighbors []OtherNode
> }

func (m *MembershipList) Contains(id uint64) int
> Contains will check if a node is in the list. If it is, it will return the index. If it isn't, return -1
 
func (m *MembershipList) Add(node OtherNode) error
> Add will add a new node to the list if it isn't already in the list

func (m *MembershipList) Remove(id uint64) error
> Remove removes a node from the list if it exists

func (m *MembershipList) Sort()
> Sort will sort the membership list by NodeID

func (n *NeighborList) Update(m *MembershipList, id uint64) error
> Update goes through the membership list and finds our new neighbors

func mod(a int, n int) int 
> mod performs the % (modulo) operation since Go does not have the implementation we want
 
### node.go
**This contains all the members and functions that define a node**

InitNode(conn io.Reader) (*ThisNode, error)
> InitNode will setup this node

    Define Input Flags
    Parse Input Flags
    Generate our own NodeID randomly
    Open log file
    Create Logger with format [DATE] [TIME] ???.go line ?? (Node ?) message
    Resolve our own UDP address (ip:port)
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
	}

func (node *ThisNode) Start(ctx context.Context) error 
> Start will start the node and all its services

```
Spawn Heartbeat Thread
Spawn Recent Message Cache Flusher Thread
Spawn thread for stabilization
Spawn FailureDectection Thread
Get User Input
Main node controller loop
    case <-node.ctx.Done():
		node.Logger.Printf("@ Node Terminated Normally @")
		os.Exit(0)
    case text := <-uiChan:
        strings.Contains(text, "grep")
        strings.Contains(text, "neighbor")
        strings.Contains(text, "member")
        strings.Contains(text, "put")
            Usage: put <path_to_local_file> <SDFS_file_name>
        strings.Contains(text, "get")
            Usage: get <SDFS_fie_name> <local_output_path>
        strings.Contains(text, "delete")
            Usage: delete <SDFS_file_name>
        strings.Contains(text, "store")
        strings.Contains(text, "ls")
        strings.Contains(text, "leave")
        strings.Contains(text, "id")
        strings.Contains(text, "join")
```
func (node *ThisNode) Stop() error
> Stop kills the service

### put.go
func (node *ThisNode) PutFile(Source, SDFSName string, confirmChan chan bool) error
> PutFile will add a new file to the system

func (node *ThisNode) AnnounceNewFile(name string, timestamp int64, hash uint64) error
> AnnounceNewFile after we know other nodes have it

func (node *ThisNode) HandleNewFile(msg *FileMessage)
> HandleNewFile appends a new file to our file list


### rpc_part.go
// FileVersionAsk contains the filename
> type FileVersionAsk struct {
> 	Filename string
> 	// Node     *ThisNode
> }

// FileVersionAnswer says whether we have the file and what the timestamp is
> type FileVersionAnswer struct {
> 	Have      bool
> 	Timestamp int64
> }

func (node *ThisNode) GetFileVersion(other OtherNode, filename string) (int64, bool)
GetFileVersion gets the file version from another node

func (*RPCNode) RPCGetFileVersion(args *FileVersionAsk, reply *FileVersionAnswer) error
RPCGetFileVersion processes an RPC call asking for the file version

### rsync.go
func (node *ThisNode) RSyncSend(src, SDFSName string, dst OtherNode, ackChan chan bool)
> RSyncSend sends a file somewhere else

func (node *ThisNode) RSyncFetch(filename, localPath, src string) 
> RSyncFetch fetches a file

### stabilization.go
func (node *ThisNode) CheckFileStabilization() 
> CheckFileStabilization periodically checks that we have the right files and that they are current

func (node *ThisNode) ListFileLocations(filename string) (locations []OtherNode)
> ListFileLocations lists locations where a file is stored

### types.go
// RPCNode is used for rpc calls
> type RPCNode int

// OtherNode contains info on other nodes in the network
> type OtherNode struct 

// ThisNode contains info on our node
type ThisNode struct 

// RecentMessageCache with list and lock
type RecentMessageCache struct 

// FileEntry holds data for a file we know about
> type FileEntry struct

// GlobalFileList holds data for a collection of files
> type GlobalFileList struct 

// NeedFileResponses holds the responses from nodes when we have requested a file
// We wait until we have received a majority until doing the transfer from the most recent one
> type NeedFileResponses struct

// RecentMessage format
> type RecentMessage struct 

// PastHaveMessage collected here
> type PastHaveMessage struct 

// OngoingNeedLabel labels each ongoing need with a timeout
> type OngoingNeedLabel struct

// OngoingNeeds holds ongoing Needs
> type OngoingNeeds struct

// WriteVerification holds data for confirming write-write conflicts
> type WriteVerification struct

### utils.go

    // HashRingSize is how big our hashring is
    const HashRingSize = 4294967296

    // LogLocation is where the log is located
    const LogLocation = "./out/machine.%d.log"

    // HeartbeatFrequencyMilliseconds is how often we send heartbeats
    const HeartbeatFrequencyMilliseconds = 1000

    // MessageCacheFlushIntervalSeconds is how often to flush the message cache
    const MessageCacheFlushIntervalSeconds = 60

    // MessageCacheMaxAge is how long we keep messages
    const MessageCacheMaxAge = 60

    // FailureTimeoutSeconds is how long we wait until a node fails
    const FailureTimeoutSeconds = 3

    // FileListRefreshSeconds is how long we wait to refresh the file list
    const FileListRefreshSeconds = 5

    // FailureCheckFrequencyMilliseconds is often we check for failures
    const FailureCheckFrequencyMilliseconds = 1000

    // UDPPort is the port our protocol uses for membership
    var UDPPort = 31337

    // IntroPort is used for introduction
    var IntroPort = 33333
    
func (node *ThisNode) GetResponsibleNodes(filename string) []OtherNode
> GetResponsibleNodes finds nodes that are responsible for this file by name

func (node *ThisNode) GetResponsibleNodesByHash(targetID uint64) []OtherNode 
> GetResponsibleNodesByHash finds responsible nodes by hash

func (node *ThisNode) GetQuorumSize() int
> GetQuorumSize calculates the size of the quorum for our current network

func (node *ThisNode) ListLocalFiles() (answer []FileEntry)
> ListLocalFiles lists local files we have

func (node *ThisNode) IsConflicted(filename string) bool
> IsConflicted determines if there was a previous write within 60 seconds

