package node

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"sort"
	"strings"
	"time"
)

// HashRingSize is how big our hashring is
const HashRingSize = 4294967296

// LogLocation is where the log is located
const LogLocation = "./out/machine.%d.log"

// HeartbeatFrequencyMilliseconds is how often we send heartbeats
const HeartbeatFrequencyMilliseconds = 1000

// MessageCacheFlushIntervalSeconds is how often to flush the message cache
const MessageCacheFlushIntervalSeconds = 60

// MessageCacheMaxAge is how long we keep messages
const MessageCacheMaxAge = 60

// FailureTimeoutSeconds is how long we wait until a node fails
const FailureTimeoutSeconds = 3

// FileListRefreshSeconds is how long we wait to refresh the file list
const FileListRefreshSeconds = 5

// FailureCheckFrequencyMilliseconds is often we check for failures
const FailureCheckFrequencyMilliseconds = 1000

// UDPPort is the port our protocol uses for membership
var UDPPort = 31337

// IntroPort is used for introduction
var IntroPort = 33333

// GetResponsibleNodes finds nodes that are responsible for this file by name
func (node *ThisNode) GetResponsibleNodes(filename string) []OtherNode {

	filename = strings.ReplaceAll(filename, "/", "^")

	h := sha1.New()
	h.Write([]byte(filename))
	hash := binary.BigEndian.Uint64(h.Sum(nil))

	//fmt.Println("Hash: %d", hash)

	return node.GetResponsibleNodesByHash(hash)
}

// GetResponsibleNodesByHash finds responsible nodes by hash
func (node *ThisNode) GetResponsibleNodesByHash(targetID uint64) []OtherNode {

	// Get sorted list of current IDs in place
	ids := []uint64{}
	for _, member := range node.Members.Members {
		ids = append(ids, member.NodeID)
	}
	sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
	//fmt.Println("++++", targetID)
	//fmt.Println(ids)

	// Fill up from the ring
	responsibleIDs := []uint64{}
	for _, id := range ids {
		if id >= targetID {
			responsibleIDs = append(responsibleIDs, id)
		}
		if len(responsibleIDs) == 4 {
			break
		}
	}

	// Wrap around the ring as much as necessary to fill it out
	i := 0
	for len(responsibleIDs) < 4 {
		responsibleIDs = append(responsibleIDs, ids[i])
		i++
		if i >= len(ids) {
			i = 0
		}
	}

	//fmt.Println("After the normal traversal: ", responsibleIDs)

	// Grab the OtherNode structs corresponding to the responsible Nodes
	answer := []OtherNode{}
	for _, responsibleID := range responsibleIDs {
		for _, member := range node.Members.Members {
			if member.NodeID == responsibleID {
				answer = append(answer, member)
			}
		}
	}

	//fmt.Println("Answer: ", responsibleIDs)
	return answer
}

// GetQuorumSize calculates the size of the quorum for our current network
func (node *ThisNode) GetQuorumSize() int {
	numMembers := len(node.Members.Members)
	responsesNeeded := math.Ceil(float64((numMembers + 1) / 2))
	if responsesNeeded > 3 {
		responsesNeeded = 3
	}
	return int(responsesNeeded)
}

// ListLocalFiles lists local files we have
func (node *ThisNode) ListLocalFiles() (answer []FileEntry) {
	localFiles, _ := ioutil.ReadDir("/shared")
	for _, fileInfo := range node.Files.Files {

		responsible := false
		responsibleNodes := node.GetResponsibleNodes(fileInfo.SDFSName)
		for _, rnode := range responsibleNodes {
			if node.NodeID == rnode.NodeID {
				responsible = true
			}
		}
		if responsible == false {
			continue
		}
		// List it if it is in the directory and we are supposed to have it
		for _, file := range localFiles {
			if fileInfo.SDFSName == file.Name() {
				answer = append(answer, fileInfo)
			}
		}
	}
	return
}

// IsConflicted determines if there was a previous write within 60 seconds
func (node *ThisNode) IsConflicted(filename string) bool {

	//Check if there was a write to this file within 60 seconds ago
	for _, fileInfo := range node.Files.Files {
		if fileInfo.SDFSName == filename {
			timeNow := time.Now().UnixNano() / 1000000
			fmt.Printf("Checking for file conflicts. Time now: %d, last edit time: %d\n", timeNow, fileInfo.TimeAdded)
			if timeNow-fileInfo.TimeAdded < 60000 {
				return true
			}
		}
	}
	return false
}
