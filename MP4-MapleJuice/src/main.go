package main

import (
	"os"
	"dfsinterface"
	"membership/node"
	"os/signal"
	"fmt"
	"time"
)

/*
//useful reference
func (m *MembershipList) Sort() {
	m.L.Lock()
	defer m.L.Unlock()
	sort.Slice(m.Members[:], func(i, j int) bool {
		return m.Members[i].NodeID < m.Members[j].NodeID
	})
}*/


func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	uiChan := make(chan string)
	dfsinterface.FileSystemRun(uiChan) //sets up everything including the MeNode variable in node.go

	//can now call functions that use GetAliveIPs
	fmt.Println("sleeping for 5 seconds")
	time.Sleep(5 * time.Second)
	fmt.Println("Done sleeping")
	for _, hostname := range(GetAliveIPs()) {
		fmt.Println(hostname)
	}

	dir, _ := os.Getwd()
	fileName := dir+"/shared/"+"example.txt"
	fmt.Println(fileName)

	//waiting for the process to be killed
	
	select { // Catches interrupts so we can end the program
	case <-interrupt:
		//UniversalLog.Printf("@ Node Terminated Normally @")
		fmt.Printf("Got an interupt message")
		os.Exit(0)
	}
}



func GetAliveIPs() []string{
	/*1. get membership list from ThisNode var stored in node.go
	2. lock the membership list
	3. copy the ips from the membership list of the nodes that are alive
	4. unlock the membership list
	5 return the ip list*/
	node.MeNode.Members.L.Lock()
	defer node.MeNode.Members.L.Unlock()


	var ipList []string
	for _, memberNode := range(node.MeNode.Members.Members) {
		ipString := memberNode.UDPAddr.IP.String()
		ipList = append(ipList, ipString)
	}
	return ipList
}



