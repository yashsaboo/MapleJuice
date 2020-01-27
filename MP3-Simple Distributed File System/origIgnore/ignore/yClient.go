package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

type InPacketStruct struct {
	PacketType string //client_request, server_request (who is sending this packet)
	Command    string //put, ls, read, write
	FileName   string //name of the file on the server
}

const replicaMetadataLocalFilePath = ""
const operationsDatalogLocalFilePath = ""

var serverIPs = [1]string{"1.1.1.1"} //TODO add IPs of all the machines. Doubt, how will you select randomly a IP for asking who is master?

func encodeArgumentsToJSONString(operation string, sdfsfilename string) string {
	JSONStringForArguments := &InPacketStruct{
		PacketType: "client_request",
		Command:    operation,
		FileName:   sdfsfilename,
	}
	data, _ := json.Marshal(JSONStringForArguments)
	return string(data)
}

func getMastersIP() string { //sync

	conn, _ := net.Dial("tcp", serverIPs[0])                                   //make a connection to the one of the IPS. NEED to CHANGE this
	JSONStringForArguments := encodeArgumentsToJSONString("who_is_master", "") //Command and FileName are irrelevant here so sending empty string
	fmt.Fprintf(conn, JSONStringForArguments+"\n")                             // send to socket
	masterIP, _ := bufio.NewReader(conn).ReadString('\n')                      //get the message back from server
	return masterIP
}

// func getMembershipList() map[int]*membership.MemberNode {
// 	// TODO Query the server or ask the server to store the memebership list to some local file periodiacally, and access it directly
// 	return memberMap;
// }

// //TODO
// func decodeJSONtoMembershipMemberNode(listOfReplicasInJSON string) ([]membership.MemberNode, bool) {
// 	//Initialise the return values
// 	var listOfReplicas [4]string
// 	quorumTestPassed = false

// 	// TODO: Make a Format for JSON which sends out replica information. also it should containt, if it passed the quorum test. If it doesn't pass the quorum test then listOfReplicasInJSON=nil
// 	json.Unmarshal([]byte(birdJson), &birds)

// 	return listOfReplicas, quorumTestPassed
// }

// //TODO
// func decodeJSONtoOperationSuccess(operationSuccessStatusInJSON string) bool {
// 	var operationSuccessStatus bool
// 	json.Unmarshal([]byte(operationSuccessStatusInJSON), &operationSuccessStatus)
// 	return operationSuccessStatus
// }

func callMaster(masterNodeIP string, JSONStringForArguments string) string {
	conn, _ := net.Dial("tcp", masterNodeIP)                                    //make a connection to the local server
	fmt.Fprintf(conn, JSONStringForArguments+"\n")                              // send to socket
	returnReplicaIDsInStringFormat, _ := bufio.NewReader(conn).ReadString('\n') //get the message back from server
	return returnReplicaIDsInStringFormat
}

func callServer(replicaNodeIP string, JSONStringForArguments string) string {
	conn, _ := net.Dial("tcp", replicaNodeIP)                           //make a connection to the local server
	fmt.Fprintf(conn, JSONStringForArguments+"\n")                      // send to socket
	operationSuccessStatus, _ := bufio.NewReader(conn).ReadString('\n') //get the message back from server
	return operationSuccessStatus
}

func readLines(path string) ([]string, error) { //stolen from stackoverflow :)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLines(lines []string, path string) error { //stolen from stackoverflow :)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

// Main thread of execution
func main() {

	fmt.Println(len(os.Args))
	fmt.Println(os.Args[0])

	// Check commandline arguments
	if len(os.Args) != 2 && len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Println("Usage: go run yClient.go operation [sdfsfilename] [localfilename]")
		return
	}

	fmt.Println("Got the args")

	// Get Membership List
	masterIP := getMastersIP()

	switch os.Args[1] {

	case "who_is_master":
		fmt.Println("In who_is_master")
		break

	case "put":
		JSONStringForArguments := encodeArgumentsToJSONString(os.Args[1], os.Args[2])
		listOfReplicasInStringFormat := callMaster(masterIP, JSONStringForArguments)
		if listOfReplicasInStringFormat == "Confirm" {
			fmt.Println("This file was updated in last one minute by another node. Are you sure that you want to put into this file? Please type 'yes' in next 30 seonds [This is to prevent write-write conflict]")
			// read in input from stdin
			reader := bufio.NewReader(os.Stdin)
			confirmStatus, _ := reader.ReadString('\n')
			if confirmStatus == "yes" {
				listOfReplicasInStringFormat = callMaster(masterIP, confirmStatus)
			}
		}
		if listOfReplicasInStringFormat == "No" { //If it was "No", then Quorum test would have had failed
			fmt.Println("Couldn't perform 'put' operation. Master returned 'No'")
			break
		}
		listOfReplicas := strings.Split(listOfReplicasInStringFormat, " ; ") //Format: "replica1 ; replica2 ; replica3 ; replica4"
		for _, replica := range listOfReplicas {

			//First send the InPacketStruct structure as JSON, telling the server replica, that I need to perform 'put' operation, and also this struct has intended file name too 'sdfsfilename'
			operationSuccessStatus := callServer(replica, JSONStringForArguments)
			fmt.Println(operationSuccessStatus)

			//If server replica returns "ok", then run a for loop and send the file contents line by line.
			if operationSuccessStatus == "ok" {
				lines, err := readLines(os.Args[3]) // read localfilename
				if err != nil {
					fmt.Println("File Not Present")
				}

				// connect to this socket AGAINNNN. First the socket is connected to let the server know that it wants to perform put. Second connection to perform file transfer
				conn, err := net.Dial("tcp", replica)

				//Read file and send it to server
				for i, line := range lines {
					fmt.Fprintf(conn, string(line)+"\n")
					fmt.Println(i)
				}
			}
		}
		break

	case "get":
		JSONStringForArguments := encodeArgumentsToJSONString(os.Args[1], os.Args[2])
		singleReplicaIDInStringFormat := callMaster(masterIP, JSONStringForArguments)
		if singleReplicaIDInStringFormat == "No" { //If it was "No", then Quorum test would have had failed
			fmt.Println("Couldn't perform 'get' operation. Master returned 'No'")
			break
		}
		// listen on all interfaces
		ln, _ := net.Listen("tcp", ":8082") //CHANGED the port number for client

		// accept connection on port
		conn, _ := ln.Accept()

		var lines []string

		// run loop forever (or until ctrl-c)
		for {
			// will listen for message to process ending in newline (\n)
			message, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				break
			}
			// fmt.Println("Got a line")
			lines = append(lines, message)
		}
		writeLines(lines, os.Args[2])
		fmt.Println("Got the file successfully")
		break

	case "delete":
		JSONStringForArguments := encodeArgumentsToJSONString(os.Args[1], os.Args[2])
		_ = callMaster(masterIP, JSONStringForArguments)
		break

	case "ls":
		JSONStringForArguments := encodeArgumentsToJSONString(os.Args[1], os.Args[2])
		listOfReplicasInStringFormat := callMaster(masterIP, JSONStringForArguments)
		if listOfReplicasInStringFormat == "No" {
			fmt.Println("Couldn't perform 'ls' operation. Master returned 'No'")
			break
		}
		listOfReplicas := strings.Split(listOfReplicasInStringFormat, " ; ") //Format: "replica1 ; replica2 ; replica3 ; replica4"
		for _, replica := range listOfReplicas {                             //print the replicas id's where the os.Args[4] file is currently stored
			fmt.Println(replica)
		}
		break
	}
}
