package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type InPacketStruct struct {
	PacketType string //client_request, server_request (who is sending this packet)
	Command    string //put, ls, read, write
	FileName   string //name of the file on the server
}

const replicaMetadataLocalFilePath = ""
const operationsDatalogLocalFilePath = ""

const PORTForServer = ":8090"

var serverIPs = []string{
	"172.22.156.228",
	"172.22.152.233",
	"172.22.154.229",
	"172.22.156.229",
	"172.22.152.234",
	"172.22.154.230",
	"172.22.156.230",
	"172.22.152.235",
	"172.22.154.231",
	"172.22.156.231"} 
// var serverIPs = [1]string{"127.0.0.1"} //For debugging on local machine

func GetMyIP() string {
	var myip string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("Cannot get my IP")
		os.Exit(1)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				myip = ipnet.IP.String()
			}
		}
	}
	return myip
}

func encodeArgumentsToJSONString(operation string, sdfsfilename string) string {
	JSONStringForArguments := &InPacketStruct{
		PacketType: "client_request",
		Command:    operation,
		FileName:   sdfsfilename,
	}
	data, _ := json.Marshal(JSONStringForArguments)
	return string(data)
}

func encodePacket(packetType string, command string, fileName string) string{
	return "{\"PacketType\":\""+packetType+"\",\"Command\":\""+command+"\",\"FileName\":\""+fileName+"\"}"
}

func getMastersIP() string { //sync
	for _, val := range(serverIPs) {
		//fmt.Println("Trying to connect to ", val)
		conn, err := net.Dial("tcp", val+PORTForServer)
		if err == nil { //connection established!
			outMessage := encodePacket("client_request", "who_is_master", "")
			fmt.Fprintf(conn, outMessage + "\n")
			message, err2 := bufio.NewReader(conn).ReadString('\n')
			if err2 == nil {
				return strings.TrimSuffix(message, "\n")
			}
		}
	}
	return "127.0.0.1"
}

func callMaster(masterNodeIP string, JSONStringForArguments string) string {
	fmt.Print("callMaster() masterNodeIP:" + masterNodeIP + PORTForServer)
	conn, err := net.Dial("tcp", masterNodeIP+PORTForServer) //make a connection to the local server
	fmt.Print("callMaster() err:")
	fmt.Println(err)
	fmt.Fprintf(conn, JSONStringForArguments+"\n")                              // send to socket
	returnReplicaIDsInStringFormat, _ := bufio.NewReader(conn).ReadString('\n') //get the message back from server
	fmt.Println("callMaster() returnReplicaIDsInStringFormat:" + returnReplicaIDsInStringFormat)
	if returnReplicaIDsInStringFormat == "confirm\n" { //for write-write conflict
		// read in input from stdin
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("The update for this file took place in last one minute. Are you sure you want edit it again? yes/no: ")
		text, _ := reader.ReadString('\n')
		// send to socket
		fmt.Fprintf(conn, text+"\n")
		// listen for reply
		returnReplicaIDsInStringFormat, _ = bufio.NewReader(conn).ReadString('\n')
	}
	return returnReplicaIDsInStringFormat
}

func makeTimestamp() int64 {
    return time.Now().UnixNano() / int64(time.Millisecond)
}

func callServerForPut(replicaNodeIP string, JSONStringForArguments string) {
	fmt.Println("callServer() IP:" + replicaNodeIP + PORTForServer)
	conn, err := net.Dial("tcp", replicaNodeIP+PORTForServer) //make a connection to the local server
	fmt.Print("callServer() err:")
	fmt.Println(err)
	fmt.Fprintf(conn, JSONStringForArguments+"\n")                      // send to socket
	operationSuccessStatus, _ := bufio.NewReader(conn).ReadString('\n') //get the message back from server
	operationSuccessStatus = strings.TrimSuffix(operationSuccessStatus, "\n")
	operationSuccessStatus = strings.TrimSuffix(operationSuccessStatus, " ")
	fmt.Println("main() operationSuccessStatus: " + operationSuccessStatus)

	//If server replica returns "ok", then run a for loop and send the file contents line by line.
	if operationSuccessStatus == "ok" {
		lines, err := readLines(os.Args[3]) // read localfilename
		if err != nil {
			fmt.Println("File Not Present")
		}
		// connect to this socket AGAINNNN. First the socket is connected to let the server know that it wants to perform put. Second connection to perform file transfer
		// conn, err := net.Dial("tcp", replica+PORTForServer)
		fmt.Println("(CLIENT)Reading File")
		//Read file and send it to server
		timestamp := makeTimestamp() //Dummy, initialise it with the file's timestamp in the metadata

		//2. If less than 1 minute, ask for confirmation
		for _, line := range lines {
			fmt.Fprintf(conn, string(line)+"\n")
			//fmt.Println(i)
		}
		currentTime := makeTimestamp()
		difference := currentTime - timestamp
		fmt.Println("(CLIENT) TIME TO UPLOAD(miliseconds) = ", difference)
	}
	conn.Close()
}

