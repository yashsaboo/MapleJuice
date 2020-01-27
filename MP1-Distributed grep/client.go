package main

// client.go is the program that is run locally on one VM and communicates with servers (server.go program)
// on all VMs to retrieve the matched log contents on each VM. Note that what the client does is that it initiates RPC
// on the remote servers and provides them with the target regex pattern, based on which the servers
// match the log entries one by one and return the matched results back to the client.
// The client then displays the results from all servers like a grep command

import (
	"fmt"
	"net/rpc"
	"os"
	"sync"
)

// LogQueryRst is a struct which is return by the server RPC,
// storing the regex-matched line count and the matched log entries
// (all entries combined together as a string separated via \n)
type LogQueryRst struct {
	MatchedLineCnt int
	MatchedLines   string
}

// callRPC is the function which performs RPC on give remote server
// addr					: address of remote server, in the format of "IP:PORT"
// serviceMethodName	: the "name" of the procedure on the remote server to be called
// rpcReply				: pointer to LogQueryRst struct to store the RPC results
// wg					: pointer to the waitgroup, used for the main goroutine to wait
//						  until all goroutines running callRPC are done
func callRPC(addr string, serviceMethodName string, regexPattern string, rpcReply *LogQueryRst, wg *sync.WaitGroup) error {
	// Decrement waitgroup counter so that when all callRPC goroutines are done the main goroutine can continue
	defer wg.Done()

	// Dial to servers
	fmt.Fprintf(os.Stderr, "Dialing %s ...\n", addr)
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when dialing %s\n", addr)

		// If error occurred when performing RPC,
		// store empty LogQueryRst struct with MatchedLineCnt being -1
		// to notify the main goroutine of such event
		*rpcReply = LogQueryRst{}
		rpcReply.MatchedLineCnt = -1
		return err
	}

	// Call server RPC and store results
	fmt.Fprintf(os.Stderr, "Calling %s on server at %s ... \n", serviceMethodName, addr)
	err = client.Call(serviceMethodName, regexPattern, rpcReply)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling %s on server at %s\n", serviceMethodName, addr)

		// Error
		*rpcReply = LogQueryRst{}
		rpcReply.MatchedLineCnt = -1
		return err
	}

	return nil
}

func main() {

	// Check commandline arguments
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run client.go REGEX_PATTERN")
		return
	}

	// The VM IPv4 addresses that the client will connect to
	// Using port 5566
	serverAddresses := []string{
		"172.22.152.146:5566",
		"172.22.154.142:5566",
		"172.22.156.142:5566",
		"172.22.152.147:5566",
		"172.22.154.143:5566",
		"172.22.156.143:5566",
		"172.22.152.148:5566",
		"172.22.154.144:5566",
		"172.22.156.144:5566",
		"172.22.152.149:5566"}

	// For server on each VMs initiate RPC
	// To speed up, use concurrent execution via goroutines
	// Waitgroup works by first when a goroutine is initiated, increment 1 to wg
	// When the goroutine is finished, in the goroutine function decrement 1
	// The main routine that waits on wg will hang when wg count > 0
	// and will end waiting when wg count is decremented back to 0
	wg := new(sync.WaitGroup)
	wg.Add(len(serverAddresses))        // Increment the number of goroutines that will initiated for main goroutine to wait for them
	grepLogResults := [10]LogQueryRst{} // Set up space for the goroutines to store the RPC results
	for indx, addr := range serverAddresses {
		go callRPC(addr, "LogQueryServer.QueryLogRegex", os.Args[1], &(grepLogResults[indx]), wg)
	}

	// Wait for all callRPC goroutines to finish
	wg.Wait()

	// Print all logs of RPC results stored in grepLogResults
	// Note that MatchedLineCnt -1 means RPC failed, therefore no need to print log
	for _, rpcReply := range grepLogResults[:len(serverAddresses)] {
		if rpcReply.MatchedLineCnt != -1 {
			fmt.Print(rpcReply.MatchedLines)
		}
	}

	// Print line counts for all RPC results
	// Note that failed RPC result are also printed
	totalMatchedLineCount := 0
	for indx, rpcReply := range grepLogResults[:len(serverAddresses)] {
		if rpcReply.MatchedLineCnt != -1 {
			fmt.Printf("Machine %d Matched Line Count:\t%d\n", (indx + 1), rpcReply.MatchedLineCnt)
			totalMatchedLineCount += rpcReply.MatchedLineCnt
		} else {
			fmt.Printf("Warning: Failure to retrieve log file content from Machine %d\n", (indx + 1))
		}
	}

	// Print total line cnt as required
	fmt.Printf("Total Matched Line Count:\t%d\n", totalMatchedLineCount)
}
