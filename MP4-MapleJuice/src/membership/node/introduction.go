package node

// This contains functions for handling the introduction process. Every node in the network can be an introducer.
import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// AskForIntroduction gets the membership list from the introducer
func (node *ThisNode) AskForIntroduction(IntroducerPort int, idx int) error {
	hostName := "" // Ability to ask any node in the network to be an introducer
	if idx < 10 {
		hostName = "fa19-cs425-g69-0" + strconv.Itoa(idx) + ".cs.illinois.edu"
	} else {
		hostName = "fa19-cs425-g69-" + strconv.Itoa(idx) + ".cs.illinois.edu"
	}

	port := strconv.Itoa(IntroducerPort)
	introAddr := hostName + ":" + port

	remote, err := net.ResolveUDPAddr("udp", introAddr) // Resolve introducer's UDP address

	if err != nil {
		node.Logger.Printf("Couldn't resolve address %s\n", introAddr)
		return err
	}

	connection, err := net.DialUDP("udp", nil, remote)

	if err != nil {
		node.Logger.Print("Could not dial destination of INTRO message: ")
		node.Logger.Println(err)
		return err
	}

	defer connection.Close()
	myHostname, err := os.Hostname()
	if err != nil {
		node.Logger.Fatal("Couldn't get my hostname")
	}
	introMsg := []byte("INTRO," + strconv.FormatUint(node.NodeID, 10) + "," + myHostname) // For the Intro message

	_, err = connection.Write(introMsg) // Send our intro message to the introducer
	if err != nil {
		node.Logger.Printf("Couldn't write introduction message: %s\n", introMsg)
		return err
	}

	return nil
}

// HandleIntroList processes introductions if someone asked this node for an introduction
func (node *ThisNode) HandleIntroList(msg string) error {
	var err error
	messages := strings.Split(msg, "\n") // Intro messages may contain many members. Split each one so we can add it individually

	for i := 0; i < len(messages); i++ {
		if len(messages[i]) < 5 {
			break
		}
		messageFields := strings.Split(messages[i], ",")
		var newNode OtherNode

		newNode.NodeID, _ = strconv.ParseUint(messageFields[1], 10, 64) // For a new node from the fields in the message
		newNode.Hostname = messageFields[2]
		newNode.TCPPort, _ = strconv.Atoi(messageFields[3])
		newNode.UDPPort, _ = strconv.Atoi(messageFields[4])

		introAddr := newNode.Hostname + ":" + strconv.Itoa(newNode.UDPPort)
		newNode.UDPAddr, err = net.ResolveUDPAddr("udp", introAddr)
		if err != nil {
			node.Logger.Print("Could not dial destination of INTRO message: ")
			node.Logger.Println(err)
			return err
		}

		if err := node.Members.Add(newNode); err != nil { // Add the node to our members list
			node.Logger.Print("Could not add node to member list ")
			node.Logger.Println(err)
		}

		node.Neighbors.Update(node.Members, node.NodeID) // Update our neighbor list

	}

	return nil
}

// HandleIntro processes introductions if this node is the introducer
func (node *ThisNode) HandleIntro(msg *IntroMessage, retAddr *net.UDPAddr) error {
	introAddr := msg.Hostname + ":" + strconv.Itoa(33333)

	remote, err := net.ResolveUDPAddr("udp", introAddr)

	connection, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		node.Logger.Print("Error dialing UDP in HandleIntro ")
		node.Logger.Println(err)
	}
	message := ""
	for i := 0; i < len(node.Members.Members); i++ {
		message += "MEMBER," + strconv.FormatUint(node.Members.Members[i].NodeID, 10) + "," +
			node.Members.Members[i].Hostname + "," +
			strconv.Itoa(node.Members.Members[i].TCPPort) + "," +
			strconv.Itoa(node.Members.Members[i].UDPPort) + "\n"
	}

	_, err = connection.Write([]byte(message))

	message = ""
	for i := 0; i < len(node.Files.Files); i++ {
		message += "FILELIST," + strconv.FormatUint(node.NodeID, 10) + "," +
			node.Hostname + "," +
			node.Files.Files[i].LocalName + "," +
			node.Files.Files[i].SDFSName + "," +
			strconv.FormatInt(node.Files.Files[i].TimeAdded, 10) + "," + "\n"
	}

	_, err = connection.Write([]byte(message))

	if err != nil {
		node.Logger.Print("Error sending membership list: ")
		node.Logger.Println(err)
	}

	return nil
}

// HandleIntroFileList processes an incoming file list
func (node *ThisNode) HandleIntroFileList(msg string) error {
	messages := strings.Split(msg, "\n") // Intro messages may contain many members. Split each one so we can add it individually

	for i := 0; i < len(messages); i++ {
		if len(messages[i]) < 6 {
			break
		}
		messageFields := strings.Split(messages[i], ",")
		var newFile FileEntry

		newFile.LocalName = messageFields[3]
		newFile.SDFSName = messageFields[4]
		fmt.Printf("Just learned about file %s\n", strings.ReplaceAll(messageFields[4], "^", "/"))
		newFile.TimeAdded, _ = strconv.ParseInt(messageFields[5], 10, 64)
		newFile.Hash = messageFields[6]
		added := false
		for _, file := range node.Files.Files {
			if strings.Compare(file.SDFSName, newFile.SDFSName) == 0 {
				added = true
			}
		}
		if added == false {
			node.Files.L.Lock()
			fmt.Printf("Adding file %s to list\n", strings.ReplaceAll(newFile.SDFSName, "^", "/"))
			node.Files.Files = append(node.Files.Files, newFile)
			node.Files.L.Unlock()

		}
	}

	return nil
}
