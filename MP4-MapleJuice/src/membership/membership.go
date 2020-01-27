package membership

// This file contains the main function for the program. The node is initialized, the two UDP monitors are started,
// and the RPC server is started for MP1 querying

import (
	"fmt"
	"net"
	"strconv"

	"strings"
	"time"

	"membership/node"
)

// UDPPort for all nodes
var UDPPort = 31337

// IntroPort is used for introduction
var IntroPort = 33333

// HashRingSize is how big our hashring is
const HashRingSize = 4294967296

// StartNode starts the two listeners and handles membership messages
func StartNode(myNode *node.ThisNode) {

	go func() { // Introducer loop with boilerplate code for implementing a UDP listener in Go
		fmt.Println("Started intro listener")
		introAddr, err := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(IntroPort))
		if err != nil {
			myNode.Logger.Fatal("UDP Server Address Resolution Error:", err)
		}

		introListener, err := net.ListenUDP("udp4", introAddr)
		if err != nil {
			myNode.Logger.Fatal("UDP Server Listen Error:", err)
		}
		for {
			buffer := make([]byte, 2048)
			n, returnAddr, err := introListener.ReadFromUDP(buffer)

			if err != nil {
				myNode.Logger.Fatal("Error in UDP listener read: ", err)
			}

			message := string(buffer[:n])
			fmt.Printf("Message: %s\n", message)

			// Message Type, NodeID, Hostname, Message
			messageFields := strings.Split(message, ",")
			if len(messageFields) < 3 {
				continue
			}
			messageType := messageFields[0]

			incNodeID, err := strconv.ParseUint(messageFields[1], 10, 64)
			curTime := time.Now()

			baseMsg := &node.Message{
				NodeID:     incNodeID,
				T:          uint64(curTime.UnixNano() / 1000000),
				RemoteAddr: returnAddr,
				Hostname:   messageFields[2],
				UDPPort:    UDPPort,
				Orig:       message,
			}

			switch messageType {
			case "INTRO":
				introMsg := &node.IntroMessage{
					Message: baseMsg,
				}
				if err := myNode.HandleIntro(introMsg, returnAddr); err != nil {
					myNode.Logger.Println(err)
				}

			case "MEMBER":
				if err := myNode.HandleIntroList(message); err != nil {
					myNode.Logger.Println(err)
				}

			case "FILELIST":
				if err := myNode.HandleIntroFileList(message); err != nil {
					myNode.Logger.Println(err)
				}
			}
		}
	}()

	go func() { // Spawn UDP listener thread for all other messages
		udpAddr, err := net.ResolveUDPAddr("udp4", ":"+strconv.Itoa(UDPPort))
		if err != nil {
			myNode.Logger.Fatal("UDP Server Address Resolution Error:", err)
		}

		udpListener, err := net.ListenUDP("udp4", udpAddr)
		if err != nil {
			myNode.Logger.Fatal("UDP Server Listen Error:", err)
		}
		for {
			buffer := make([]byte, 2048)
			n, returnAddr, err := udpListener.ReadFromUDP(buffer)

			if err != nil {
				myNode.Logger.Fatal("Error in UDP listener read: ", err)
			}

			message := string(buffer[:n])

			// Message Type, NodeID, Hostname, Message
			messageFields := strings.Split(message, ",")
			messageType := messageFields[0]

			incNodeID, err := strconv.ParseUint(messageFields[1], 10, 64)
			curTime := time.Now()

			baseMsg := &node.Message{
				NodeID:     incNodeID,
				T:          uint64(curTime.UnixNano() / 1000000),
				RemoteAddr: returnAddr,
				Hostname:   messageFields[2],
				UDPPort:    UDPPort,
				Orig:       message,
			}
			if strings.Contains(messageType, "HEART") == false {
				fmt.Printf("Message: %s\n", message)

				// myNode.Logger.Printf("Message: %s\n", message)
				myNode.Logger.Printf("Message: %s\n", message)
			}

			switch messageType { // Messages are decoded and processed according to type
			case "INTRO":
				introMsg := &node.IntroMessage{
					Message: baseMsg,
				}
				fmt.Println("Got an INTRO on the wrong port!")
				if err := myNode.HandleIntro(introMsg, returnAddr); err != nil {
					myNode.Logger.Println(err)
				}
			case "HEART":
				heartMsg := &node.HeartMessage{
					Message: baseMsg,
				}
				if err := myNode.HandleHeartbeat(heartMsg); err != nil {
					myNode.Logger.Println(err)
				}
			case "JOIN":
				tcpPort, _ := strconv.Atoi(messageFields[3])
				joinMsg := &node.JoinMessage{
					Message: baseMsg,
					TCPPort: tcpPort,
				}
				if err := myNode.HandleJoinMsg(joinMsg); err != nil {
					myNode.Logger.Println(err)
				}
			case "LEAVE":
				leaveMsg := &node.LeaveMessage{
					Message: baseMsg,
				}
				if err := myNode.HandleLeave(leaveMsg); err != nil {
					myNode.Logger.Println(err)
				}
			case "FAIL":
				failMsg := &node.FailMessage{
					Message: baseMsg,
				}
				if err := myNode.HandleFailure(failMsg); err != nil {
					myNode.Logger.Println(err)
				}

			case "NEED":

				requestID, _ := strconv.ParseUint(messageFields[5], 10, 64)

				if err := myNode.HandleNeed(messageFields[3], requestID, messageFields[2]); err != nil {
					myNode.Logger.Println(err)
				}

			case "DELETE":
				if err := myNode.HandleDelete(messageFields[3]); err != nil {
					myNode.Logger.Println(err)
				}

			case "HAVE":
				timestamp, _ := strconv.ParseInt(messageFields[5], 10, 64)
				requestID, _ := strconv.ParseUint(messageFields[4], 10, 64)
				hostname := messageFields[2]
				filename := messageFields[3]
				myNode.HandleHave(hostname, filename, timestamp, requestID)

			case "NEWFILE":
				newTime, _ := strconv.ParseInt(messageFields[4], 10, 64)
				newfileMsg := &node.FileMessage{
					Message:  baseMsg,
					SDFSName: messageFields[3],
					Hash:     messageFields[5],
					Updated:  newTime,
				}
				fmt.Println(time.Now().Format("15:04:05.000"))
				myNode.HandleNewFile(newfileMsg)

			default:
				myNode.Logger.Printf("Unknown message type: " + messageType + " was ignored")
			}
		}
	}()
}
