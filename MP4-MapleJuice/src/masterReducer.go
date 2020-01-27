package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	// "encoding/json"
	"fmt"
	// "log"
	// "net"
	"os"
	// "strings"
	"hash/fnv"
	"math"
	"sort"
)

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
	fmt.Println("callServer() IP:" + workerNodeIP + ":" + port)
	conn, err := net.Dial("tcp", workerNodeIP+":"+port) //make a connection to the local server
	if err != nil {
		fmt.Print("callServer() err:")
		fmt.Println(err)
		return
	}
	//MasterIP, InputFilePath with name, OutputfilePath with name, "reduce"
	fmt.Fprintf(conn, GetMyIP()+","+os.Args[4]+fileNameToOperateMapOn+","+os.Args[5]+fileNameToOperateMapOn+","+"reduce"+"\n") // send to socket
	conn.Close()
}

func createNFiles(N int, filePathWithName string) { //stolen from stackoverflow :)

	for i := 0; i < N; i++ {

		file, err := os.Create(filePathWithName + strconv.Itoa(i))
		defer file.Close()
		if err != nil {
			fmt.Println(err)
		} else {
			w := bufio.NewWriter(file)
			w.Flush()
		}
	}
}

func findHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

//https://stackoverflow.com/questions/31239330/go-langs-equivalent-of-charcode-method-of-javascript
func findRange(s string) uint32 {
	// fmt.Print(s + ":")
	// fmt.Println(uint32([]rune(strings.ToLower(s))[0] - 97))
	return uint32([]rune(strings.ToLower(s))[0] - 97)
}

func translateSingleMapOutputFileToReduceInputFiles(mapOutputFilePathwithFileName string, noOfReducerFiles int) {

	//Open the mapOutput file
	file, err := os.Open(mapOutputFilePathwithFileName)
	defer file.Close()
	if err != nil {
		fmt.Println(err)
	}
	
	scanner := bufio.NewScanner(file)

	//Iterate over each line in mapOutput file
	for scanner.Scan() {

		line := scanner.Text()
		s := strings.Split(line, ",")

		fileValue := 0

		if os.Args[2] == "1" {
			//Find hash value of the word
			fileValue = int(math.Mod(float64(findHash(s[0])), float64(noOfReducerFiles)))
		} else {
			//Find range value of the word
			divValue := 26/noOfReducerFiles + 1
			//fmt.Print("divValue:")
			//fmt.Println(divValue)
			//fmt.Print("findRange:")
			//fmt.Println(findRange(s[0]))
			// fileValue = int(math.Mod(float64(findRange(s[0])), float64(modValue)))
			fileValue = int(float64(findRange(s[0])) / float64(divValue))
			//fmt.Print("fileValue:")
			//fmt.Println(fileValue)
		}

		//Append to a file whose name matches the hash value: https://yourbasic.org/golang/append-to-file/
		f, err := os.OpenFile(os.Args[4]+"/"+strconv.Itoa(fileValue), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logIt(err.Error())
		}
		
		if _, err := f.WriteString(line + "\n"); err != nil {
			logIt(err.Error())
		}
		f.Close()
	}
}

func performPatitioning() {

	//Get the list of files present in MapOutFolder and ReduceInputFolder
	mapOutputFilesNames, _ := ioutil.ReadDir(os.Args[3])
	reduceInputFilesNames, _ := ioutil.ReadDir(os.Args[4])

	//Iterate over each MapOutput File
	for _, file := range mapOutputFilesNames {
		translateSingleMapOutputFileToReduceInputFiles(os.Args[3]+file.Name(), len(reduceInputFilesNames))
	}

	logIt("Partition Successful")

}