func callServerForGet(replicaNodeIP string, JSONStringForArguments string) {
	fmt.Println("callServerForGet() IP:" + replicaNodeIP + PORTForServer)
	conn, err := net.Dial("tcp", replicaNodeIP+PORTForServer) //make a connection to the local server
	fmt.Print("callServerForGet() err:")
	fmt.Println(err)
	fmt.Fprintf(conn, JSONStringForArguments+"\n")                      // send to socket
	operationSuccessStatus, _ := bufio.NewReader(conn).ReadString('\n') //get the message back from server
	operationSuccessStatus = strings.TrimSuffix(operationSuccessStatus, "\n")
	operationSuccessStatus = strings.TrimSuffix(operationSuccessStatus, " ")
	fmt.Println("callServerForGet() operationSuccessStatus: " + operationSuccessStatus)

	//If server replica returns "ok", then run a for loop and send the file contents line by line.
	if operationSuccessStatus == "ok" {
		var lineSlice []string

		// // listen on all interfaces
		// ln, _ := net.Listen("tcp", ":8082") //CHANGED the port number for client
		// // accept connection on port
		// conn, _ := ln.Accept()

		// run loop forever (or until ctrl-c)
		for {
			// fmt.Println("callServerForGet() I'm here 1")
			// will listen for message to process ending in newline (\n)
			message, err2 := bufio.NewReader(conn).ReadString('\n')
			if err2 != nil {
				break
			}
			// fmt.Println(message)
			message = strings.TrimSuffix(message, "\n")
			lineSlice = append(lineSlice, message)
		}
		fmt.Println("callServerForGet() I'm here 5")

		err := writeLines(lineSlice, "data/"+os.Args[3])
		if err != nil {
			fmt.Println("(CLIENT)Could not write to file" + os.Args[3])
		}
	}
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

	//Handle slashes in file name
	if len(os.Args) == 3 {
		os.Args[2] = strings.ReplaceAll(os.Args[2], "/", "_")
	}
	if len(os.Args) == 4 {
		os.Args[3] = strings.ReplaceAll(os.Args[3], "/", "_")
	}
	// fmt.Println(os.Args[3]) //Debugging

	//Load the listOfServersIP with your own IP, since your IP will also have server running
	//serverIPs = append(serverIPs, GetMyIP()) //comment it if debugging
	// fmt.Println(serverIPs[0]) //for debugging

	// Get Membership List
	masterIP := getMastersIP()
	//masterIP = strings.TrimSuffix(masterIP, "\n") //Have to do this because the IP returned by Server has \n newline feed in the end, so when we attach a port, it's not able to render
	masterIP = strings.TrimSuffix(masterIP, " ")
	fmt.Println("MasterIP: " + masterIP)

	switch os.Args[1] {

	case "put":
		JSONStringForArgumentsForMaster := encodeArgumentsToJSONString(os.Args[1]+"ForMaster", os.Args[2])
		listOfReplicasInStringFormat := callMaster(masterIP, JSONStringForArgumentsForMaster)
		listOfReplicasInStringFormat = strings.TrimSuffix(listOfReplicasInStringFormat, "\n")
		listOfReplicasInStringFormat = strings.TrimSuffix(listOfReplicasInStringFormat, " ")
		fmt.Println("main() listOfReplicasInStringFormat: " + listOfReplicasInStringFormat + "Hi")
		if listOfReplicasInStringFormat == "Confirm" {
			fmt.Println("This file was updated in last one minute by another node. Are you sure that you want to put into this file? Please type 'yes' in next 30 seonds [This is to prevent write-write conflict]")
			// read in input from stdin
			reader := bufio.NewReader(os.Stdin)
			confirmStatus, _ := reader.ReadString('\n')
			if confirmStatus == "yes" {
				listOfReplicasInStringFormat = callMaster(masterIP, confirmStatus)
			}
		}
		fmt.Println("listOfReplicasInStringFormatIsALongVariableName [", listOfReplicasInStringFormat, "]")

		if listOfReplicasInStringFormat == "No" || listOfReplicasInStringFormat == "invalid ip ; invalid ip ; invalid ip ; invalid ip ;" { //If it was "No", then Quorum test would have had failed'
			fmt.Println("Couldn't perform 'put' operation. Master returned 'No'")
			break //Uncomment it if not debugging
		}
		listOfReplicas := strings.Split(listOfReplicasInStringFormat, " ; ") //Format: "replica1 ; replica2 ; replica3 ; replica4"
		for _, replica := range listOfReplicas {

			fmt.Println("main() replica: " + replica)
			//First send the InPacketStruct structure as JSON, telling the server replica, that I need to perform 'put' operation, and also this struct has intended file name too 'sdfsfilename'
			JSONStringForArgumentsForServer := encodeArgumentsToJSONString(os.Args[1]+"ForServer", os.Args[2])
			callServerForPut(replica, JSONStringForArgumentsForServer) //Uncomment it if not debugging
			// callServerForPut(masterIP, JSONStringForArgumentsForServer) //Comment it if not debugging
		}

		break

	case "get":
		JSONStringForArgumentsForMaster := encodeArgumentsToJSONString(os.Args[1]+"ForMaster", os.Args[2])
		singleReplicaIDInStringFormat := callMaster(masterIP, JSONStringForArgumentsForMaster)
		singleReplicaIDInStringFormat = strings.TrimSuffix(singleReplicaIDInStringFormat, "\n")
		singleReplicaIDInStringFormat = strings.TrimSuffix(singleReplicaIDInStringFormat, " ")
		if singleReplicaIDInStringFormat == "No" { //If it was "No", then Quorum test would have had failed
			fmt.Println("Couldn't perform 'get' operation. Master returned 'No'")
			break //Uncomment it if not debugging
		}

		fmt.Println("Client is Listening")
		//Let the server know that it has to transfer files to this client
		JSONStringForArgumentsForServer := encodeArgumentsToJSONString(os.Args[1]+"ForServer", os.Args[2])
		// singleReplicaIDInStringFormat = masterIP //Comment it if not debugging
		callServerForGet(singleReplicaIDInStringFormat, JSONStringForArgumentsForServer)

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
		for _, replica := range listOfReplicas {                             //print the replicas id's where the os.Args[2] file is currently stored
			fmt.Println(replica)
		}
		break
	}
}
