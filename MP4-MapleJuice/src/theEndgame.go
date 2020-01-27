package main

import (
	"bufio"
	"io/ioutil"
	"strings"

	"fmt"
	"os"
	"sort"
	// "log"
	// "net"
	// "encoding/json"
)

// func combineFilesAndFlushItToOutput2() {

// 	reduceOutputFilesNames, _ := ioutil.ReadDir(os.Args[2])

// 	file, err := os.Create(os.Args[3])
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer file.Close()

// 	w := bufio.NewWriter(file)

// 	for _, fileName := range reduceOutputFilesNames {

// 		//Open the reduceOutput file
// 		file2, err2 := os.Open(fileName.Name())
// 		if err2 != nil {
// 			fmt.Println(err2)
// 		}
// 		defer file2.Close()

// 		scanner := bufio.NewScanner(file2)

// 		//Iterate over each line in mapOutput file
// 		for scanner.Scan() {
// 			line := scanner.Text()
// 			fmt.Fprintln(line)
// 		}
// 	}
// }

func combineFilesAndFlushItToOutput() {

	reduceOutputFilesNames, _ := ioutil.ReadDir(os.Args[2])

	var listOfLines []string

	for _, fileName := range reduceOutputFilesNames {

		//Open the reduceOutput file
		file2, err2 := os.Open(os.Args[2] + fileName.Name())
		if err2 != nil {
			fmt.Println(err2)
		}
		

		scanner := bufio.NewScanner(file2)

		//Iterate over each line in mapOutput file and store it in sorting variable
		for scanner.Scan() {
			line := scanner.Text()
			listOfLines = append(listOfLines, line)
		}
		file2.Close()
	}

	//Sort
	sort.Strings(listOfLines)

	file, err := os.Create(os.Args[3])
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)

	//Flush it to the file
	for _, line := range listOfLines {
		line = strings.ReplaceAll(line, ",", "\t")
		fmt.Fprintln(w, line)
	}
	w.Flush()
}

// func deleteSDFSFiles() {

// 	mapOutputFilesNames, _ := ioutil.ReadDir("shared/SDFS/")

// 	for _, fileName := range mapOutputFilesNames {

// 		//delete <SDFS_file_name>: delete fileName.Name()
// 		sdfs_command := "delete " + fileName.Name()
// 		uiChan <- sdfs_command //push the message into the command channel

// 	}
// }

// Main thread of execution
// go run theEndgame.go operation shared/shardedReduceOutputData/ shared/outputData/wordCountOutput.txt 0
func main() {
	/*for _, ip := range(FetchIPList()) {
		fmt.Println(ip)
	}*/

	fmt.Println(len(os.Args))
	fmt.Println(os.Args[2])

	// Check commandline arguments
	if len(os.Args) != 5 {
		fmt.Println("Usage: go run theEndgame.go operation reduceOutputFolder outputFileName deleteOrNot={0 (don't delete), 1(delete)}")
		return
	}

	//Combines Reduce Worker files into one into the location mentioned by the user
	combineFilesAndFlushItToOutput()

	//Delete SDFS Intermediate Files
	if os.Args[4] == "1" {
		// deleteSDFSFiles()
	}
}
