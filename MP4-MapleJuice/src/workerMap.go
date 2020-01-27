package main

import (
	"bufio"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	// "encoding/json"
	"fmt"
	// "log"
	// "net"
	"os"
	// "strings"
	"dfsinterface"
	"membership/node"
)

var m map[string]int //hashmap......

<<<<<<< HEAD
var uiChan = make(chan string)

var OPERATION_PORT string = "9090"  //for accepting data to process
var MEMBERSHIP_PORT string = "9091" //for sending the alive nodes over
=======
var OPERATION_PORT string = "9090"    //for accepting data to process
var MEMBERSHIP_PORT string = "9091"   //for sending the alive nodes over
>>>>>>> remotes/origin/newYS
var SDFS_REQUEST_PORT string = "9092" // for getting sdfs requests

func getAliveIPsList() []string { //because this is server get from mp3 interface
	node.MeNode.Members.L.Lock()
	defer node.MeNode.Members.L.Unlock()

	var ipList []string
	for _, memberNode := range node.MeNode.Members.Members {
		ipString := memberNode.UDPAddr.IP.String()
		ipList = append(ipList, ipString)
	}
	return ipList
}

func handleMembershipRequests() { //is a tcp connection listener for membership requests
	ln, _ := net.Listen("tcp", ":"+MEMBERSHIP_PORT)
	for {
		conn, _ := ln.Accept()

		for _, ip_name := range getAliveIPsList() {
			fmt.Fprintf(conn, ip_name+" ")
		}
		fmt.Fprintf(conn, "\n")
		conn.Close()
	}
}

func handleSDFSRequests() { //is a tcp connection listener for membership requests
	ln, _ := net.Listen("tcp", ":" + SDFS_REQUEST_PORT)
	for {
		conn, _ := ln.Accept()
		message, _ := bufio.NewReader(conn).ReadString('\n')
		message = strings.Replace(message, "\n", "", -1)
		conn.Close()

		uiChan <- message //push the message into the command channel

	}
}

func trimWhitespaceAndNewlineFeedFromString(str string) string {
	s := strings.Replace(str, "\n", "", -1)
	s = strings.TrimSpace(s)
	return s
}

//https://golangcode.com/how-to-remove-all-non-alphanumerical-characters-from-a-string/
func removeAllNonAlphaNumericCharactersFromString(str string) string {
	// Make a Regex to say we only want letters, space and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9 ]+")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(str, "")
}

//For reference: https://blog.golang.org/go-maps-in-action
func translateToHashMap(path string) {

	//get <SDFS_fie_name> <local_output_path>: get <SDFS_fie_name> <local_output_path>
	file, err := os.Open(path)
	if err != nil {
		fmt.Print("Couldn't open MapInputFile because")
		fmt.Println(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = trimWhitespaceAndNewlineFeedFromString(line)
		line = removeAllNonAlphaNumericCharactersFromString(line)
		s := strings.Split(line, " ")

		for _, word := range s {
			if word == "" {
				break
			}
			i, ok := m[word] //Checks for the word in hashmap. If present, then i stores the current value and ok holds true bool value, else, false value and i=0
			if ok == true {
				m[word] = i + 1 //If value already present, then just increment the count
			} else {
				m[word] = 1 //If value not present, then initilialise it to 1
			}
		}
	}
}

func flushHashMaptoFile(filePathwithName string) error { //stolen from stackoverflow :)
	file, err := os.Create(filePathwithName)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	s := strings.Split(filePathwithName, "/")
	fileName := s[len(s)-1]

	//Iterate over hashmpa and flush each key,value to file
	for key, value := range m {

		//Add to SDFS

		//First create file with key in shared/SDFS/fileName_key
		full_file_path := "shared/SDFS/" + fileName + "_" + key
		file2, err2 := os.Create(full_file_path)
		if err2 != nil {
			return err2
		}
		defer file2.Close()

		w2 := bufio.NewWriter(file2)
		fmt.Fprintln(w2, key+","+strconv.Itoa(value))
		//Put it into SDFS : TODO
		//put <path_to_local_file> <SDFS_file_name>: put "shared/SDFS/fileName_" + key "fileName_" + key
		sdfs_command := "put " + full_file_path + " " + fileName + "_" + key
		uiChan <- sdfs_command //push the message into the command channel


		fmt.Fprintln(w, key+","+strconv.Itoa(value))
		fmt.Println("Key:", key, "Value:", value) //For debugging
	}
	return w.Flush()
}

func handleWordCount(message string) {

	message = trimWhitespaceAndNewlineFeedFromString(message)
	fmt.Println(message)
	s := strings.Split(message, ",")

	//Translate the file to Hashmap
	translateToHashMap(s[1])

	//Flush the Hashmap to the mapoutput
	err := flushHashMaptoFile(s[2])
	if err != nil {
		fmt.Println("Could not write to file: " + s[2])
	} else {
		//SCP the file to shardedMapOutputData/ folder on Master Node
		ip := s[0]
		command_string := "pihess@" + ip + ":workspace/MP4/src/" + s[2]
		fmt.Println(command_string)
		cmd := exec.Command("scp", s[2], command_string) //TODO update the VM number for second arg
		//scp -r xaa shared/shardedMapInputData
		err := cmd.Run()
		if err != nil {
			fmt.Print("Couldn't SCP file back to " + ip + " because")
			fmt.Println(err)
		}

	}
}

func handleCUMTD(message string) {

}

// Main thread of execution
// go run workerMap.go wordCount 8090
func main() {

	//starting up the mp3 subsystem
	
	dfsinterface.FileSystemRun(uiChan) //sets up everything including the MeNode variable in node.go

	go handleMembershipRequests()
	go handleSDFSRequests()

	//Create a hashmap
	m = make(map[string]int)

	// Check commandline arguments
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run workerMap.go operation")
		return
	}

	fmt.Println(len(os.Args))
	fmt.Println(os.Args[1])

	fmt.Println("Got the args")

	// Listen to Master on some port infinitely
	fmt.Println("Data input listening on ", OPERATION_PORT)

	// listen on all interfaces //Uncomment if not debugging
	ln, _ := net.Listen("tcp", ":"+OPERATION_PORT)

	fmt.Println("Going in infinite loop")

	for { //accept connections

		//Uncomment the next line if not deubgging
		// message := "127.0.0.1,shared/shardedMapInputData/wordCount.txt,shared/shardedMapOutputData/xaa "
		//MasterIP, InputFilePath with name, OutputfilePath with name
		//Comment the next four lines if debugging
		conn, _ := ln.Accept()
		fmt.Println("I did Listen")

		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Println("Message Received:", string(message))

		//Create a hashmap
		m = make(map[string]int)

		if os.Args[1] == "wordCount" {
			go handleWordCount(message) //Add go if not debugging
			// break                    //Comment if not debugging
		} else if os.Args[1] == "cumtd" {
			go handleCUMTD(message) //Add go if not debugging
			// break                   //Comment if not debugging
		} else {
			fmt.Println("Wrong Operation")
			break
		}
	}
}
