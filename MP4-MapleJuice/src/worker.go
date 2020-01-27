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
	"path"
	"dfsinterface"
	"membership/node"
	"io/ioutil"
)

var m map[string]int //hashmap......
var mForReverse map[string]string
var uiChan = make(chan string, 200)

var OPERATION_PORT string = "9090"    //for accepting data to process
var MEMBERSHIP_PORT string = "9091"   //for sending the alive nodes over
var SDFS_REQUEST_PORT string = "9092" // for getting sdfs requests

func logIt(messageToLog string) {
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	log.Print(messageToLog)
	file.Close()
}

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

func handleSDFSRequests(command_chan chan string) { //is a tcp connection listener for membership requests
	ln, _ := net.Listen("tcp", ":"+SDFS_REQUEST_PORT)
	for {
		conn, _ := ln.Accept()
		message, _ := bufio.NewReader(conn).ReadString('\n')
		message = strings.Replace(message, "\n", "", -1)
		conn.Close()

		command_chan <- message //push the message into the command channel

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
func translateToHashMap(path string) { //stolen from stackoverflow :)
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		logIt("Couldn't open MapInputFile because")
		logIt(err.Error())
	}

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

	//fmt.Println("in translateToHashMap")

	//Iterate over hashmpa and flush each key,value to file
	//for key, value := range m {
		//fmt.Println("Key:", key, "Value:", value) //For debugging
	//}
}

//For reference: https://blog.golang.org/go-maps-in-action
func translateToHashMapForReduce(path string) { //stolen from stackoverflow :)
	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		logIt("Couldn't open MapInputFile because")
		logIt(err.Error())
	} else {
		

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			line = trimWhitespaceAndNewlineFeedFromString(line)
			//fmt.Println("Line is: "+line)
			if line == "" || line == " " {
				continue
			}
			s := strings.Split(line, ",")

			z, _ := strconv.Atoi(s[1])

			i, ok := m[s[0]] //Checks for the word in hashmap. If present, then i stores the current value and ok holds true bool value, else, false value and i=0
			if ok == true {
				//fmt.Println(s[0])
				m[s[0]] = i + z //If value already present, then just increment the count
			} else {
				m[s[0]] = z //If value not present, then initilialise it to 1
			}
		}
	}
}

func flushHashMaptoFile(filePathwithName string) error { //stolen from stackoverflow :)
	file, err := os.Create(filePathwithName)
	defer file.Close()
	
	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)

	s := strings.Split(filePathwithName, "/")
	fileName := s[len(s)-1]

	//fmt.Print("trying to write to sdfs")

	//Iterate over hashmpa and flush each key,value to file
	for key, value := range m {
		//fmt.Println("Starting for loop")

		//Add to SDFS

		//First create file with key in shared/SDFS/fileName_key
		full_file_path := "shared/SDFS/" + fileName + "_" + key
		file2, err2 := os.Create(full_file_path)
		if err2 != nil {
			fmt.Println("error writing to local sdfs file...")
			return err2
		}

		w2 := bufio.NewWriter(file2)
		fmt.Fprintln(w2, key+","+strconv.Itoa(value))
		w2.Flush()
		file2.Close()
		//fmt.Println("FILE SCLSOSLELSE")
		//Put it into SDFS : TODO
		//put <path_to_local_file> <SDFS_file_name>: put "shared/SDFS/fileName_" + key "fileName_" + key
		

		//sdfs_command := "put " + full_file_path + " " + fileName + "_" + key
		//uiChan <- sdfs_command //push the message into the command channel

		//storing for the 1st open file (aka outside of the for loop)
		fmt.Fprintln(w, key+","+strconv.Itoa(value))
		//fmt.Println("Key:", key, "Value:", value) //For debugging
	}
	//fmt.Println("done with for loop")
	return w.Flush()
	//return nil
}

func flushHashMaptoFileForReduce(filePathwithName string) error {
	file, err := os.Create(filePathwithName)
	defer file.Close()
	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)

	//Iterate over hashmpa and flush each key,value to file
	for key, value := range m {
		fmt.Fprintln(w, key+","+strconv.Itoa(value))
		//fmt.Println("Key:", key, "Value:", value) //For debugging
	}
	return w.Flush()
}

