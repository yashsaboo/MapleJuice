package node

import (
	"errors"
	//"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
)

// AnnounceDelete accounces that a file is deleted
func (node *ThisNode) AnnounceDelete(name string) error {

	// Create Message
	message := "DELETE," + strconv.FormatUint(node.NodeID, 10) + "," + node.Hostname + "," + name // Construct the delete message

	// Send Message to all other nodes including yourself
	for _, member := range node.Members.Members {
		connection, err := net.DialUDP("udp", nil, member.UDPAddr) // Connect to and send the message
		defer connection.Close()
		if err != nil {
			node.Logger.Print("Could not dial destination of DELETE message: ")
			node.Logger.Println(err)
			return err
		}
		connection.Write([]byte(message))
	}
	return nil
}

// HandleDelete handles a delete message that someone sent us
func (node *ThisNode) HandleDelete(name string) error {

	dir, _ := os.Getwd()
	fileName := dir+"/shared/"+name
	exec.Command("rm", fileName).Output() // Try to delete.

	// Delete from global file list
	for i, file := range node.Files.Files {
		if file.SDFSName == name {
			// Remove file from global list
			node.Files.L.Lock()
			node.Files.Files = append(node.Files.Files[:i], node.Files.Files[i+1:]...)
			node.Files.L.Unlock()
		}
	}
	return nil
}

// DeleteAllFiles deletes all files in the /shared directory upon startup/rejoin
func (node *ThisNode) DeleteAllFiles() error {

	// Golang directory delete referenced from https://www.dotnetperls.com/os-remove-go
	dir, _ := os.Getwd()
	fileName := dir+"/shared/"

	read, err := os.Open(fileName)
	if err != nil {
		return errors.New("Couldn't open directory " + err.Error())
	}
	files, err := read.Readdir(0)
	if err != nil {
		return errors.New("Couldn't read directory " + err.Error())
	}
	for idx := range files {
		curFile := files[idx]
		path := fileName + curFile.Name()
		err = os.Remove(path)
		if err != nil {
			return errors.New("Couldn't remove file " + path + " " + err.Error())
		}
	}
	return nil
}
