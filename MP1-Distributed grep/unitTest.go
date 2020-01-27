package main

// unitTest.go is the program that is run to test the overall architecture
// of the client server programs

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {

	// Check commandline arguments
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run unitTest.go FileNameWithoutVMIndex")
		return
	}

	// A list of querying patterns to check if the whole architecture works correctly
	// based on different number of returned entries and different returning machines
	regexPatterns := [6]string{"Keynes", "Gandhi", "Bismarck", "Hagel", "T[.]R[.]", "King jr[.]"}
	regexPatternPrint := [6]string{"Frequent", "Somewhat Frequent", "Rare", "Appears on all VMs", "Appears on Odd numbered VMs", "Appears only on VM number 3"}

	for i := 0; i < 6; i++ {
		fmt.Println(regexPatternPrint[i])

		// Get the grep results from client.go
		var resultFromClient string
		var listOfActiveVMs []int
		getResultFromClient(regexPatterns[i], &resultFromClient, &listOfActiveVMs)
		fmt.Println("Result from Client" + resultFromClient)

		// Get the ground truth by GNU grep command
		var resultFromLocalGreps string
		getResultFromLocalGreps(regexPatterns[i], os.Args[1], listOfActiveVMs, &resultFromLocalGreps)
		fmt.Println("Result from LocalGreps" + resultFromLocalGreps)

		// Verify if they are the same
		var unitTestPassedOrNot bool
		unitTestPassedOrNot = didUnitTestPassorNot(resultFromClient, resultFromLocalGreps)

		if unitTestPassedOrNot {
			fmt.Println("Unit Test Passed")
		} else {
			fmt.Println("Unit Test Not Passed")
		}
	}

}

// Call grep for local log files and construct the results to match the format
// of those printed by client.go
func getResultFromLocalGreps(regexPattern string, fileNameWithoutVM string, listOfActiveVMs []int, resultFromLocalGreps *string) error {

	*resultFromLocalGreps = ""

	for i := 0; i < len(listOfActiveVMs); i++ {
		// Prepare for command execution
		app := "grep"
		arg0 := "-n"
		arg1 := "-E"
		arg2 := regexPattern
		arg3 := fileNameWithoutVM + strconv.Itoa(listOfActiveVMs[i]) + ".log" // FileName will have the index of VM attached to it in the end

		// Run command
		rst := exec.Command(app, arg0, arg1, arg2, arg3)
		stdout, err := rst.Output()

		// If err != nil, it can either be error or no matched lines
		if err != nil {
			fmt.Println("Error occurred when greping file")
		} else {
			// Append filename to each entry to match those returned by client.go
			var tmpStr strings.Builder
			for _, entry := range strings.Split(string(stdout), "\n") {
				if len(entry) != 0 {
					tmpStr.WriteString(fileNameWithoutVM + strconv.Itoa(listOfActiveVMs[i]) + ".log" + " line " + entry + "\n")
				}
			}

			// Convert string builder to string
			*resultFromLocalGreps += tmpStr.String()
		}
	}
	return nil
}

// Call client.go to see the grep result given by the distributed grep architecture
func getResultFromClient(regexPattern string, resultFromClient *string, listOfActiveVMs *[]int) error {

	// Prepare command execution
	cmd := "go"
	arg0 := "run"
	arg1 := "client.go"
	arg2 := regexPattern

	// Run command
	rst := exec.Command(cmd, arg0, arg1, arg2)
	stdout, err := rst.Output()

	// Deal with client.go call error
	if err != nil {
		fmt.Println("Error occurred when calling client.go")
		return err
	}

	// Parse client.go output to see which servers are available
	// Here we use the line count part of the client program output to see
	// the servers that actually reponded and discard the lines afterwards
	*resultFromClient = string(stdout)
	lineCnt := 0
	startIndx := 0

	// Search in reverse order since the line counts are at the end of output
	for i := len(*resultFromClient) - 1; i >= 0; i-- {
		if (*resultFromClient)[i] == '\n' {
			lineCnt++
		}

		// Found start of target line count portion
		// All grep results has at least 11 lines, meaning that finding 12 \n
		// or reaching start before finding 12 is enough
		if lineCnt == 12 || i == 0 {

			// The 12th newline char is not our target
			if lineCnt == 12 {
				startIndx = i + 1
			}

			// Parse each line to see if machine is available
			for indx, str := range strings.Split((*resultFromClient)[startIndx:], "\n") {
				// Last line not needed
				if indx > 9 {
					break
				}

				// Not "Warning" starting line means that the server is up
				if str[0] != 'W' {
					*listOfActiveVMs = append(*listOfActiveVMs, indx+1)
				}
			}

			// Found all machines available
			break
		}
	}

	// Remove line count part
	*resultFromClient = (*resultFromClient)[:startIndx]

	return nil
}

// Check if two strings are the same
func didUnitTestPassorNot(resultFromClient string, resultFromLocalGreps string) bool {

	if resultFromClient == resultFromLocalGreps {
		return true
	} else {
		return false
	}
}
