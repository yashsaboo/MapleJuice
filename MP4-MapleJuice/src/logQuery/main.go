package logQuery

import (
	"bufio"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"

	"logQuery/rpc_export"
)

// NodeInfo Stores config information about nodes
type NodeInfo struct {
	Hostname string
	IP       string
	Port     string
}

type LogSearchReplyQueueItem struct {
	Reply      rpc_export.LogSearchReply
	Error      string
	ReplyCount int
}

func main() {

	// Load list of peers
	nodeList, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		log.Fatal("Error reading node file", err)
	}

	// Check for command line args
	if len(os.Args) < 2 {
		log.Fatal("Usage: main.go <command> [arguments]")
	}

	// Get the user command
	command := os.Args[1]

	// Run the appropriate command
	switch command {
	case "grep":
		err = DistributedGrep(os.Args[2:], nodeList, "./log/vm%d.log")
		if err != nil {
			log.Fatal("Error processing grep command: ", err)
		}
	default:
		log.Fatal("Error. Invalid Command \"" + command + "\"")
	}
}

// Run grep across distributed logs. Corresponds to the "grep" command.
func DistributedGrep(query []string, nodes []NodeInfo, location string) error {

	// Build the query object for RPC
	DistributedQuery := rpc_export.LogSearchQuery{
		strings.Join(query, " "), location}

	// Query every other node all at once
	wg := new(sync.WaitGroup)
	logReplyQueue := make(chan *LogSearchReplyQueueItem, len(nodes))
	for i := 0; i < len(nodes); i++ {
		wg.Add(1)
		go getRemoteLog(logReplyQueue, nodes[i], wg, DistributedQuery)
	}
	wg.Wait()

	// Slice to track the number of logs received from each Node
	reply_counts := []int{}
	for i := 0; i < len(nodes); i++ {
		reply_counts = append(reply_counts, 0)
	}

	replies := []*LogSearchReplyQueueItem{}

	// Print all replies from all nodes.
	for len(logReplyQueue) > 0 {
		reply := <-logReplyQueue

		replies = append(replies, reply)

		// Don't print anything until the summary for failed nodes
		if reply.Error != "" {
			continue
		}

		// Print each log
		for j := 0; j < len(reply.Reply.Logs); j++ {
			fmt.Printf("Node %d: %s\n", reply.Reply.GrepID, reply.Reply.Logs[j])
		}
	}

	// Print total count summary
	fmt.Println("\nDistributed Grep Summary:")
	for i := 0; i < len(replies); i++ {
		if replies[i].Error != "" {
			fmt.Println("Node " + strconv.Itoa(replies[i].Reply.GrepID) + " failed with message: " + replies[i].Error)
		} else {
			fmt.Println("Node " + strconv.Itoa(replies[i].Reply.GrepID) + " results count: " + strconv.Itoa(replies[i].ReplyCount))
		}
	}

	//Finished
	fmt.Println("Done")
	return nil
}

// Read the config file of nodes
func ReadNodeAddrFile(path string) ([]NodeInfo, error) {

	nodeList := []NodeInfo{}
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		curLine := strings.Split(scanner.Text(), " ")
		node := NodeInfo{Hostname: curLine[0], IP: curLine[1],
			Port: curLine[2]}
		nodeList = append(nodeList, node)
	}
	return nodeList, nil
}

// Log Fetch GoRoutine
func getRemoteLog(logReplyQueue chan *LogSearchReplyQueueItem, node NodeInfo, wg *sync.WaitGroup, query rpc_export.LogSearchQuery) {

	// Alert main thread when this function returns
	defer wg.Done()

	// Dial peer node
	path := node.IP + ":" + node.Port
	curClient, err := rpc.DialHTTP("tcp", path)
	if err != nil {
		nodeID, _ := strconv.Atoi(node.Port[3:])
		logReplyQueue <- &LogSearchReplyQueueItem{rpc_export.LogSearchReply{nodeID, nil}, "Connection Failure", -1}
		return
	}

	// Make RPC Log Search Call
	var response rpc_export.LogSearchReply
	err = curClient.Call("Node.LogSearch", &query, &response)
	if err != nil {
		nodeID, _ := strconv.Atoi(node.Port[3:])
		logReplyQueue <- &LogSearchReplyQueueItem{rpc_export.LogSearchReply{nodeID, nil}, "Remote Execution Failure: " + err.Error(), -1}
		return
	}

	if response.Logs == nil {
		response.Logs = []string{}
	}

	// Insert reply into thread-safe channel ignoring trailing newline from grep
	if len(response.Logs) == 0 {
		logReplyQueue <- &LogSearchReplyQueueItem{response, "", 0}
	} else {
		logReplyQueue <- &LogSearchReplyQueueItem{response, "", len(response.Logs) - 1}
	}
}
