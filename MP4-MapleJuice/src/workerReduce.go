package main

import (
	"bufio"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"

	// "encoding/json"
	"fmt"
	// "log"
	// "net"
	"os"
	// "strings"
)

var m map[string]int

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
	if err != nil {
		fmt.Print("Couldn't open MapInputFile because")
		fmt.Println(err)
	} else {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			line = trimWhitespaceAndNewlineFeedFromString(line)

			s := strings.Split(line, ",")

			z, _ := strconv.Atoi(s[1])

			i, ok := m[s[0]] //Checks for the word in hashmap. If present, then i stores the current value and ok holds true bool value, else, false value and i=0
			if ok == true {
				fmt.Println(s[0])
				m[s[0]] = i + z //If value already present, then just increment the count
			} else {
				m[s[0]] = z //If value not present, then initilialise it to 1
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

	//Iterate over hashmpa and flush each key,value to file
	for key, value := range m {
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

	//Flush the Hashmap to the reduceoutput
	err := flushHashMaptoFile(s[2])
	if err != nil {
		fmt.Println("Could not write to file: " + s[2])
	} else {
		//SCP the file to shardedReduceOutputData/ folder on Master Node
		//TODO
	}
}

func handleCUMTD(message string) {

}

// Main thread of execution
// go run workerReduce.go wordCount 8090
func main() {

	fmt.Println(len(os.Args))
	fmt.Println(os.Args[1])

	// Check commandline arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run workerReduce.go operation portNumber")
		return
	}

	fmt.Println("Got the args")

	// Listen to Master on some port infinitely
	fmt.Println("Filesystem Network listening on ", os.Args[2])

	// listen on all interfaces //Uncomment if not debugging
	ln, _ := net.Listen("tcp", ":"+os.Args[2])

	fmt.Println("Going in infinite loop")

	for { //accept connections

		//Create a hashmap
		m = make(map[string]int)

		//Uncomment the next line if not deubgging
		// message := "127.0.0.1,shared/shardedReduceInputData/1,shared/shardedReduceOutputData/1 "
		//MasterIP, InputFilePath with name, OutputfilePath with name
		//Comment the next four lines if debugging
		conn, _ := ln.Accept()
		fmt.Println("I did Listen")
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Println("Message Received:", string(message))

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
