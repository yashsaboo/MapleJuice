package node

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GetFile will get a file
func (node *ThisNode) GetFile(name, localPath string) {

	// First make sure this file actually exists
	found := false
	for _, file := range node.Files.Files {
		if strings.Compare(file.SDFSName, name) == 0 {
			found = true
		}
	}

	if found == false {
		fmt.Println("Error! File " + name + " does not exist.")
		return
	}

	// Figure out which nodes should have it
	responsibleNodes := node.GetResponsibleNodes(name)

	// Generate a unique identifier
	requestID := rand.Uint64()

	// Add this to the list of file gets in progress
	node.GetsInProgress.L.Lock()

	// Timeout search after 5 seconds
	timeout := time.Now().Add(5 * time.Second)

	// Label the search
	label := OngoingNeedLabel{
		ID:        requestID,
		Timeout:   timeout,
		LocalPath: localPath,
	}

	// Initialize the search
	node.GetsInProgress.Labels = append(node.GetsInProgress.Labels, label)
	node.GetsInProgress.Needs = append(node.GetsInProgress.Needs, []PastHaveMessage{})
	node.GetsInProgress.L.Unlock()

	// Fire off requests
	for _, responsibleNode := range responsibleNodes {
		go node.SendNeed(responsibleNode.Hostname, name, requestID)
	}
}

// SendNeed tells another nods that we need a file
func (node *ThisNode) SendNeed(host, name string, requestID uint64) error {

	// Hash filename
	h := sha1.New()
	h.Write([]byte(name))
	hash := binary.BigEndian.Uint64(h.Sum(nil))

	var n OtherNode
	for _, member := range node.Members.Members {
		if host == member.Hostname {
			n = member // Send it to the right node
		}
	}

	// Create Message
	message := "NEED," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname + "," + name + "," + strconv.FormatUint(hash, 10) + "," + strconv.FormatUint(requestID, 10)

	// connect
	connection, err := net.DialUDP("udp", nil, n.UDPAddr)
	defer connection.Close()
	if err != nil {
		node.Logger.Print("Could not dial destination of NEED message: ")
		node.Logger.Println(err)
		return err
	}

	// Send the message
	connection.Write([]byte(message))
	return nil
}

// HandleNeed handles a need message
func (node *ThisNode) HandleNeed(name string, requestID uint64, neederHost string) error {

	var needer OtherNode
	for _, member := range node.Members.Members {
		if neederHost == member.Hostname {
			needer = member
		}
	}

	for _, file := range node.Files.Files {

		// Do I have it?
		if file.SDFSName == name {

			// Create Message
			message := "HAVE," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname + "," + name + "," + strconv.FormatUint(requestID, 10) + "," + strconv.FormatInt(file.TimeAdded, 10)

			// connect
			connection, err := net.DialUDP("udp", nil, needer.UDPAddr)
			defer connection.Close()
			if err != nil {
				node.Logger.Print("Could not dial destination of HAVE message: ")
				node.Logger.Println(err)
				return err
			}

			// Send the message
			connection.Write([]byte(message))

		}
	}
	return nil
}

// HandleHave handles a have message
func (node *ThisNode) HandleHave(hostname, name string, timestamp int64, requestID uint64) {

	// Lock the Ongoing Needs
	node.GetsInProgress.L.Lock()

	// Defer releasing the Lock
	defer node.GetsInProgress.L.Unlock()

	// Loop through the labels to find old, failed gets
	timedOut := []int{}
	for i, label := range node.GetsInProgress.Labels {
		if time.Now().After(label.Timeout) {
			timedOut = append(timedOut, i)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(timedOut)))

	// Remove expired things
	for _, expired := range timedOut {
		//TODO log the failure of the GET
		node.Logger.Println("Get Timed Out")
		node.GetsInProgress.Labels = append(node.GetsInProgress.Labels[:expired], node.GetsInProgress.Labels[expired+1:]...)
		node.GetsInProgress.Needs = append(node.GetsInProgress.Needs[:expired], node.GetsInProgress.Needs[expired+1:]...)
	}

	for i, label := range node.GetsInProgress.Labels {
		if label.ID == requestID {

			// Struct to hold the relevant info
			newHave := PastHaveMessage{
				Filename:  name,
				Timestamp: timestamp,
				Hostname:  hostname,
			}

			// Add the node to the search
			node.GetsInProgress.Needs[i] = append(node.GetsInProgress.Needs[i], newHave)

			// If the response list is a quorum, delete it and its label and get the file
			if len(node.GetsInProgress.Needs[i]) >= node.GetQuorumSize() {

				//TODO Log the success of the GET
				node.Logger.Println("Succeeded in getting " + newHave.Filename)

				// Get most up-to-date reply
				maxTime := int64(0)
				maxHost := ""
				for _, version := range node.GetsInProgress.Needs[i] {
					if version.Timestamp > maxTime {
						maxHost = version.Hostname
						maxTime = version.Timestamp
					}
				}
				localPath := node.GetsInProgress.Labels[i].LocalPath

				// Do the rSync
				go node.RSyncFetch(name, localPath, maxHost)

				// Cleanup Gets
				node.GetsInProgress.Labels = append(node.GetsInProgress.Labels[:i], node.GetsInProgress.Labels[i+1:]...)
				node.GetsInProgress.Needs = append(node.GetsInProgress.Needs[:i], node.GetsInProgress.Needs[i+1:]...)
			}
		}
	}
}
