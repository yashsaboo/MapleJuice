package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net"
	//"os/exec"
	//"strconv"
	//"time"

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
	fmt.Println("callServer() IP:" + workerNodeIP + ":" + port + " " + fileNameToOperateMapOn)
	conn, err := net.Dial("tcp", workerNodeIP+":"+port) //make a connection to the local server
	if err != nil {
		fmt.Print("callServer() err:")
		fmt.Println(err)
	}
	//MasterIP, InputFilePath with name, OutputfilePath with name
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
	command:="put main.go sdfs_test_file.txt"
	SendSDFSCommand(command)
}