func handleWordCount(message string) {
	logIt("Working on wordCount now on the server...")
	message = trimWhitespaceAndNewlineFeedFromString(message)
	fmt.Println(message)
	s := strings.Split(message, ",")

	//Translate the file to Hashmap
	if s[3] == "map" {
		translateToHashMap(s[1])
	} else {
		translateToHashMapForReduce(s[1])
	}

	did_error := false

	//Flush the Hashmap to the mapoutput
	if s[3] == "map" {
		err := flushHashMaptoFile(s[2])
		if err != nil {
			//fmt.Println("Map flushed1")
			did_error = true
		}
	} else {
		err := flushHashMaptoFileForReduce(s[2])
		if err != nil {
			//fmt.Println("Map flushed for reduce")
			did_error = true
		}
	}


	if did_error {
		//fmt.Println("Error did occure")
		logIt("Could not write to file: " + s[2])
	} else {
		//fmt.Println("Error did not occure should be SCPing now......")
		//SCP the file to shardedMapOutputData/ folder on Master Node
		ip := s[0]
		command_string := "pihess@" + ip + ":workspace/MP4/src/" + s[2]
		logIt(command_string)
		cmd := exec.Command("scp", s[2], command_string) //TODO update the VM number for second arg
		//scp -r xaa shared/shardedMapInputData
		err := cmd.Run()
		if err != nil {
			logIt("Couldn't send file back to " + ip + " because ")
			logIt(err.Error())
		}

	}
	//fmt.Println("Finished the wordCount function")
	if (os.Args[2] == "1" && s[3] != "map") {
		directoryName := "shared/SDFS/"
		dir, err := ioutil.ReadDir(directoryName)
		if err != nil {
			fmt.Println(err)
		} else {
			for _, d := range dir {
				os.RemoveAll(path.Join([]string{directoryName, d.Name()}...))
			}
		}
	}
}



func handleCUMTD(message string) {

}

func handleReverseWebLinkForMap(message string) {

	message = trimWhitespaceAndNewlineFeedFromString(message)
	logIt(message)
	s := strings.Split(message, ",")

	logIt("Dataset File:"+ s[1]+ "Output File:"+ s[2]) //For debugging

	//Open Dataset File
	fileDataset, errDataset := os.Open(s[1])
	if errDataset != nil {
		fmt.Print("Couldn't open MapInputFile because")
		fmt.Println(errDataset)
	}
	defer fileDataset.Close()

	//Open Output File
	fileOutput, errOutput := os.Create(s[2])
	if errOutput != nil {
		logIt(errOutput.Error())
	}
	defer fileOutput.Close()

	//Writer for output file
	w := bufio.NewWriter(fileOutput)
	

	//Reader for dataset
	scanner := bufio.NewScanner(fileDataset)

	//Read from dataset and write it into mapper output file
	for scanner.Scan() {
		line := scanner.Text()
		if (line == "") || (line == " ") {
			continue
		}
		pages := strings.Split(line, "\t")
		fmt.Fprintln(w, pages[1]+","+pages[0])
		//fmt.Println("Page 0:", pages[0], "Page 1:", pages[1]) //For debugging
	}
	w.Flush()

	//SCP the file to shardedMapOutputData/ folder on Master Node
	ip := s[0]
	command_string := "pihess@" + ip + ":workspace/MP4/src/" + s[2]
	
	fmt.Println(command_string)
	cmd := exec.Command("scp", s[2], command_string) //TODO update the VM number for second arg
	//scp -r xaa shared/shardedMapInputData
	err := cmd.Run()
	if err != nil {
		logIt("Couldn't send file back to " + ip + " because ")
		logIt(err.Error())
	}

	fmt.Println("Finished the reverseURL function")
}


func translateToHashMapForReverseWebLinkForReduce(path string) { //stolen from stackoverflow :)
	file, err := os.Open(path)
	if err != nil {
		logIt("Couldn't open MapInputFile because")
		logIt(err.Error())
	} else {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			line = trimWhitespaceAndNewlineFeedFromString(line)

			s := strings.Split(line, ",")

			// z, _ := strconv.Atoi(s[1])

			i, ok := mForReverse[s[0]] //Checks for the word in hashmap. If present, then i stores the current value and ok holds true bool value, else, false value and i=0
			if ok == true {
				//logIt(s[0])
				mForReverse[s[0]] = i + " " + s[1] //If value already present, then just append it to existing string with tab as a seperator
			} else {
				mForReverse[s[0]] = s[1] //If value not present, then initilialise it to the pageA
			}
		}
	}
}

