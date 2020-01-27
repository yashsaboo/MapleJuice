package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"membership"
	"net"
	"os"
	"sort"
	"strings"
	"time"
)

type InPacketStruct struct {
	PacketType string //client_request, server_request (who is sending this packet)
	Command    string //put, ls, read, write
	FileName   string //name of the file on the server
}

type MetaDataStruct struct { //who has what files
	Alive     bool
	FileNames []string //files stored on this node
}

//GLOBAL VARS
//var memberMap = make(map[int]*membership.MemberNode) //only access synchronously
var metaMap = make(map[string]*MetaDataStruct) //sync, ip -> MetaDataStruct

var masterIP string

var myIP string

const PORT = ":8090"

func getSortedKeys(theMap *(map[int]*membership.MemberNode)) []int { //sync
	// To store the keys in sorted order since Go doesn't store map in a sorted order
	var keys []int
	for k := range *theMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
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
	// fmt.Println(lines)
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

func doesNodeContainFile(fileName string, nodeIP string) bool { //does a given node contain a file with the given name
	fileList := metaMap[nodeIP].FileNames
	for _, file := range fileList {
		if file == fileName {
			return true
		}
	}
	return false
}

func getNodeWithFile(fileName string) string { //returns the IP of an alive node that has the file specified
	for key, value := range metaMap {
		if value.Alive { // only check if the node is alive
			if doesNodeContainFile(fileName, key) {
				return key
			}
		}
	}
	fmt.Println("getNodeWithFile failed because it could not find an alive node with the file %s", fileName)
	return "No" //should not occure because there should be at least 1 alive node with the file
}

func getCandidateNode(fileName string) string {
	//return the node that does not have the file already and has the fewest number of files
	fewestFiles := 9999999
	bestIP := "invalid ip"

	for key, value := range metaMap {
		if value.Alive { //node must be alive to be a candidate
			if !doesNodeContainFile(fileName, key) { //must not already have the file
				if len(value.FileNames) < fewestFiles { //must have the fewest numebr of files stored already
					fewestFiles = len(value.FileNames)
					bestIP = key
				}
			}
		}
	}
	return bestIP
}

func sendPacket(ip string, message string) {
	conn, _ := net.Dial("tcp", ip+PORT)
	fmt.Fprintf(conn, message+"\n")
}

//called by the server and tells one node to send a file to
func sendFileTransferRequest(fileHolderIP string, fileDestIP string, fileName string) {
	//PacketType = server_request
	//Command = ip to send to
	//FileName = file to send
	packet := "{\"PacketType\":\"transfer_request\", \"Command\":\"" + fileDestIP + "\", \"FileName\":\"" + fileName + "\"}"
	sendPacket(fileHolderIP, packet)
}

func delegateFileRelocation(failureIP string) { //called by master, sends packets to nodes with replicas telling them who to send replicas to
	fileList := metaMap[failureIP].FileNames
	for _, file := range fileList { //for each file that was on the failed node

		replicaNode := getNodeWithFile(file)      //node that has a file
		replacementNode := getCandidateNode(file) //node that does not have the file already and has the fewest number of files

		//send the replica node a packet telling it to send the file to the replacement Node
		sendFileTransferRequest(replicaNode, replacementNode, file)
	}
}

func sendFullMetaDict(ip string) {
	//TODO send all of the data to the new node that just connected to the network
}

func handleNodeConnection(ip string) { //sync and called by anyone (not just master)
	fmt.Println("Node reconnected with ip: ", ip)
	//Filenames slice already cleared so no need to mess with it
	metaMap[ip].Alive = true
	if myIP == masterIP {
		sendFullMetaDict(ip)
	}
}

func handleNewNodeConnection(ip string) { //sync and called by anyone (not just master)
	fmt.Println("New node connected with ip: ", ip)
	newStruct := MetaDataStruct{Alive: true, FileNames: make([]string, 0)}
	metaMap[ip] = &newStruct

	if myIP == masterIP {
		sendFullMetaDict(ip)
	}
}

func handleNodeDisconnection(ip string) { //sync and called by anyone (not just master)
	fmt.Println("Node disconnected with ip: ", ip)
	metaMap[ip].Alive = false

	if myIP == masterIP { //I am master so tell how to redistribute the files stored on the failed node
		delegateFileRelocation(ip)
	}

	// remove metadata because that file system reset for the now downed node
	// don't remove data before delegating file relocation because need to know
	// which files need their replicas duplicated
	metaMap[ip].FileNames = make([]string, 0) //setting to empty slice

}

func updateMaster() {
	lowestAliveIP := "999.999.999.999"
	lowestAliveIP = "127.0.0.1" //only for debugging
	for key, value := range metaMap {
		fmt.Println("updateMaster() in metaMap Loop: ")
		if key < lowestAliveIP && value.Alive {
			lowestAliveIP = key
		}
	}
	masterIP = lowestAliveIP
}

func handleMembershipChange(newMemberMap map[int]*membership.MemberNode) { //sync

	for _, value := range newMemberMap {
		metaNode, exists := metaMap[value.IP]
		if !exists { //not in the meta stores so new
			handleNewNodeConnection(value.IP)
		} else {
			if value.Alive && !metaNode.Alive { //reconnected
				handleNodeConnection(value.IP)
			} else if !value.Alive && metaNode.Alive { //just found out it's dead
				handleNodeDisconnection(value.IP)
			}
		}
	}
	updateMaster()
	printMembershipList(newMemberMap)
	fmt.Println("Master ip: ", masterIP)
}

func tcpListen(connectionChan chan net.Conn) { //async

	fmt.Println("Filesystem Network listening on ", PORT)

	// listen on all interfaces
	ln, _ := net.Listen("tcp", PORT)

	fmt.Println("Going in infinite loop")

	for { //accept connections then push to main thread in buffer
		conn, _ := ln.Accept()
		fmt.Println("I did Listen")
		connectionChan <- conn
	}
}

func decodeInJson(packet string) InPacketStruct { //sync (convert packet string -> InPacketStruct)
	var packetStruct InPacketStruct
	json.Unmarshal([]byte(packet), &packetStruct)
	return packetStruct
}

//start sending a file that is stored on this node to the destination ip
func sendFile(conn net.Conn, fileName string) {
	// packet := "{\"PacketType\":\"write_init\", \"Command\":\"" + destIP + "\", \"FileName\":\"" + fileName + "\"}"
	// conn, _ := net.Dial("tcp", destIP+PORT)
	// fmt.Fprintf(conn, packet+"\n")

	//read the file into memory from filesystem
	fmt.Println("In sendFile()")

	fmt.Println("sendFile() fileName:" + fileName)

	lines, err := readLines("data/" + fileName)
	if err != nil {
		fmt.Println("Could not read from file %s", fileName)
		return
	}

	//sending the file through the connection line by line
	//Read file and send it to server
	for _, line := range lines {
		fmt.Fprintf(conn, string(line)+"\n")
		time.Sleep(1 * time.Microsecond) //Required to do this since files gets transferred really fast, so we need to keep it in control
		// fmt.Println(i)
	}
	conn.Close()
}

func handleTransferRequest(packet InPacketStruct) {
	// destinationIP := packet.Command
	// fileName := packet.FileName

	// sendFile(fileName, destinationIP)
}

func acceptFileTransfer(conn net.Conn, fileName string) {

	fmt.Println("acceptFileTransfer() fileName:" + fileName)

	// //read from connection line by line
	// line, err := bufio.NewReader(conn).ReadString('\n')
	// lineSlice := make([]string, 0)

	// fmt.Println("acceptFileTransfer() I'm here 1")

	// for err != nil { //read all of the lines from the file over the network
	// 	fmt.Println("acceptFileTransfer() I'm here 2")
	// 	lineSlice = append(lineSlice, line)
	// 	fmt.Println("acceptFileTransfer() I'm here 3")
	// 	line, err = bufio.NewReader(conn).ReadString('\n')
	// 	fmt.Println("acceptFileTransfer() I'm here 4")
	// }
	// fmt.Println("acceptFileTransfer() I'm here 5")

	var lineSlice []string

	// run loop forever (or until ctrl-c)
	for {
		fmt.Println("acceptFileTransfer() I'm here 1")
		// will listen for message to process ending in newline (\n)
		message, err2 := bufio.NewReader(conn).ReadString('\n')
		if err2 != nil {
			break
		}
		fmt.Println("Got a line")
		lineSlice = append(lineSlice, message)
	}
	fmt.Println("acceptFileTransfer() I'm here 5")

	err := writeLines(lineSlice, "data/"+fileName)
	if err != nil {
		fmt.Println("Could not write to file" + fileName)
	}
}

//sends ip and then the list of files on that ip then "END" when done with a list
func sendLS(conn net.Conn) {
	for key, value := range metaMap {
		if value.Alive { // only part of the network if
			conn.Write([]byte(key + "\n"))
			for _, value := range value.FileNames {
				conn.Write([]byte(value + "\n")) //sending the names of all the files on that machine
			}
			conn.Write([]byte("END\n"))
		}
	}
}

//sends ips of 'count' nodes that can be written to for the given file name
func sendCandidateList(conn net.Conn, fileName string, count int) {
	for idx := 0; idx < count; idx++ {
		nodeIP := getCandidateNode(fileName)
		conn.Write([]byte(nodeIP + " ; "))
	}
	conn.Write([]byte("\n"))
}

//Does not use json for responses
func handleClientRequest(conn net.Conn, packet InPacketStruct) { //sync
	if packet.Command == "who_is_master" { //accepted on any machine
		conn.Write([]byte(masterIP + "\n")) //Sends the Master's IP
	} else if packet.Command == "ls" { //accepted on any machine
		sendLS(conn)
	} else if packet.Command == "putForMaster" { //must be on master
		//TODO: write-write conflict: 1. Compare timestamp with current time.
		timestamp := time.Now() //Dummy, initialise it with the file's timestamp in the metadata

		//2. If less than 1 minute, ask for confirmation
		currentTime := time.Now()
		difference := currentTime.Sub(timestamp)
		difference = (difference / time.Second) //Represent different in seconds
		fmt.Printf("difference = %v\n", difference)
		confirmationMessage := "no"
		if difference < 60 {
			conn.Write([]byte("confirm" + "\n")) //Sends the Master's IP
			confirmationMessage, _ = bufio.NewReader(conn).ReadString('\n')
			confirmationMessage = strings.TrimSuffix(confirmationMessage, "\n")
			confirmationMessage = strings.TrimSuffix(confirmationMessage, " ")
		}
		//check if confirmation is "yes"
		fmt.Println("handleClientRequest() confirmationMessage:" + confirmationMessage)
		if strings.Contains(confirmationMessage, "yes") {
			fmt.Println("handleClientRequest() In conformed message if")
			sendCandidateList(conn, packet.FileName, 4) //Sends 4 replica for doing 'put' operation
		}
	} else if packet.Command == "putForServer" { //accepted on any machine
		conn.Write([]byte("ok" + "\n"))           //Sends "ok" to let the client to know that file can be transferred now
		acceptFileTransfer(conn, packet.FileName) //Accepts file transfer
		// } else if packet.Command == "write" { //must be on master
		// 	sendCandidateList(conn, packet.FileName, 4)
	} else if packet.Command == "getForMaster" { //must be on master
		nodeIP := getNodeWithFile(packet.FileName)
		conn.Write([]byte(nodeIP + "\n")) //sends the replica IP for performing "get" operation
	} else if packet.Command == "getForServer" { //must be on master
		conn.Write([]byte("ok" + "\n")) //sends "ok" to let the client to know that start listening to file transfer
		sendFile(conn, packet.FileName)
	}
}

func handleMetaUpdate(packet InPacketStruct) { //single line update
	//TODO: update metadata in local map

}

func handleFullMetaUpdate(conn net.Conn, packet InPacketStruct) { //single line update
	//TODO: update metadata in local map from all the data comming in from the connection

}

func handleConnection(conn net.Conn) { //sync
	message, _ := bufio.NewReader(conn).ReadString('\n')
	fmt.Println("Message Received:", string(message))

	packet := decodeInJson(message)

	if packet.PacketType == "client_request" {
		handleClientRequest(conn, packet)
	} else if packet.PacketType == "transfer_request" { //Won't be able to use this type of packet type based on my updated put and get operation
		handleTransferRequest(packet)
		return //transfer requests don't need a response packet : UPDATE - they require one
	} else if packet.PacketType == "write_init" { //actually about to write to this node (not asking who to write to)
		acceptFileTransfer(conn, packet.FileName)
		//TODO: Send single-meta update to all active nodes

		return //file transfer function will handle the connection cleanup
	} else if packet.PacketType == "meta_update" { //packet for updating metadata to know that a node has a new file now
		handleMetaUpdate(packet)
		return //file transfer function will handle the connection cleanup
	} else if packet.PacketType == "full_meta_update" { //packet for updating metadata to know that a node has a new file now
		handleFullMetaUpdate(conn, packet)
		return //file transfer function will handle the connection cleanup
	} else {
		fmt.Println("Server got a packet with unknown type: ", packet.PacketType)
		conn.Write([]byte("error" + "\n")) //respond to the request
	}
}

func printMembershipList(memberMap map[int]*membership.MemberNode) { //sync
	fmt.Printf("My members: [")
	for id := range memberMap {
		fmt.Printf("%d %t;  ", id, memberMap[id].Alive)
	}
	fmt.Printf("]\n")
}

func main() {
	myIP = membership.GetMyIP()

	//buffered membership channel that gets updates whenever the membership map updates
	membershipChannel := make(chan map[int]*membership.MemberNode, 20)
	membership.MembershipRun(membershipChannel) //starts go rotines and returns immidiately

	updateMaster()
	fmt.Println("MasterIP: " + masterIP)

	connectionChannel := make(chan net.Conn, 20)
	go tcpListen(connectionChannel)

	for { //main event loop
		select { //WILL Wait until one is ready and then handle it
		case newMembershipMap := <-membershipChannel:
			handleMembershipChange(newMembershipMap)
		case conn := <-connectionChannel:
			fmt.Println("Server handleing a new connection")
			handleConnection(conn)
		}
	}
}
