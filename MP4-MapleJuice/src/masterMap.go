package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"strconv"
	"time"

	"fmt"
	"os"
	"strings"
	// "log"
	// "net"
	// "encoding/json"
)

// const PORTForServer = ":8090"
var OPERATION_PORT string = "9090"
var MEMBERSHIP_PORT string = "9091"
var SDFS_REQUEST_PORT string = "9092"

func logIt(messageToLog string) {
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	log.Print(messageToLog)
	file.Close()
}

//Get's it's own IP: referred from MP2 solution uploaded
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

func FetchIPList() []string {
	conn, _ := net.Dial("tcp", "127.0.0.1"+":"+MEMBERSHIP_PORT)
	message, _ := bufio.NewReader(conn).ReadString('\n')
	ip_list := strings.Fields(message)
	return ip_list
}

func SendSDFSCommand(command string) {
	conn, _ := net.Dial("tcp", "127.0.0.1"+":"+SDFS_REQUEST_PORT)
	fmt.Fprintf(conn, command+"\n")
}

func readLines(path string) ([]string, error) { //stolen from stackoverflow :)
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("Error reading the file")
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

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []os.FileInfo) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x.Name()] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x.Name()]; !found {
			diff = append(diff, x.Name())
		}
	}
	return diff
}


func callWorkerNode(workerNodeIP string, fileNameToOperateMapOn string, port string) {
	logIt("callServer() IP:" + workerNodeIP + ":" + port + " " + fileNameToOperateMapOn)
	conn, err := net.Dial("tcp", workerNodeIP+":"+port) //make a connection to the local server
	if err != nil {
		fmt.Print("callServer() err:")
		fmt.Println(err)
		return
	}
	//MasterIP, InputFilePath with name, OutputfilePath with 
	fmt.Fprintf(conn, GetMyIP()+","+os.Args[3]+fileNameToOperateMapOn+","+os.Args[4]+fileNameToOperateMapOn+","+"map"+"\n") // send to socket
	conn.Close()
}

func findNoOfFilesInADirectory(directoryName string) int {
	files, _ := ioutil.ReadDir(directoryName)
	return len(files)
}


// Main thread of execution
// go run masterMap.go wordCount shared/inputData/wordCountInput.txt shared/shardedMapInputData/ shared/shardedMapOutputData/
func main() {
	/*for _, ip := range(FetchIPList()) {
		fmt.Println(ip)
	}*/

	// Check commandline arguments
	if len(os.Args) != 5 {
		fmt.Println("Usage: go run masterMap.go operation inputFileName mapInputFolderPath mapOutputFolderPath")
		return
	}

	fmt.Println(len(os.Args))
	fmt.Println(os.Args[2])

	fmt.Println("Working...")

	//Get the IPs for the VMs
	//var workerIPs = [1]string{"127.0.0.1"} //For debugging on local machine
	var workerIPs = FetchIPList()

	//Get the number of VMs which are alive: N
	var noOfVMs = len(workerIPs)

	//Split the file into N parts: https://stackoverflow.com/questions/7764755/how-to-split-a-file-into-equal-parts-without-breaking-individual-lines
	cmd := exec.Command("split", "-n", "l/"+strconv.Itoa(noOfVMs), os.Args[2])
	err := cmd.Run()
	if err != nil {
		logIt("Couldn't split the file because")
		logIt(err.Error())
	}
	splitFileNames := [10]string{"xaa", "xab", "xac", "xad", "xae", "xaf", "xag", "xah", "xai", "xaj"}

	//SCP all those files into all VMs into shardedMapInputData/ folder
	for _, ip := range workerIPs {
		if ip == "127.0.0.1" { //For debugging
			for i := 0; i < noOfVMs; i++ {
				cmd := exec.Command("scp", splitFileNames[i], os.Args[3])
				//scp -r xaa shared/shardedMapInputData/
				err := cmd.Run()
				if err != nil {
					logIt("Couldn't send file to" + ip + "because")
					logIt(err.Error())
				}
			}
			break
		} else {
			for i := 0; i < noOfVMs; i++ {
				//scp LOCAL_FILE USERID@REMOTE_IP:REMOTE_FILE_PATH
				command_string := "pihess@" + ip + ":workspace/MP4/src/" + os.Args[3] + splitFileNames[i]
				logIt(command_string)
				cmd := exec.Command("scp", splitFileNames[i], command_string) //TODO update the VM number for second arg
				//scp -r xaa shared/shardedMapInputData
				err := cmd.Run()
				if err != nil {
					logIt("Couldn't send file to " + ip + " because")
					logIt(err.Error())
				}
			} //for
		} //else
	} //for

	time.Sleep(5 * time.Second) //Wait for 5 seconds

	//Notify all VMs to start their map task
	for i, ip := range workerIPs {
		go callWorkerNode(ip, splitFileNames[i], OPERATION_PORT) //TODO keep the port number constant
	}

	//Wait for x amount of time, while worker nodes populate the shardedMapOutputData/ folder with their map work
	time.Sleep(5 * time.Second) //Wait for 5 seconds

	//After x amount of seconds, check if N different files are present or not.
	mapOutputFilesNames, _ := ioutil.ReadDir(os.Args[4])
	mapInputFilesNames, _ := ioutil.ReadDir(os.Args[3])
	fmt.Println("OutputFileNames")
	for _, file := range mapOutputFilesNames {
		fmt.Println(file.Name())
	}
	fmt.Println("InputFileNames")
	for _, file := range mapInputFilesNames {
		fmt.Println(file.Name())
	}

	//Find difference between two file directories
	fmt.Println("Difference")
	for _, file := range difference(mapInputFilesNames, mapOutputFilesNames) {
		fmt.Println(file)
	}

	//Define max number of failures
	maxNoOfFailures := 3

	// counter := 0
	for (len(mapOutputFilesNames) != len(mapInputFilesNames)) && (maxNoOfFailures > 0) {
		//Update the IPs for the VMs
		//TODO get the IPs from MP3 interface
		//workerIPs = [1]string{"127.0.0.1"} //For debugging on local machine
		workerIPs = FetchIPList()

		// noOfVMs = len(workerIPs) //Comment it if debugging

		//After x amount of seconds, check if N different files are present or not.
		mapOutputFilesNames, _ := ioutil.ReadDir(os.Args[4])
		mapInputFilesNames, _ := ioutil.ReadDir(os.Args[3])
		inputFilesNotYetProcessedDueToFailure := difference(mapInputFilesNames, mapOutputFilesNames)

		//Find which map tasks were failed and reassign them
		for i := 0; i < len(inputFilesNotYetProcessedDueToFailure) && i < len(workerIPs); i++ {
			go callWorkerNode(workerIPs[i], inputFilesNotYetProcessedDueToFailure[i], OPERATION_PORT) //TODO keep the port number constant
		}

		//Wait for x amount of time, while worker nodes populate the shardedMapOutputData/ folder with their map work
		time.Sleep(5 * time.Second) //Wait for 5 seconds

		maxNoOfFailures = maxNoOfFailures - 1

	}


	if len(mapOutputFilesNames) != len(mapInputFilesNames) {
		fmt.Println("Map Output generated")
	}

}
