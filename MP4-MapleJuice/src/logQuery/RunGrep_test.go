package logQuery

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test that we can read the nodes file and get the proper number of nodes
func TestReadFile(t *testing.T) {
	var response []NodeInfo
	response, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	if len(response) != 10 {
		t.Error(err)
	}

}

// Test that we can grep on all 10 nodes for a common query
func TestGrepCommon(t *testing.T) {
	var nodes []NodeInfo
	query := []string{"-nHE", `'^[0-9]*[a-z]{5}'`} // Common query with ~4000 responses per file
	nodes, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	// Capture output of function to check for correctness
	newFile := filepath.Join(os.TempDir(), "stdout")
	oldOut := os.Stdout
	temp, err := os.Create(newFile)
	if err != nil {
		t.Error(err)
	}
	os.Stdout = temp

	err = DistributedGrep(query, nodes, "./log/vm%d.log") // Run query on all 10 nodes

	temp.Close() // Return stdout back to normal
	os.Stdout = oldOut
	newOut, err := ioutil.ReadFile(newFile)
	if err != nil {
		t.Error(err)
	}
	badNodes := 0 // This will check that we got correct results from all 10 nodes
	if strings.Contains(string(newOut), "Node 1 results count: 4102") != true {
		fmt.Println("VM1 bad common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 2 results count: 4012") != true {
		fmt.Println("VM2 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 3 results count: 4154") != true {
		fmt.Println("VM3 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 4 results count: 4246") != true {
		fmt.Println("VM4 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 5 results count: 4130") != true {
		fmt.Println("VM5 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 6 results count: 4165") != true {
		fmt.Println("VM6 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 7 results count: 4083") != true {
		fmt.Println("VM7 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 8 results count: 4211") != true {
		fmt.Println("VM8 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 9 results count: 4069") != true {
		fmt.Println("VM9 bad common")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 10 results count: 4075") != true {
		fmt.Println("VM10 bad common")
		badNodes++
	}

	if err != nil {
		t.Error(err)
	}

	if badNodes != 0 {
		t.Error(errors.New("All 10 nodes did not respond correctly"))
	}
}

// Test that we can grep on all 10 nodes for an uncommon query. Same structure as TestGrepCommon
func TestGrepUncommon(t *testing.T) {
	var nodes []NodeInfo
	query := []string{"-nHE", "asd"}
	nodes, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	// Capture output of function to check for correctness
	newFile := filepath.Join(os.TempDir(), "stdout")
	oldOut := os.Stdout
	temp, err := os.Create(newFile)
	if err != nil {
		t.Error(err)
	}
	os.Stdout = temp

	err = DistributedGrep(query, nodes, "./log/vm%d.log")

	temp.Close()
	os.Stdout = oldOut
	newOut, err := ioutil.ReadFile(newFile)
	if err != nil {
		t.Error(err)
	}

	badNodes := 0
	if strings.Contains(string(newOut), "Node 1 results count: 27") != true {
		fmt.Println("VM1 bad uncommon")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 2 results count: 32") != true {
		fmt.Println("VM2 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 3 results count: 23") != true {
		fmt.Println("VM3 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 4 results count: 20") != true {
		fmt.Println("VM4 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 5 results count: 23") != true {
		fmt.Println("VM5 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 6 results count: 18") != true {
		fmt.Println("VM6 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 7 results count: 23") != true {
		fmt.Println("VM7 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 8 results count: 29") != true {
		fmt.Println("VM8 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 9 results count: 25") != true {
		fmt.Println("VM9 bad uncommon")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 10 results count: 23") != true {
		fmt.Println("VM10 bad uncommon")
		badNodes++
	}

	if err != nil {
		t.Error(err)
	}

	if badNodes != 0 {
		t.Error(errors.New("All 10 nodes did not respond correctly"))
	}
}

// Test that we can grep on all 10 nodes for an uncommon query. Same structure as TestGrepCommon
func TestGrepRare(t *testing.T) {
	var nodes []NodeInfo
	query := []string{"-nHE", "j8U5QU9Ttasdz65Zdodj4Q"}
	nodes, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	// Capture output of function to check for correctness
	newFile := filepath.Join(os.TempDir(), "stdout")
	oldOut := os.Stdout
	temp, err := os.Create(newFile)
	if err != nil {
		t.Error(err)
	}
	os.Stdout = temp

	err = DistributedGrep(query, nodes, "./log/vm%d.log")

	temp.Close()
	os.Stdout = oldOut
	newOut, err := ioutil.ReadFile(newFile)
	if err != nil {
		t.Error(err)
	}

	badNodes := 0
	if strings.Contains(string(newOut), "Node 1 results count: 0") != true {
		fmt.Println("VM1 bad rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 2 results count: 0") != true {
		fmt.Println("VM2 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 3 results count: 1") != true {
		fmt.Println("VM3 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 4 results count: 0") != true {
		fmt.Println("VM4 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 5 results count: 0") != true {
		fmt.Println("VM5 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 6 results count: 0") != true {
		fmt.Println("VM6 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 7 results count: 0") != true {
		fmt.Println("VM7 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 8 results count: 1") != true {
		fmt.Println("VM8 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 9 results count: 0") != true {
		fmt.Println("VM9 bad rare")
		badNodes++
	}

	if strings.Contains(string(newOut), "Node 10 results count: 0") != true {
		fmt.Println("VM10 bad rare")
		badNodes++
	}

	if err != nil {
		t.Error(err)
	}

	if badNodes != 0 {
		t.Error(errors.New("All 10 nodes did not respond correctly"))
	}
}

func TestLocalLogsRare(t *testing.T) {
	var nodes []NodeInfo
	query := []string{"-nHE", "Rare"}
	nodes, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	// Capture output of function to check for correctness
	newFile := filepath.Join(os.TempDir(), "stdout")
	oldOut := os.Stdout
	temp, err := os.Create(newFile)
	if err != nil {
		t.Error(err)
	}
	os.Stdout = temp

	err = DistributedGrep(query, nodes, "./log/test%d.log")

	temp.Close()
	os.Stdout = oldOut
	newOut, err := ioutil.ReadFile(newFile)
	if err != nil {
		t.Error(err)
	}

	badNodes := 0
	if strings.Contains(string(newOut), "Node 1 results count: 1") != true {
		fmt.Println("VM1 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 2 results count: 2") != true {
		fmt.Println("VM2 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 3 results count: 3") != true {
		fmt.Println("VM3 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 4 results count: 4") != true {
		fmt.Println("VM4 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 5 results count: 5") != true {
		fmt.Println("VM5 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 6 results count: 6") != true {
		fmt.Println("VM6 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 7 results count: 7") != true {
		fmt.Println("VM7 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 8 results count: 8") != true {
		fmt.Println("VM8 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 9 results count: 9") != true {
		fmt.Println("VM9 bad local rare")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 10 results count: 10") != true {
		fmt.Println("VM10 bad local rare")
		badNodes++
	}
	if badNodes != 0 {
		t.Error(errors.New("All 10 nodes did not respond correctly"))
	}

}

func TestLocalLogsCommon(t *testing.T) {
	var nodes []NodeInfo
	query := []string{"-nHE", "Common"}
	nodes, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	// Capture output of function to check for correctness
	newFile := filepath.Join(os.TempDir(), "stdout")
	oldOut := os.Stdout
	temp, err := os.Create(newFile)
	if err != nil {
		t.Error(err)
	}
	os.Stdout = temp

	err = DistributedGrep(query, nodes, "./log/test%d.log")

	temp.Close()
	os.Stdout = oldOut
	newOut, err := ioutil.ReadFile(newFile)
	if err != nil {
		t.Error(err)
	}

	badNodes := 0
	if strings.Contains(string(newOut), "Node 1 results count: 1000") != true {
		fmt.Println("VM1 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 2 results count: 2000") != true {
		fmt.Println("VM2 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 3 results count: 3000") != true {
		fmt.Println("VM3 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 4 results count: 4000") != true {
		fmt.Println("VM4 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 5 results count: 5000") != true {
		fmt.Println("VM5 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 6 results count: 6000") != true {
		fmt.Println("VM6 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 7 results count: 7000") != true {
		fmt.Println("VM7 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 8 results count: 8000") != true {
		fmt.Println("VM8 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 9 results count: 9000") != true {
		fmt.Println("VM9 bad local Common")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 10 results count: 10000") != true {
		fmt.Println("VM10 bad local Common")
		badNodes++
	}
	if badNodes != 0 {
		t.Error(errors.New("All 10 nodes did not respond correctly"))
	}

}
func TestLocalLogsMiddle(t *testing.T) {
	var nodes []NodeInfo
	query := []string{"-nHE", "Middle"}
	nodes, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	// Capture output of function to check for correctness
	newFile := filepath.Join(os.TempDir(), "stdout")
	oldOut := os.Stdout
	temp, err := os.Create(newFile)
	if err != nil {
		t.Error(err)
	}
	os.Stdout = temp

	err = DistributedGrep(query, nodes, "./log/test%d.log")

	temp.Close()
	os.Stdout = oldOut
	newOut, err := ioutil.ReadFile(newFile)
	if err != nil {
		t.Error(err)
	}

	badNodes := 0
	if strings.Contains(string(newOut), "Node 1 results count: 10") != true {
		fmt.Println("VM1 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 2 results count: 20") != true {
		fmt.Println("VM2 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 3 results count: 30") != true {
		fmt.Println("VM3 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 4 results count: 40") != true {
		fmt.Println("VM4 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 5 results count: 50") != true {
		fmt.Println("VM5 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 6 results count: 60") != true {
		fmt.Println("VM6 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 7 results count: 70") != true {
		fmt.Println("VM7 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 8 results count: 80") != true {
		fmt.Println("VM8 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 9 results count: 90") != true {
		fmt.Println("VM9 bad local Middle")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 10 results count: 100") != true {
		fmt.Println("VM10 bad local Middle")
		badNodes++
	}
	if badNodes != 0 {
		t.Error(errors.New("All 10 nodes did not respond correctly"))
	}

}

func TestLocalLogsTopOnly(t *testing.T) {
	var nodes []NodeInfo
	query := []string{"-nHE", "TopOnly"}
	nodes, err := ReadNodeAddrFile("../nodes.txt")
	if err != nil {
		t.Error(err)
	}

	// Capture output of function to check for correctness
	newFile := filepath.Join(os.TempDir(), "stdout")
	oldOut := os.Stdout
	temp, err := os.Create(newFile)
	if err != nil {
		t.Error(err)
	}
	os.Stdout = temp

	err = DistributedGrep(query, nodes, "./log/test%d.log")

	temp.Close()
	os.Stdout = oldOut
	newOut, err := ioutil.ReadFile(newFile)
	if err != nil {
		t.Error(err)
	}

	badNodes := 0
	if strings.Contains(string(newOut), "Node 1 results count: 0") != true {
		fmt.Println("VM1 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 2 results count: 0") != true {
		fmt.Println("VM2 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 3 results count: 0") != true {
		fmt.Println("VM3 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 4 results count: 0") != true {
		fmt.Println("VM4 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 5 results count: 0") != true {
		fmt.Println("VM5 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 6 results count: 0") != true {
		fmt.Println("VM6 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 7 results count: 0") != true {
		fmt.Println("VM7 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 8 results count: 0") != true {
		fmt.Println("VM8 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 9 results count: 9") != true {
		fmt.Println("VM9 bad local TopOnly")
		badNodes++
	}
	if strings.Contains(string(newOut), "Node 10 results count: 10") != true {
		fmt.Println("VM10 bad local TopOnly")
		badNodes++
	}
	if badNodes != 0 {
		t.Error(errors.New("All 10 nodes did not respond correctly"))
	}

}