// Main thread of execution
// go run masterReducer.go wordCount 1 shared/shardedMapOutputData/ shared/shardedReduceInputData/ shared/shardedReduceOutputData/
// go run masterReducer.go wordCount 2 shared/shardedMapOutputData/ shared/shardedReduceInputData/ shared/shardedReduceOutputData/
// 3rd argument: 1 - Hash Partitioning; 2 - Range Partitioning
func main() {

	// Check commandline arguments
	if len(os.Args) != 7 {
		fmt.Println("Usage: go run masterMap.go operation typeOfPartitioning mapOutputFolderPath reduceInputFolderPath reduceOutputFolderPath finalOutput")
		return
	}

	fmt.Println(len(os.Args))
	fmt.Println(os.Args[2])

	fmt.Println("Got the args")

	//Get the number of VMs which are alive: M
	//TODO connect to the MP3 interface
	//var noOfVMs = 3

	//Get the IPs for the VMs
	//TODO get the IPs from MP3 interface
	//var workerIPs = [1]string{"127.0.0.1"} //For debugging on local machine


	var workerIPs = FetchIPList()

	//Get the number of VMs which are alive: N
	var noOfVMs = len(workerIPs)

	//Create M files in shardedReduceInputData/ folder
	createNFiles(noOfVMs, os.Args[4])

	//Perform Partioning
	performPatitioning()

	//SCP all those files into all VMs into shardedReduceInputData/ folder
	for _, ip := range workerIPs {
		if ip == "127.0.0.1" { //For debugging
			break
		} else {
			for i := 0; i < noOfVMs; i++ {
				command_string := "pihess@" + ip + ":workspace/MP4/src/"+os.Args[4]+strconv.Itoa(i)
				logIt(command_string)
				cmd := exec.Command("scp", os.Args[4]+strconv.Itoa(i), command_string) //TODO update the VM number for second arg
				//command_string := "pihess@" + ip + ":workspace/MP4/src/" + os.Args[4] + splitFileNames[i]
				//cmd := exec.Command("scp", splitFileNames[i], command_string)
				//scp -r xaa shared/shardedReduceInputData
				err := cmd.Run()
				if err != nil {
					logIt("Couldn't send file to " + ip + " because")
					logIt(err.Error())
				}
			} //for
		} //else
	} //for

	//Notify all VMs to start their map task
	for i, ip := range workerIPs {
		go callWorkerNode(ip, strconv.Itoa(i), OPERATION_PORT) //TODO keep the port number constant
	}

	//Wait for x amount of time, while worker nodes populate the shardedMapOutputData/ folder with their map work
	time.Sleep(5 * time.Second) //Wait for 5 seconds

	//After x amount of seconds, check if N different files are present or not.
	reduceOutputFilesNames, _ := ioutil.ReadDir(os.Args[5])
	reduceInputFilesNames, _ := ioutil.ReadDir(os.Args[4])
	fmt.Println("OutputFileNames:")
	for _, file := range reduceOutputFilesNames {
		fmt.Println(file.Name())
	}
	fmt.Println("InputFileNames:")
	for _, file := range reduceInputFilesNames {
		fmt.Println(file.Name())
	}

	//Find difference between two file directories
	fmt.Println("Difference")
	for _, file := range difference(reduceInputFilesNames, reduceOutputFilesNames) {
		fmt.Println(file)
	}

	//Define max number of failures
	maxNoOfFailures := 3

	// counter := 0

	for (len(reduceOutputFilesNames) != len(reduceInputFilesNames)) && (maxNoOfFailures > 0) {
		//Update the IPs for the VMs
		//TODO get the IPs from MP3 interface
		//workerIPs = [1]string{"127.0.0.1"} //For debugging on local machine
		workerIPs = FetchIPList()

		// noOfVMs = len(workerIPs) //Comment it if debugging

		//After x amount of seconds, check if N different files are present or not.
		reduceOutputFilesNames, _ := ioutil.ReadDir(os.Args[5])
		reduceInputFilesNames, _ := ioutil.ReadDir(os.Args[4])
		inputFilesNotYetProcessedDueToFailure := difference(reduceInputFilesNames, reduceOutputFilesNames)

		//Find which map tasks were failed and reassign them
		for i := 0; i < len(inputFilesNotYetProcessedDueToFailure) && i < len(workerIPs); i++ {
			go callWorkerNode(workerIPs[i], inputFilesNotYetProcessedDueToFailure[i], OPERATION_PORT) //TODO keep the port number constant
		}

		//Wait for x amount of time, while worker nodes populate the shardedMapOutputData/ folder with their map work
		time.Sleep(5 * time.Second) //Wait for 5 seconds

		maxNoOfFailures = maxNoOfFailures - 1


	}

	if len(reduceOutputFilesNames) != len(reduceInputFilesNames) {
		fmt.Println("Reduce Output generated")
	}

	//go run theEndgame.go operation shared/shardedReduceOutputData/ shared/outputData/wordCountOutput.txt 0
	/*fmt.Println("Trying to run the command go run theEndgame.go shared/shardedReduceOutputData/ shared/outputData/"+os.Args[1]+"Output.txt 0")
	cmd := exec.Command("go", "run", "theEndgame.go", "shared/shardedReduceOutputData/", "shared/outputData/"+os.Args[1]+"Output.txt", "0")
	err := cmd.Run()
	if err != nil {
		fmt.Print("could not run endgame: ")
		fmt.Println(err)
	} else {
		fmt.Println("Endgame should have been run successfully")
	}*/
	//Combines Reduce Worker files into one into the location mentioned by the user
	combineFilesAndFlushItToOutput()
}


func combineFilesAndFlushItToOutput() {

	reduceOutputFilesNames, _ := ioutil.ReadDir(os.Args[5])

	file, err := os.Create(os.Args[6])
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	for _, fileName := range reduceOutputFilesNames {

		//Open the reduceOutput file
		file2, err2 := os.Open(os.Args[5] + fileName.Name())
		if err2 != nil {
			fmt.Println(err2)
		}
		defer file2.Close()

		scanner := bufio.NewScanner(file2)

		var listOfLines []string

		//Iterate over each line in mapOutput file and store it in sorting variable
		for scanner.Scan() {
			line := scanner.Text()
			listOfLines = append(listOfLines, line)
		}

		//Sort
		sort.Strings(listOfLines)

		//Flush it to the file
		for _, line := range listOfLines {
			line = strings.ReplaceAll(line, ",", "\t")
			fmt.Fprintln(w, line)
		}
	}
	w.Flush()
}
