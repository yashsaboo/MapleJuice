package node

import (
	"fmt"
	"strings"
	"os"
)

// CheckFileStabilization periodically checks that we have the right files and that they are current
func (node *ThisNode) CheckFileStabilization() {

	// Loop through the global file list
	for _, file := range node.Files.Files {

		// For each, check who should have it
		responsibleNodes := node.GetResponsibleNodes(file.SDFSName)

		// Check if I should have it
		iShouldHaveIt := false
		myIndex := -1
		for i, responsibleNode := range responsibleNodes {
			if responsibleNode.NodeID == node.NodeID {
				iShouldHaveIt = true
				myIndex = i
			}
		}

		if iShouldHaveIt {
			// Check if anyone has a better version
			// Also handles the case where you don't have it at all

			// fmt.Println("I should have this file")

			timestamps := []int64{}
			for _, responsibleNode := range responsibleNodes {
				remoteTimestamp, has := node.GetFileVersion(responsibleNode, file.SDFSName)
				if !has {
					timestamps = append(timestamps, int64(-1))
				} else {
					timestamps = append(timestamps, remoteTimestamp)
				}
			}

			// get max timestamp
			maxTS := int64(0)
			maxI := -1
			for i, ts := range timestamps {
				if ts > maxTS {
					maxTS = ts
					maxI = i
				}
			}

			// If nobody has a copy
			if maxI == -1 {
				node.Logger.Println("Nobody has a copy of " + strings.ReplaceAll(file.SDFSName, "^", "/") + " anymore :(")
				node.AnnounceDelete(file.SDFSName)
				continue
			}

			// If we are already current
			if timestamps[myIndex] == maxTS {
				continue
			}

			// If someone else has a better copy
			file.TimeAdded = maxTS
			dir, _ := os.Getwd()
			fileName := dir+"/shared/"+file.SDFSName
			node.RSyncFetch(file.SDFSName, fileName, responsibleNodes[maxI].Hostname)
		}
	}
}

// ListFileLocations lists locations where a file is stored
func (node *ThisNode) ListFileLocations(filename string) (locations []OtherNode) {
	have := false
	for _, file := range node.Files.Files {
		if strings.Compare(filename, file.SDFSName) == 0 {
			have = true
		}
	}
	if have == false {
		fmt.Println("File " + strings.ReplaceAll(filename, "^", "/") + " does not exist")
		return
	}

	for _, member := range node.Members.Members {
		_, has := node.GetFileVersion(member, filename)
		if has {
			locations = append(locations, member)
		}
	}
	return
}
