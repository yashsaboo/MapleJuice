package main

// server.go is the program that is run locally on all VMs and waits for client (client.go program) to connect
// and retrieve the log contents on each VM. Note that what the server does is that it serves the RPC
// by matching the log entries with the given target regex pattern one by one,
// and in turn return the matched results back to the client.
// The client then displays the results from all servers like a grep command

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LogQueryServer exported to be used as receiver in RPC process
type LogQueryServer int

// LogQueryRst is a struct which is return by the server RPC,
// storing the regex-matched line count and the matched log entries
// (all entries combined together as a string separated via \n)
type LogQueryRst struct {
	MatchedLineCnt int
	MatchedLines   string
}

// QueryLogRegex is the RPC function that is called by the remote client
// to query local log file via given regex pattern
// regexPattern	: regex pattern passed by the remote client to match the log entries
// queryRst		: pointer to LogQueryRst struct to store the RPC results
func (lqs *LogQueryServer) QueryLogRegex(regexPattern string, queryRst *LogQueryRst) error {

	fmt.Println("Serving log query request ...")
	fmt.Printf("Target regex:\t%s\n", regexPattern)

	// Init reply
	queryRst.MatchedLineCnt = 0
	queryRst.MatchedLines = ""

	// Prepare for command execution
	app := "grep"
	arg0 := "-n"
	arg1 := "-E"
	arg2 := regexPattern
	arg3 := os.Args[2]

	// Run command
	rst := exec.Command(app, arg0, arg1, arg2, arg3)
	stdout, err := rst.Output()

	// If err != nil, it can either be error or no matched lines
	if err != nil {
		fmt.Println("Error occurred when greping file")
		return nil
	}

	// Append filename to each entry
	// Too slow with direct concat (shame on you Go memory management), using stringBuilder
	var tmpStr strings.Builder
	for _, entry := range strings.Split(string(stdout), "\n") {
		if len(entry) != 0 {
			queryRst.MatchedLineCnt++
			tmpStr.WriteString(filepath.Base(os.Args[2]) + " line " + entry + "\n")
		}
	}

	// Convert string builder to string
	queryRst.MatchedLines = tmpStr.String()

	fmt.Println("Done.")

	return nil
}

// QueryLogRegexOld is the RPC function that is called by the remote client
// to query local log file via given regex pattern
// regexPattern	: regex pattern passed by the remote client to match the log entries
// queryRst		: pointer to LogQueryRst struct to store the RPC results
func (lqs *LogQueryServer) QueryLogRegexOld(regexPattern string, queryRst *LogQueryRst) error {

	fmt.Println("Serving log query request ...")

	// Compile Regex for matching
	fmt.Printf("Target regex:\t%s\n", regexPattern)
	r, err := regexp.Compile(regexPattern)
	if err != nil {
		fmt.Println("Error when compiling Regex")
		return err
	}

	// Open log file to read content
	fp, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Println("Error when opening log file")
		return err
	}
	defer fp.Close()

	// Read log file line by line using bufio.Reader with ReadString and run regex match
	// Count and store matched log entries
	lineCnt := 1
	reader := bufio.NewReader(fp)
	entry, err := reader.ReadString('\n')
	for err == nil {

		// Check if match regex or not, store if match
		// Note that filename (extracted from path) and line number is added to stored entry
		if r.MatchString(entry) {
			queryRst.MatchedLines += filepath.Base(os.Args[2]) + " line " + strconv.Itoa(lineCnt) + ": " + entry
			queryRst.MatchedLineCnt++
		}

		lineCnt++
		entry, err = reader.ReadString('\n')
	}

	// Deal with file reading errors
	if err != io.EOF {
		fmt.Println("Error when reading log file")
		return err
	}

	// In case there isn't a '\n' at the end of the log file
	// (ReadString gives EOF even if the last line of log file has content but is not terminated with '\n')
	// Perform matching and storing again
	if r.MatchString(entry) {
		queryRst.MatchedLines += filepath.Base(os.Args[2]) + " line " + strconv.Itoa(lineCnt) + ": " + entry + "\n"
		queryRst.MatchedLineCnt++
	}

	fmt.Println("Done.")

	return nil
}

func main() {

	// Check commandline arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run server.go LISTEN_PORT LOGFILE_PATH")
		return
	}

	// setting up server to receive RPC calls
	logQueryServer := new(LogQueryServer)
	err := rpc.Register(logQueryServer)
	if err != nil {
		fmt.Println("Error when registering RPC, aborting")
		return
	}

	// setting up server to receive RPC calls
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp4", ":"+os.Args[1])
	if err != nil {
		fmt.Println("Error when listening on port, aborting")
		return
	}

	// server start serving
	fmt.Println("Server start listening for queries ...")
	err = http.Serve(listener, nil)
	if err != nil {
		fmt.Println("Error when serving, aborting")
		return
	}
}
