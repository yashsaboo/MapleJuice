package node

import (
	"errors"
	"fmt"
	"net/rpc"
	"strconv"
)

// FileVersionAsk contains the filename
type FileVersionAsk struct {
	Filename string
	// Node     *ThisNode
}

// FileVersionAnswer says whether we have the file and what the timestamp is
type FileVersionAnswer struct {
	Have      bool
	Timestamp int64
}

// GetFileVersion gets the file version from another node
func (node *ThisNode) GetFileVersion(other OtherNode, filename string) (int64, bool) {

	// Dial peer node
	path := other.Hostname + ":" + strconv.Itoa(10000+other.TCPPort)
	// fmt.Println("Getting file version for file " + filename + " from node " + path)
	curClient, err := rpc.DialHTTP("tcp", path)
	if err != nil {
		fmt.Println("Error dialing RPC to get file version " + err.Error())
		return -1, false
	}
	defer curClient.Close()
	// Make RPC Call
	query := FileVersionAsk{
		Filename: filename,
		// Node:     node,
	}
	var response FileVersionAnswer
	err = curClient.Call("RPCNode.RPCGetFileVersion", &query, &response)
	if err != nil {
		fmt.Println("Error calling RPC to get file version " + err.Error())
		return -1, false
	}

	return response.Timestamp, response.Have
}

// RPCGetFileVersion processes an RPC call asking for the file version
func (*RPCNode) RPCGetFileVersion(args *FileVersionAsk, reply *FileVersionAnswer) error {
	// fmt.Println("Someone asked us to see if we have file " + args.Filename)
	reply.Have = false
	responsible := false
	responsibleNodes := MeNode.GetResponsibleNodes(args.Filename)
	for _, node := range responsibleNodes {
		if node.NodeID == MeNode.NodeID {
			responsible = true
		}
	}
	if responsible == false {
		return nil // We aren't actually responsible for this file
	}

	for _, file := range MeNode.Files.Files { // Check all files I know about
		if file.SDFSName == args.Filename {
			reply.Timestamp = file.TimeAdded
			localFiles := MeNode.ListLocalFiles()
			for _, localFile := range localFiles { // Check all files I am currently storing
				if localFile.SDFSName == args.Filename {
					reply.Have = true
				}
			}
			// fmt.Printf("All done checking. Did we have it: %t, Timestamp: %d\n", reply.Have, reply.Timestamp)
			return nil
		}
	}
	return errors.New("never heard of this")
}