func flushHashMaptoFileForReverseWebLinkForReduce(filePathwithName string) error { //stolen from stackoverflow :)
	file, err := os.Create(filePathwithName)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	//Iterate over hashmpa and flush each key,value to file
	for key, value := range mForReverse {
		fmt.Fprintln(w, key+"\t"+value)
		//fmt.Println("Key:", key, "Value:", value) //For debugging
	}
	return w.Flush()
}

func handleReverseWebLinkForReduce(message string) {

	message = trimWhitespaceAndNewlineFeedFromString(message)
	logIt(message)
	s := strings.Split(message, ",")

	//Translate the file to Hashmap
	translateToHashMapForReverseWebLinkForReduce(s[1])

	//Flush the Hashmap to the reduceoutput
	err := flushHashMaptoFileForReverseWebLinkForReduce(s[2])
	if err != nil {
		logIt("Could not write to file: " + s[2])
	} else {
		//SCP the file to shardedReduceOutputData/ folder on Master Node
		//fmt.Println("Error did not occure should be SCPing now......")
		//SCP the file to shardedReduceOutputData/ folder on Master Node
		ip := s[0]
		command_string := "pihess@" + ip + ":workspace/MP4/src/" + s[2]
		//fmt.Println("Trying to scp back to master!")
		logIt(command_string)
		cmd := exec.Command("scp", s[2], command_string) //TODO update the VM number for second arg
		//scp -r xaa shared/shardedMapInputData
		err := cmd.Run()
		if err != nil {
			logIt("Couldn't send file back to " + ip + " because ")
			logIt(err.Error())
		}
	}
	if (os.Args[2] == "1") {
		directoryName := "shared/SDFS/"
		dir, err := ioutil.ReadDir(directoryName)
		if err != nil {
			fmt.Println(err)
		} else {
			for _, d := range dir {
				os.RemoveAll(path.Join([]string{directoryName, d.Name()}...))
			}
		}
	}
}




// Main thread of execution
// go run worker.go wordCount 8090
func main() {

	//starting up the mp3 subsystem
	dfsinterface.FileSystemRun(uiChan) //sets up everything including the MeNode variable in node.go

	go handleMembershipRequests()
	go handleSDFSRequests(uiChan)

	//Create a hashmap
	m = make(map[string]int)

	// Check commandline arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run worker.go operation delete_after(0,1)")
		return
	}

	fmt.Println(len(os.Args))
	fmt.Println(os.Args[1])

	fmt.Println("Got the args")
	// Listen to Master on some port infinitely
	logIt("Data input listening on "+ OPERATION_PORT)

	// listen on all interfaces //Uncomment if not debugging
	ln, _ := net.Listen("tcp", ":" + OPERATION_PORT)

	fmt.Println("Going in infinite loop")

	for { //accept connections

		//Uncomment the next line if not deubgging
		// message := "127.0.0.1,shared/shardedMapInputData/wordCount.txt,shared/shardedMapOutputData/xaa "
		//MasterIP, InputFilePath with name, OutputfilePath with name
		//Comment the next four lines if debugging
		conn, _ := ln.Accept()
		logIt("I did Listen")

		message, err := bufio.NewReader(conn).ReadString('\n')
		logIt("Message Received:"+ string(message))

		if message == "" || message == " " || err != nil {
			continue
		}


		//Create a hashmap
		m = make(map[string]int)

		if os.Args[1] == "wordCount" {
			//fmt.Println("Calling wordCount() on the server")
			fmt.Println("Handeling wordCount")
			handleWordCount(message) //Add go if not debugging
			// break                    //Comment if not debugging
		} else if os.Args[1] == "cumtd" {
			handleCUMTD(message) //Add go if not debugging
			// break                   //Comment if not debugging
		} else if os.Args[1] == "reverse" {
			fmt.Println("Handeling reverse")
			message = trimWhitespaceAndNewlineFeedFromString(message)

			s := strings.Split(message, ",")


			if s[3] == "map" {
				//fmt.Println("Call Map")
				handleReverseWebLinkForMap(message) //Add go if not debugging
			} else {
				//Create a hashmap
				mForReverse = make(map[string]string)
				handleReverseWebLinkForReduce(message) //Add go if not debugging
			}
		} else {
			fmt.Println("Wrong Operation")
			break
		}
	}
}



