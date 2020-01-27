package main

import (
	"bufio"
	"bytes"
	"container/heap"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

////////////////////////////////////
//                                //
// Global variable and parameters //
//                                //
////////////////////////////////////

// Intro Param
const primIntroServerAddr = "fa19-cs425-g43-01.cs.illinois.edu"
const introAttempts = 3
const introTimeoutPeriod = 5

// For primary intro server to rejoin
const secIntroServerAddr = "fa19-cs425-g43-02.cs.illinois.edu"

// Gossip Param
const gossipC = 2
const gossipB = 2
const worstMsgLossRate = 0.4
const gossipMsgCnt = 10
const gossipSelfIntroPeriod = 5

// Heartbeat param
const hbCnt = 4
const hbNeighborCnt = 3
const immediateNeighUpdatePeriod = 15

// Rejoin param
const rejoinPeriod = 10

// Must show in one membership time upperbound
const oneMemLimit = 3

// Registered peer timeout allowance
const regAllowance = 60

// Must show in all membership (WHP) time upperbound
const allMemLimit = 2.5

// Upperlimit for read gossip message to wait
// To prevent heartbeat fail waiting for too long
const maxGossipReadBlockTime = 1

// Upper limit for read heartbeat messages to wait,
// so that the thread can still detect leave and rejoin operations
const maxHBReadBlockTime = 1

// ID and membership entry of this server
var serverID memberEntry
var serverIDStr string

// Artificial UDP fail rate
var udpFailRate = float64(0)

// Gossip data structures
var membershipList membershipListStruct
var memberIDList MemIDList
var gHeap *gossipMsgHeap

// Heartbeat target data structures
var hbTargetList hbTargetListStruct

// Print monitor flag
var pHBMonitor int32

// Heartbeat main lister socket mutex
// To deal with the fact that a UDP.Dial to the main listen socket
// will not receive any other udp messages from sources other than the main listen socket
// Therefore, reply to requests are done with the main listen socket
// and concurrency control is implemented with this mutex
var hbMainConnMutex sync.Mutex

// Heartbeat message size, initialized later
var gossipMsgSize int
var hbMsgSize int

// indicate system state
// leave means that the system has not yet joined the peer network
// rejoin means that the system was detected to be failed when it is not;
// thus it voluntarily "fails" and rejoins with a new ID
var leave = int32(1)
var rejoin = int32(0)

// Log channel, string passed will be logged to log file
var logChan = make(chan string)

// HB main thread target failed or left notifying channel
// serveFailOrLeave send entries to this channel
// to notify the HB main thread of the need to change HB targets
var noteHBMainChan = make(chan memberEntry)

// Heartbeat thread ack channel
// HB acks to requests are passed to this channel
// and sent using the main HB listen socket
var hbAckTo = make(chan replyAck)

///////////////////////////////////////
//                                   //
// gossip Msg heap ID data structure //
//                                   //
///////////////////////////////////////

// Gossip & introduction message
// Define Stat meaning:	0 -> Introduction
//						1 -> Join
//						2 -> Failed
//						3 -> SelfIntro
//						4 -> Leave
type gossipMsg struct {
	Entry memberEntry
	TTL   int32
	Stat  byte
}

// GossipMsg Heap implementation
// Based on example code in Go Document:
// https://golang.org/pkg/container/heap/
// Stores received gossipMsg that should be gossiped
// to other peers. "Sorted" (by heap) based on
// TTL of the gossip messages
type gossipMsgHeap []gossipMsg

func (h gossipMsgHeap) Len() int           { return len(h) }
func (h gossipMsgHeap) Less(i, j int) bool { return h[i].TTL > h[j].TTL } // to implement max heap, based on TTL
func (h gossipMsgHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *gossipMsgHeap) Push(x interface{}) {
	*h = append(*h, x.(gossipMsg))
}
func (h *gossipMsgHeap) Pop() interface{} {
	hlen := len(*h)
	x := (*h)[hlen-1]
	*h = (*h)[0 : hlen-1]
	return x
}

///////////////////////////////////////
//                                   //
// membership ID list data structure //
//                                   //
///////////////////////////////////////

// MemIDList is a Member ID List "class" implementation
// Store all ID in the membership list in no order
// To deal with the required random selection in gossip dissemination
type MemIDList struct {
	IDList  []string
	Length  int
	realLen int
}

// Push push an new ID onto Member ID List and return index
func (ls *MemIDList) Push(ID string) int {
	if ls.realLen == ls.Length {
		ls.IDList = append(ls.IDList, ID)
		ls.realLen++
	} else if ls.realLen > ls.Length {
		ls.IDList[ls.Length] = ID
	}
	ls.Length++
	return ls.Length - 1
}

// Pop removes an target entry from the Member ID List and update membership list
func (ls *MemIDList) Pop(indx int) error {
	if indx > ls.Length-1 || indx < 0 {
		return errors.New("Invalid index range")
	}

	tmp := membershipListEntry{Entry: membershipList.Get(ls.IDList[indx]).Entry, memIDListIndx: -1}
	membershipList.Set(ls.IDList[indx], tmp)

	tmp = membershipListEntry{Entry: membershipList.Get(ls.IDList[ls.Length-1]).Entry, memIDListIndx: indx}
	membershipList.Set(ls.IDList[ls.Length-1], tmp)

	ls.IDList[indx], ls.IDList[ls.Length-1] = ls.IDList[ls.Length-1], ls.IDList[indx]

	ls.Length--
	return nil
}

// Len returns slice length
func (ls *MemIDList) Len() int {
	return ls.Length
}

// Get return value corresponding to supplied index
func (ls *MemIDList) Get(indx int) (string, error) {
	if indx > ls.Length-1 || indx < 0 {
		return "", errors.New("Invalid index range")
	}
	return ls.IDList[indx], nil
}

////////////////////////////////////
//                                //
// membership List data structure //
//                                //
////////////////////////////////////

// membership list entry definition
type memberEntry struct {
	IP        [4]byte // IP of peer
	Timestamp int64   // Peer join timestamp
}

// Incorporating member ID list indices with membership list entries
type membershipListEntry struct {
	Entry         memberEntry
	memIDListIndx int
}

// Membershiplist structure
type membershipListStruct struct {
	list   map[string]membershipListEntry // Does not contain the entry for the server itself
	mlLock sync.Mutex
}

// Get a particular membershipListEntry from the membership list
func (ls *membershipListStruct) Get(ID string) membershipListEntry {
	ls.mlLock.Lock()
	entry := ls.list[ID]
	ls.mlLock.Unlock()

	return entry
}

// Set a particular membershipListEntry from the membership list
func (ls *membershipListStruct) Set(ID string, entry membershipListEntry) {
	ls.mlLock.Lock()
	ls.list[ID] = entry
	ls.mlLock.Unlock()
}

// Del a particular membershipListEntry from the membership list
func (ls *membershipListStruct) Del(ID string) {
	ls.mlLock.Lock()
	delete(ls.list, ID)
	ls.mlLock.Unlock()
}

// Get total length of membership list
func (ls *membershipListStruct) Len() int {
	ls.mlLock.Lock()
	length := len(ls.list)
	ls.mlLock.Unlock()

	return length
}

// Check if a particular ID is in the membership list
func (ls *membershipListStruct) In(ID string) bool {
	ls.mlLock.Lock()
	var in = false
	if _, ok := ls.list[ID]; ok {
		in = true
	}
	ls.mlLock.Unlock()

	return in
}

// Extract whole list from membership list
func (ls *membershipListStruct) GetList() []string {
	var tmp []string
	ls.mlLock.Lock()
	for ID, entry := range ls.list {
		if entry.memIDListIndx != -1 {
			tmp = append(tmp, ID)
		}
	}
	ls.mlLock.Unlock()

	return tmp
}

// Print membership list content, does not print itself
func (ls *membershipListStruct) Print() {

	tmp := ls.GetList()

	sort.Strings(tmp)
	fmt.Fprintf(os.Stderr, "Membership list on %s\n", serverIDStr)
	for _, ID := range tmp {
		fmt.Fprintln(os.Stderr, ID)
	}
}

///////////////////////////////////////////
//                                       //
// heartbeat related messages definition //
//                                       //
///////////////////////////////////////////

// heartbeat message definition
// MsgType definition	0 normal heartbeat
//						1 register with peer
//						2 deregister with peer
//						3 ack peer request
type hbMsg struct {
	Entry     memberEntry // introduce server itself
	Timestamp int64       // timestamp when the message is sent
	MsgType   byte        // what is the message for
}

// hbAckTo ack definition
type replyAck struct {
	Addr net.Addr
	Msg  []byte
}

//////////////////////////////////
//                              //
// heartbeat targets datastruct //
//                              //
//////////////////////////////////

// data structure to store heartbeat targets
type hbTargetListStruct struct {
	list   map[string]memberEntry
	hlLock sync.Mutex
}

// check if a peer is on heartbeat targets list
func (ls *hbTargetListStruct) In(ID string) bool {
	ls.hlLock.Lock()
	var in = false
	if _, ok := ls.list[ID]; ok {
		in = true
	}
	ls.hlLock.Unlock()

	return in
}

// get peer entry from heartbeat targets list
func (ls *hbTargetListStruct) Get(ID string) memberEntry {

	ls.hlLock.Lock()
	var tmp = ls.list[ID]
	ls.hlLock.Unlock()

	return tmp
}

// get all peer info from heartbeat targets list
func (ls *hbTargetListStruct) GetList() []memberEntry {
	var tmp []memberEntry

	ls.hlLock.Lock()
	for _, entry := range ls.list {
		tmp = append(tmp, entry)
	}
	ls.hlLock.Unlock()

	return tmp
}

// remove peer from heartbeat targets list
func (ls *hbTargetListStruct) Del(ID string) {

	ls.hlLock.Lock()
	delete(ls.list, ID)
	ls.hlLock.Unlock()

}

// Add peer to heartbeat targets list
func (ls *hbTargetListStruct) Set(ID string, entry memberEntry) {

	ls.hlLock.Lock()
	ls.list[ID] = entry
	ls.hlLock.Unlock()

}

// Get heartbeat targets list length
func (ls *hbTargetListStruct) Len() int {
	var length int

	ls.hlLock.Lock()
	length = len(ls.list)
	ls.hlLock.Unlock()

	return length
}

///////////////////////
//                   //
// Utility functions //
//                   //
///////////////////////

// min function for float64
func minF64(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Utility to convert IP in bytes to string
func byteIP2Str(ip [4]byte) string {
	var sb strings.Builder
	for indx, element := range ip {
		if indx != 0 {
			sb.WriteString(".")
		}
		sb.WriteString(strconv.Itoa(int(element)))
	}
	return sb.String()
}

// translate memberEntry to string to faciliate printing
func memberEntry2Str(entry memberEntry) string {
	return byteIP2Str(entry.IP) + ":" + strconv.FormatInt(entry.Timestamp, 10)
}

// Get none loopback IP of this server
// Based on answer on Stack Overflow:
// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func getIP() (net.IP, error) {

	conn, err := net.Dial("udp", "101.101.101.101:80")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting IP address of machine")
		return net.IP{}, err
	}
	defer conn.Close()

	addr := conn.LocalAddr().(*net.UDPAddr)

	return addr.IP, nil
}

// net.IP byte converter
func ip2bytes(target net.IP) [4]byte {
	var rst [4]byte
	for i := 0; i < 4; i++ {
		rst[i] = target[i]
	}
	return rst
}

// Utility for sending udp message
// No receiving done
func sendUDP(addr string, msg []byte) error {

	// mimic message failure
	if rand.Float64() < float64(udpFailRate) {
		return nil
	}

	// Start normal connection
	conn, err := net.Dial("udp", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when Dialing UDP")
		return err
	}
	defer conn.Close()

	// Send bytes
	cnt, err := conn.Write(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when sending UDP packets")
		return err
	} else if cnt != len(msg) {
		fmt.Fprintln(os.Stderr, "Error when sending UDP packets")
		return errors.New("Did not write whole message")
	}

	return nil
}

// Utility for Send request and recv ack
// receiving done, will try "attempts" times before failing
func sendnRecvUDP(addr string, out chan []byte, msg []byte, timeoutSec int, attempts int) error {

	// try for specified number of attempts
	var buf = make([]byte, 4096)
	for i := 0; i < attempts; i++ {

		// Dial introducer
		conn, err := net.Dial("udp", addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error when dialing")
			out <- []byte{}
			return err
		}
		defer conn.Close()

		// Send and listen for some time before sending a new one
		// Should fail intentionally
		if rand.Float64() > float64(udpFailRate) {
			cnt, err := conn.Write(msg)
			if err != nil || cnt != len(msg) {
				fmt.Fprintln(os.Stderr, "Error when sending UDP message")
				fmt.Println(err)
				out <- []byte{}
				return err
			}
		}

		err = conn.SetDeadline(time.Now().Add(time.Second * time.Duration(timeoutSec)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error when setting timeout")
			out <- []byte{}
			return err
		}

		// Read until timeout
		msgLen, _ := conn.Read(buf)

		if msgLen != 0 {
			out <- buf[:msgLen]
			return nil
			//break
		}
	}

	out <- []byte{}
	return nil
}

// Utility for sending udp message with heartbeat ack thread,
// will use the hb main listen socket
func sendUDPAck(addr net.Addr, msg []byte) {

	// mimic message failure
	if rand.Float64() < float64(udpFailRate) {
		return
	}

	hbAckTo <- replyAck{Addr: addr, Msg: msg}

	return
}

// Utility to encode a single gossipMsg into bytes
func encodeGossipMsg(target gossipMsg) []byte {
	var rst bytes.Buffer
	binary.Write(&rst, binary.BigEndian, target)
	return rst.Bytes()
}

// Utility to decode bytes into a single gossipMsg
func decodeGossipMsg(target []byte) gossipMsg {
	var rst gossipMsg
	var tmp = bytes.NewReader(target)
	binary.Read(tmp, binary.BigEndian, &rst)
	return rst
}

// Utiliy to turn a list of gossipMsg into bytes
func encodeWholeGossipMsg(gossipMsgList []gossipMsg) []byte {
	var rst []byte

	for _, element := range gossipMsgList {
		rst = append(rst, encodeGossipMsg(element)...)
	}

	return rst
}

// Utiliy to turn bytes into a list of gossipMsg
func decodeWholeGossipMsg(data []byte) []gossipMsg {
	var rst []gossipMsg

	gossipMsgLen := len(data)
	for i := 0; i < gossipMsgLen/gossipMsgSize; i++ {
		rst = append(rst, decodeGossipMsg(data[i*gossipMsgSize:(i+1)*gossipMsgSize]))
	}

	return rst
}

// Calculate the number of rounds that a gossip message needs to be sent
// to satisfy a certain percentage of peers receiving it based on c and b
// as discussed in lecture
func calGossipMsgRounds(introFlag bool) int {
	membershipLen := float64(memberIDList.Len() + 1)

	// introFlag used in serveIntro to make up of one member difference in intro msg
	if introFlag {
		membershipLen++
	}

	if membershipLen <= 4 {
		// Deal with case where peer number is small (<= 4)
		// Doesn't mean much to deal with worse case of three simultaneous failure
		// with small system
		// Instead, only consider message loss rate
		return int(math.Ceil(gossipC / worstMsgLossRate * math.Log2(membershipLen)))
	}

	return int(math.Ceil(
		((gossipB*gossipC-2)*math.Log2(membershipLen)/math.Log2(membershipLen-3) + 2) *
			math.Log2(membershipLen)))
}

// Calculate time period to gossip based on number of rounds
func calGossipMsgPeriod() float64 {
	var roundCnt = calGossipMsgRounds(false)

	// Though no need to gossip
	// Still gossip once
	if roundCnt == 0 {
		roundCnt = 1
	}

	// Gossip period should be the lower of
	// the calculated theoratical gossip period and maxGossipReadBlockTime
	return minF64(float64(allMemLimit)/float64(roundCnt), float64(maxGossipReadBlockTime))
}

// Retrieve the list of gossip messages that should be sent in this gossip round
func getTargetGossipMsgList() []gossipMsg {
	var rst = make([]gossipMsg, 0)
	var realGossipMsgCnt int

	// Generate self introduce message
	// Only generate such message if there are other peers
	if memberIDList.Len() > 0 {
		rst = append(rst, gossipMsg{Entry: serverID, TTL: -1, Stat: byte(3)})
	}

	// To prevent heap out of bound
	if gHeap.Len() > gossipMsgCnt {
		realGossipMsgCnt = gossipMsgCnt
	} else {
		realGossipMsgCnt = gHeap.Len()
	}

	// Retrieving gossipMsg from heap
	var tmp gossipMsg
	for i := 0; i < realGossipMsgCnt; i++ {
		tmp = heap.Pop(gHeap).(gossipMsg)
		rst = append(rst, tmp)
		tmp.TTL--
		if tmp.TTL != 0 {
			// TTL not exhausted yet, push back for further gossiping
			heap.Push(gHeap, tmp)
		}
	}

	return rst
}

// Get num of non-duplicative integers in the range of [0, end)
func genListOfRandInts(end int, num int) []int {
	var rst []int
	var set = make(map[int]bool)
	var tmp int

	if end > 3*num {
		for cnt := 0; cnt < num; {
			tmp = rand.Intn(end)
			if _, in := set[tmp]; !in {
				rst = append(rst, tmp)
				set[tmp] = true
				cnt++
			}
		}
	} else if end > num {
		// Deal with case where avalable range is too small
		// use permutation instead
		return rand.Perm(end)[:num]
	} else {
		// num is no less than number of numbers in range
		// return all number in range
		rst = make([]int, end)
		for i := 0; i < end; i++ {
			rst[i] = i
		}
	}

	return rst
}

// gossip list of messages to gossip targets
func sendWholeGossipMsg(target []byte) error {

	// Pick random gossipB targets
	// May be less than gossipB
	targetIndx := genListOfRandInts(memberIDList.Len(), gossipB)

	// Iterate through all gossip targets and send gossip messages
	var ID string
	var err error
	for _, indx := range targetIndx {
		// Get IP of target peer
		ID, _ = memberIDList.Get(indx)

		// Start udp sending
		tmp := membershipList.Get(ID)
		err = sendUDP(byteIP2Str(tmp.Entry.IP)+":"+os.Args[2], target)
		if err != nil {
			return err
		}
	}

	return nil
}

// Gossip thread Leave cleanup operations
func gossipPreLeave() error {

	// prepare to leave the system by sending out leave messages
	leaveTTL := int32(calGossipMsgRounds(false))
	for i := leaveTTL; i > 0; i-- {
		leaveMsg := gossipMsg{Entry: serverID, TTL: i, Stat: byte(4)}
		err := sendWholeGossipMsg(encodeGossipMsg(leaveMsg))
		if err != nil {
			return err
		}
	}

	// No garbage collection needed
	// the next init will take care of everything

	return nil
}

// Log function, for go running to log in back ground
func logToFile(entry string) {
	logChan <- entry
}

// Utility to encode hearbeat message to bytes
func encodeHBMsg(target hbMsg) []byte {
	var rst bytes.Buffer
	binary.Write(&rst, binary.BigEndian, target)
	return rst.Bytes()
}

// Utility to decode bytes into a single hbMsg
func decodeHBMsg(target []byte) hbMsg {
	var rst hbMsg
	var tmp = bytes.NewReader(target)
	binary.Read(tmp, binary.BigEndian, &rst)
	return rst
}

// Utility to update heartbeat targets if not full or not immediate neighbor
func updateHBTarget(doneFlag *int32) {

	// Get sorted slice of membership list IDs
	tmpList := membershipList.GetList()
	sort.Strings(tmpList)

	// Find first successors' index in virtual ring
	// the virtual ring is based on IDs in dictionary order
	firstLargerIndx := -1
	for indx, ID := range tmpList {
		if ID > serverIDStr {
			firstLargerIndx = indx
			break
		}
	}
	if firstLargerIndx == -1 {
		firstLargerIndx = 0
	}

	// Set up heartbeat target set for future lookup
	set := make(map[string]bool)
	curNeighborList := hbTargetList.GetList()
	for _, entry := range curNeighborList {
		set[memberEntry2Str(entry)] = true
	}

	// Start from firstLargerIndx, check if the three successors are registered
	// and register with them if not
	var rstChan = make(chan []byte)
	start := firstLargerIndx
	tmpListLen := len(tmpList)
	startFlag := true
	for hbTargetCnt := 0; (hbTargetCnt < hbNeighborCnt) && (start != firstLargerIndx || startFlag) && tmpListLen != 0; {
		// passed the start
		startFlag = false

		// Check if already registered
		if _, in := set[tmpList[start]]; in {
			// Already chosen
			delete(set, tmpList[start])
			hbTargetCnt++
			start = (start + 1) % tmpListLen
			continue
		}

		// register with new target
		entry := membershipList.Get(tmpList[start])
		go sendnRecvUDP(byteIP2Str(entry.Entry.IP)+":"+os.Args[1],
			rstChan, encodeHBMsg(hbMsg{Entry: serverID, Timestamp: time.Now().Unix(), MsgType: byte(1)}), 5, 3)
		rst, _ := <-rstChan
		if len(rst) == 0 {
			fmt.Fprintln(os.Stderr, "Failed when contacting "+tmpList[start])
			start = (start + 1) % tmpListLen
			continue
		}

		// Add to heartbeat target list
		hbTargetList.Set(tmpList[start], entry.Entry)
		hbTargetCnt++
		start = (start + 1) % tmpListLen
	}

	// Deregister the rest old heartbeat targets in set
	for ID := range set {

		go sendnRecvUDP(strings.Split(ID, ":")[0]+":"+os.Args[1],
			rstChan, encodeHBMsg(hbMsg{Entry: serverID, Timestamp: time.Now().Unix(), MsgType: byte(2)}), 5, 3)
		rst, _ := <-rstChan
		if len(rst) == 0 {
			fmt.Fprintln(os.Stderr, "Failed when contacting "+ID)
		}

		// Remove from heartbeat target list
		hbTargetList.Del(ID)
	}

	// Notify heartbeat main thread done
	atomic.StoreInt32(doneFlag, 1)
}

// Utility to perform clean up for heartbeating before leaving
func hbPreLeave() {
	// deregister all targets concurrently
	targetList := hbTargetList.GetList()
	var rstChan = make(chan []byte)
	for _, entry := range targetList {
		go sendnRecvUDP(byteIP2Str(entry.IP)+":"+os.Args[1],
			rstChan, encodeHBMsg(hbMsg{Entry: serverID, Timestamp: time.Now().Unix(), MsgType: byte(2)}), 5, 3)
	}

	// Wait for them to end
	for i := 0; i < len(targetList); i++ {
		<-rstChan
	}

	// No need to notify monitored since they will find out eventually
}

// Utility to notify heartbeat main thread of heartbeat target failure
func notifyHBPeerDead(entry memberEntry) {
	noteHBMainChan <- entry
}

// For resetting the error value to nil
func errNil() error {
	return nil
}

// To bootstrap new peers
// Obtains membership list content and have join message be disseminated to others
func bootstrapping(targetIntroServerAddr string) error {

	var fromIntroChan = make(chan []byte)
	go sendnRecvUDP(targetIntroServerAddr+":"+os.Args[2],
		fromIntroChan,
		encodeGossipMsg(gossipMsg{Entry: serverID, TTL: -1, Stat: byte(0)}),
		introTimeoutPeriod,
		introAttempts)

	rst, _ := <-fromIntroChan
	if len(rst) != 0 {
		return nil
	}

	return errors.New("Cannot contact intro server")

}

// Initialize server states
func initServer() error {
	// Set server ID
	machineIP, err := getIP()
	if err != nil {
		return err
	}

	serverID = memberEntry{IP: ip2bytes(machineIP), Timestamp: time.Now().Unix()}
	serverIDStr = memberEntry2Str(serverID)
	fmt.Printf("Server ID:\t%s\n", serverIDStr)

	// Clear membership list and heartbeat list
	membershipList = membershipListStruct{list: make(map[string]membershipListEntry)}
	hbTargetList = hbTargetListStruct{list: make(map[string]memberEntry)}

	// Create and init member ID list
	memberIDList = MemIDList{}

	// Create gossipMsg heap
	gHeap = &gossipMsgHeap{}
	heap.Init(gHeap)

	// Get gossipMsg binary size
	gossipMsgSize = len(encodeGossipMsg(gossipMsg{}))

	// Get hbMsg binary size
	hbMsgSize = len(encodeHBMsg(hbMsg{}))

	// Initialize UDP failure rate
	if len(os.Args) == 6 {
		udpFailRate, err = strconv.ParseFloat(os.Args[5], 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error when converting UDP_FAIL_RATE to float, using default value")
			udpFailRate = float64(0)
		}
	}

	// Init system state
	atomic.StoreInt32(&leave, 1)
	atomic.StoreInt32(&rejoin, 0)

	return nil
}

////////////////////////////////
//                            //
// Serve gossip msg functions //
//                            //
////////////////////////////////

// Serve gossip Introduction message
func serveIntro(target gossipMsg, addr net.Addr) error {
	// for introduction of new node
	// Ack the intro (one byte ack), add to membership list,
	// add to hearbeat list and store gossip message

	// Send Ack
	err := sendUDP(addr.String(), []byte{byte(1)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Into message ACK failed")
		return err
	}

	// do not reintro those already in membershiplist
	if membershipList.In(memberEntry2Str(target.Entry)) {
		return nil
	}

	// other than the ack, introduce do the same thing as serveJoinOrSelfIntro
	target.Stat = byte(1)
	target.TTL = int32(calGossipMsgRounds(true))
	serveJoinOrSelfIntro(target)

	// Init new node with membershipList entries
	newPeerAddr := byteIP2Str(target.Entry.IP) + ":" + os.Args[2]
	for i := 0; i < memberIDList.Len(); i++ {
		ID, _ := memberIDList.Get(i)
		err = sendUDP(newPeerAddr, encodeGossipMsg(gossipMsg{Entry: membershipList.Get(ID).Entry,
			TTL:  -1,
			Stat: byte(3)}))
	}

	return nil
}

// Serve gossip Join or Self Intro message
func serveJoinOrSelfIntro(target gossipMsg) error {
	// for joining of new node
	// Add to membership list, store gossip message

	// Ignore if message about itself
	if reflect.DeepEqual(target.Entry, serverID) {
		return nil
	}

	// Do things only if not on membership list
	newIDStr := memberEntry2Str(target.Entry)
	if !(membershipList.In(newIDStr)) {
		// Add to member ID list
		newPeerIndx := memberIDList.Push(newIDStr)

		// Add to membership list
		membershipList.Set(newIDStr, membershipListEntry{Entry: target.Entry, memIDListIndx: newPeerIndx})

		// Store gossip msg on heap
		// Will not do this step for rand intro msg
		if target.Stat != byte(3) {
			heap.Push(gHeap, target)
		}

		// Log to file
		go logToFile(time.Now().String() +
			"; Change made to local membership list; type JOIN; ID: " + newIDStr + "\n")
	}

	return nil
}

// Serve gossip Fail or Leave message
func serveFailOrLeave(target gossipMsg, self bool) error {

	// for failing of node
	// Update membership list, store gossip message

	// If message about itself and is fail message, means false failure detection
	// Instead of overriding the gossip, voluntary fail and rejoin immediately
	// Return error to signal upper level thread of such event
	if reflect.DeepEqual(target.Entry, serverID) && target.Stat == byte(2) {
		fmt.Fprintln(os.Stderr, "Detect failure of myself, voluntary leave and rejoin needed")
		return errors.New("Detect failure of myself, voluntary leave and rejoin needed")
	}

	// Remove from member ID list if such entries exists
	var firstPop = false
	failIDStr := memberEntry2Str(target.Entry)
	if membershipList.In(failIDStr) {
		tmp := membershipList.Get(failIDStr)
		if tmp.memIDListIndx != -1 {
			firstPop = true // to signal afterwards that this is the first fail message received
			memberIDList.Pop(tmp.memIDListIndx)
		}
	}

	// Store gossip msg on heap
	// Only store when message not seen before
	// i.e. not in membership list or is on list but not nullified yet
	//fmt.Println("IN0")
	//fmt.Println(failIDStr)
	//fmt.Println(!membershipList.In(failIDStr))
	//if membershipList.In(failIDStr) {
	//	fmt.Println(membershipList.Get(failIDStr))
	//}
	//fmt.Println("done")
	var inMLLogFlag = false
	if !membershipList.In(failIDStr) {
		//fmt.Println("IN1")
		heap.Push(gHeap, target)
	} else {
		if firstPop {
			heap.Push(gHeap, target)
			inMLLogFlag = true
		}
	}

	// Log to file about receiving fail message
	if target.Stat == byte(2) {
		if self {
			go logToFile(time.Now().String() +
				"; Detected failure; ID: " + failIDStr + "\n")
		} else {
			go logToFile(time.Now().String() +
				"; Got fail message; ID: " + failIDStr + "\n")
		}
	}

	// Log to file about change in membership list
	if inMLLogFlag {
		if target.Stat == byte(2) {
			// fail
			go logToFile(time.Now().String() +
				"; Change made to local membership list; type FAIL; ID: " + failIDStr + "\n")
		} else {
			// leave
			go logToFile(time.Now().String() +
				"; Change made to local membership list; type LEAVE; ID: " + failIDStr + "\n")
		}
	}

	// Update membership list to nullify the entry
	// If no entries exist, still add a null one to prevent delayed join message
	// The null entry is deleted after the TTL for the failed message expires
	// which should expire after the delayed join message
	membershipList.Set(failIDStr, membershipListEntry{Entry: target.Entry, memIDListIndx: -1})

	// Check if in heartbeat target list
	// If in notify heartbeat thread that it is dead
	if hbTargetList.In(failIDStr) {
		go notifyHBPeerDead(target.Entry)
	}

	return nil
}

// Serve gossip Msg based on message type
func serveMsg(target gossipMsg, addr net.Addr) (bool, error) {
	var err error
	var reInitFlag = false
	//fmt.Println("In func")

	switch target.Stat {
	case byte(0):
		//fmt.Println("b0")
		// intro message
		err = serveIntro(target, addr)
		break
	case byte(1):
		//fmt.Println("b1")
		fallthrough
	case byte(3):
		//fmt.Println("b2")
		// join message or SelfIntro message
		err = serveJoinOrSelfIntro(target)
		break
	case byte(2):
		//fmt.Println("b3")
		fallthrough
	case byte(4):
		//fmt.Println("b4")
		//fmt.Println("Got here")
		// fail message or leave message
		err = serveFailOrLeave(target, false)
		if err != nil {
			reInitFlag = true
		}
		break
	default:
		//fmt.Println("DDDDDDD")
		err = errors.New("Unkown state in gossip message")
		fmt.Fprintln(os.Stderr, err)
	}

	return reInitFlag, err
}

///////////////////////////////////
//                               //
// Serve heartbeat msg functions //
//                               //
///////////////////////////////////

// Serve hearbeat message based on message type
func serveHBMsg(msg hbMsg, monitorList []hbMsg, addr net.Addr) []hbMsg {
	// Based on different message, perform different action
	switch msg.MsgType {
	case byte(0):
		monitorList = serveHBOrgMsg(msg, monitorList)
		break
	case byte(1):
		monitorList = serveHBRegMsg(msg, monitorList, addr)
		break
	case byte(2):
		monitorList = serveHBDeregMsg(msg, monitorList, addr)
		break
	case byte(3):
	default:
		// No Ack should be received by heartbeat main thread
		// Ignore it
	}

	return monitorList
}

// Serve normal heartbeat message
func serveHBOrgMsg(msg hbMsg, monitorList []hbMsg) []hbMsg {
	// Update monitor list. Ignore those that are not on membership list
	//fmt.Println("Got heartbeat ", msg)
	for indx, target := range monitorList {
		if reflect.DeepEqual(msg.Entry, target.Entry) {
			monitorList[indx] = hbMsg{Entry: target.Entry,
				Timestamp: (time.Now().Add(time.Millisecond * time.Duration(oneMemLimit*1000))).Unix(),
				MsgType:   target.MsgType}
			break
		}
	}

	return monitorList
}

// Serve heartbeat register request
func serveHBRegMsg(msg hbMsg, monitorList []hbMsg, addr net.Addr) []hbMsg {
	// Check if already on list or not
	var checkMonitored = false
	for _, target := range monitorList {
		if reflect.DeepEqual(msg.Entry, target.Entry) {
			checkMonitored = true
			break
		}
	}

	// Not on monitor list, add it
	if !checkMonitored {
		monitorList = append(monitorList, hbMsg{Entry: msg.Entry,
			Timestamp: time.Now().Add(time.Second * time.Duration(regAllowance)).Unix(),
			MsgType:   byte(0)})
	}

	// Send Ack whether on list or not
	//fmt.Println("ACK HERE")
	go sendUDPAck(addr, encodeHBMsg(hbMsg{Entry: msg.Entry,
		Timestamp: msg.Timestamp,
		MsgType:   byte(3)}))

	return monitorList
}

// Serve heartbeat deregister request
func serveHBDeregMsg(msg hbMsg, monitorList []hbMsg, addr net.Addr) []hbMsg {
	// Locate entry in monitor list
	var delIndx = -1
	for indx, target := range monitorList {
		if reflect.DeepEqual(msg.Entry, target.Entry) {
			delIndx = indx
			break
		}
	}

	// Update monitor list if exists
	if delIndx != -1 {
		monitorList = append(monitorList[:delIndx], monitorList[delIndx+1:]...)
	}

	// Send Ack
	go sendUDPAck(addr, encodeHBMsg(hbMsg{Entry: msg.Entry,
		Timestamp: msg.Timestamp,
		MsgType:   byte(3)}))

	return monitorList
}

/////////////////////
//                 //
// main goroutines //
//                 //
/////////////////////

// Membership list dissemination and update, as well as dealing with intro msg
// Will also notify HB thread if HB targets fail or leave
func gossipThread(out chan string) error {

	// open UDP listen socket for gossip message listening
	ip := strings.Split(serverIDStr, ":")
	conn, err := net.ListenPacket("udp", ip[0]+":"+os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when listening on UDP port for gossip message")
		fmt.Fprintln(os.Stderr, err)
		out <- "RIP"
		return err
	}
	defer conn.Close()

	// Infinite loop to listen and gossip messages
	var buf = make([]byte, gossipMsgSize*gossipMsgCnt)
	var gossipPeriod int
	var baseTime = time.Now()
	var dueTime time.Time
	var selfIntroTime = time.Now()
	for true {
		// Set up timers
		gossipPeriod = int(math.Floor(calGossipMsgPeriod() * 1000000000)) // period precision to nano second
		dueTime = baseTime.Add(time.Nanosecond * time.Duration(gossipPeriod))

		// Construct gossip message from gossip message heap and encode as bytes
		// Update heap & membershipList while getting the messages
		nani := getTargetGossipMsgList()
		targetGossipMsgBytes := encodeWholeGossipMsg(nani)
		//fmt.Println("Gonna send ", nani)

		// Extract gossip msgs from UDP socket and serve them while waiting to send out gossip
		conn.SetReadDeadline(dueTime)
		for true {
			// init this two variables
			msgLen := 0
			err := errNil()

			msgLen, addr, err := conn.ReadFrom(buf)

			// Break if non timeout error happens
			if err != nil && !err.(net.Error).Timeout() {
				fmt.Fprintln(os.Stderr, err)
				break
			}

			// No message received
			if msgLen < gossipMsgSize {
				break
			}

			// Decode received bytes back to list of gossipMsg
			gotGossipMsgList := decodeWholeGossipMsg(buf[:msgLen])

			// Based on gossipMsg content, perform action
			for _, element := range gotGossipMsgList {
				rejoinFlag, _ := serveMsg(element, addr)

				// Notify main goroutine of rejoining
				if rejoinFlag {
					out <- "REJOIN"
				}
			}

			// Reached dissemination time, stop listening and serving
			if time.Now().Unix() > dueTime.Unix() {
				break
			}
		}

		// Send Gossip Message
		// Note that if the gossip message only contains self intro
		// send at most once per gossipSelfIntroPeriod
		if len(targetGossipMsgBytes) != gossipMsgSize ||
			time.Now().After(selfIntroTime.Add(time.Second*gossipSelfIntroPeriod)) {
			err = sendWholeGossipMsg(targetGossipMsgBytes)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error when sending whole gossip messages\n", err)
			}

			selfIntroTime = time.Now()
		}

		// Set base time for next round of gossiping
		baseTime = dueTime

		// Found fail message of myself
		// voluntary leave and rejoin by breaking infinite loop
		if atomic.LoadInt32(&rejoin) == 1 {
			break
		}

		// Notified to leave
		if atomic.LoadInt32(&leave) == 1 {
			gossipPreLeave()
			break
		}

	}

	// Notify main goroutine of end of execution
	out <- "RIP"

	return nil
}

// Heartbeat main thread that monitors other peers and update heartbeat target list
func heartbeatThread(out chan string) error {

	// Start heartbeat sending thread
	var hbSendFrom = make(chan string)
	go heartbeatSendThread(hbSendFrom)

	// Set up udp listen socket
	ip := strings.Split(serverIDStr, ":")
	conn, err := net.ListenPacket("udp", ip[0]+":"+os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when listening on UDP port for heartbeat message")
		fmt.Fprintln(os.Stderr, err)
		// Wait for send thread to die
		<-hbSendFrom
		out <- "RIP"
		return err
	}
	defer conn.Close()

	// Start ack replying thread
	var hbAckFrom = make(chan string)
	hbAckTo = make(chan replyAck)
	go heartbeatThreadReply(conn, hbAckTo, hbAckFrom)

	// Infinite loop for listening and processing registration and timeouts
	var buf = make([]byte, gossipMsgSize*gossipMsgCnt)
	var msg hbMsg
	var minTime int64
	var curTime int64
	var monitorList []hbMsg
	var updateNeighborTime = time.Now().Unix() + immediateNeighUpdatePeriod
	var updateHBTargetFlag = int32(1)
	for true {

		if atomic.LoadInt32(&pHBMonitor) == 1 {
			atomic.StoreInt32(&pHBMonitor, 0)
			fmt.Println(monitorList)
		}

		// Set read deadline based on heartbeat time out or max read wait time period
		minTime = time.Now().Unix() + maxHBReadBlockTime
		for _, target := range monitorList {
			if minTime > target.Timestamp {
				minTime = target.Timestamp
			}
		}

		hbMainConnMutex.Lock()
		conn.SetReadDeadline(time.Unix(minTime, 0))
		hbMainConnMutex.Unlock()

		// Read from buffer
		// Using mutex since connection is used by two threads
		hbMainConnMutex.Lock()
		err := errNil()
		msgLen, addr, err := conn.ReadFrom(buf)
		hbMainConnMutex.Unlock()

		// Check timeout
		// Do not timeout when leaving or rejoining
		if (atomic.LoadInt32(&leave) != 1) && (atomic.LoadInt32(&rejoin) != 1) {
			if !((err != nil && !err.(net.Error).Timeout()) || msgLen < hbMsgSize) {
				// Got correct message, decode and serve
				msg = decodeHBMsg(buf)

				// Serve received message
				// Ack replies are replied using original listening socket
				monitorList = serveHBMsg(msg, monitorList, addr)
			} else if err != nil && err.(net.Error).Timeout() {

				// Timeout occurred, check if any failed
				curTime = time.Now().Unix()
				for indx, target := range monitorList {
					if curTime >= target.Timestamp {
						// Found failed peer, serving
						serveFailOrLeave(gossipMsg{Entry: target.Entry,
							TTL: int32(calGossipMsgRounds(false)), Stat: byte(2)}, true)

						// Mark failed on monitor list
						monitorList[indx] = hbMsg{Entry: target.Entry, Timestamp: -1, MsgType: target.MsgType}
					}
				}

				// Update monitor list
				tmp := monitorList
				monitorList = make([]hbMsg, 0)
				for _, target := range tmp {
					if target.Timestamp != -1 {
						monitorList = append(monitorList, target)
					}
				}
			}
		}

		// Check if failed or leave note made to thread
		select {
		case entry, _ := <-noteHBMainChan:
			// Notified of heartbeat target failing
			// Remove target from list
			//fmt.Println("Check and Removing entry ", entry)
			detectedID := memberEntry2Str(entry)
			if hbTargetList.In(detectedID) {
				hbTargetList.Del(detectedID)
				//fmt.Println("Deleted ID")
				//fmt.Println(hbTargetList.GetList())
				//fmt.Println(monitorList)
			}
			break
		default:
			// Poll channel rather than block
		}

		// Do not update target list when leaving or rejoining
		if (atomic.LoadInt32(&leave) != 1) && (atomic.LoadInt32(&rejoin) != 1) {
			// Fill up neighbors if heartbeat targets smaller than hbNeighborCnt
			if hbTargetList.Len() < hbNeighborCnt || time.Now().Unix() > updateNeighborTime {
				//Check if previous round done
				if atomic.LoadInt32(&updateHBTargetFlag) == 1 {
					atomic.StoreInt32(&updateHBTargetFlag, 0)
					go updateHBTarget(&updateHBTargetFlag)
				}
				updateNeighborTime = time.Now().Unix() + immediateNeighUpdatePeriod
			}
		}

		// Check if leave or rejoin flag is up
		if atomic.LoadInt32(&rejoin) == 1 {
			break
		}

		// Notified to leave
		// Before leave, deregister every registered target
		if atomic.LoadInt32(&leave) == 1 {
			hbPreLeave()
			break
		}

	}

	// Wait for send thread to die
	<-hbSendFrom

	// Wait for ack thread to die
	hbAckTo <- replyAck{Msg: []byte{}}
	<-hbAckFrom

	// Notify main thread end of execution
	out <- "RIP"
	return nil
}

// Heartbeat sending thread to send out heartbeat periodically
func heartbeatSendThread(out chan string) error {

	var nxtSendTime time.Time
	var hbPeriod = time.Duration(int(math.Floor(float64(oneMemLimit) / float64(hbCnt) * 1000000000))) // nano second scale

	// send heartbeat periodically before detection timeout
	for true {
		nxtSendTime = time.Now().Add(time.Nanosecond * hbPeriod)

		// Construct message and send heartbeat via go routine
		targetList := hbTargetList.GetList()
		for _, entry := range targetList {
			msg := encodeHBMsg(hbMsg{Entry: serverID, Timestamp: time.Now().Unix(), MsgType: byte(0)})
			go sendUDP(byteIP2Str(entry.IP)+":"+os.Args[1], msg)
		}

		// Found that should die
		if atomic.LoadInt32(&leave) == 1 || atomic.LoadInt32(&rejoin) == 1 {
			break
		}

		//sleep until next send time
		time.Sleep(nxtSendTime.Sub(time.Now()))
	}

	// Notify main thread of end of execution
	out <- "RIP"

	return nil
}

// Thread to ack message of peer HB request
// Uses main UDP listen socket of HB main thread
func heartbeatThreadReply(conn net.PacketConn, in chan replyAck, out chan string) {

	var dieFlag = false

	for !dieFlag {
		select {
		case reply, _ := <-in:
			if len(reply.Msg) == 0 {
				dieFlag = true
			} else {
				hbMainConnMutex.Lock()
				conn.WriteTo(reply.Msg, reply.Addr)
				hbMainConnMutex.Unlock()

				//fmt.Println("ACK: ", reply)
			}
		}
	}

	out <- "RIP"
}

// UI thread to receive commands from user
func uiThread(out chan string) {
	leaveFlag := false
	reader := bufio.NewReader(os.Stdin)

	// infinite loop until user decides to leave
	for !leaveFlag {
		fmt.Fprintln(os.Stderr, "Type in\t1 to show membership list")
		fmt.Fprintln(os.Stderr, "\t2 to show self ID")
		fmt.Fprintln(os.Stderr, "\t3 to join")
		fmt.Fprintln(os.Stderr, "\t4 to voluntarily leave")
		fmt.Fprintf(os.Stderr, "> ")
		text, _ := reader.ReadString('\n')
		switch text {
		case "1\n":
			// check membership list
			if atomic.LoadInt32(&leave) == 0 {
				membershipList.Print()
			} else {
				fmt.Fprintln(os.Stderr, "In leave state, membership table empty")
			}
			break
		case "2\n":
			// get server ID
			if atomic.LoadInt32(&leave) == 0 {
				fmt.Fprintf(os.Stderr, "ID:\t%s\n", serverIDStr)
			} else {
				fmt.Fprintln(os.Stderr, "In leave state, server ID not yet assigned")
			}
			break
		case "3\n":
			// Join
			if atomic.LoadInt32(&leave) == 1 {
				fmt.Fprintln(os.Stderr, "Start joining")
				out <- "join"
			} else {
				fmt.Fprintln(os.Stderr, "Already in join state")
			}
			break
		case "4\n":
			// Leave
			if atomic.LoadInt32(&leave) == 0 {
				fmt.Fprintln(os.Stderr, "Time to leave")
				out <- "leave"
			} else {
				fmt.Fprintln(os.Stderr, "Already in leave state")
			}
			break
		case "5\n":
			// Show HB List and Monitor List
			if atomic.LoadInt32(&leave) == 0 {
				fmt.Println("HB: ", hbTargetList.GetList())
				atomic.StoreInt32(&pHBMonitor, 1)
			} else {
				fmt.Fprintln(os.Stderr, "In leave state")
			}
			break
		case "exit\n":
			leaveFlag = true
			out <- "exit"
			break
		default:
			fmt.Fprintln(os.Stderr, "Unknown option")
			break

		}
		fmt.Fprintln(os.Stderr, "")
	}
	fmt.Fprintln(os.Stderr, "See you")
}

// Log thread to do the logging stuff
func logThread(in chan string) {
	// listen on channel forever
	for true {
		select {
		case entry, _ := <-in:
			// append those log entries received from channel to log file
			fp, err := os.OpenFile(os.Args[4], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error when opening log file")
			}

			_, err = fp.WriteString(entry)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error when writing to log file")
			}

			fp.Close()
		}
	}
}

// Main thread of execution, deal with starting of all threads and notifying them to die
// Receive commands from UI thread
func main() {

	// Check commandline arguments
	// Note that the log querying part should be run separately in another program for performance
	if len(os.Args) != 5 && len(os.Args) != 6 {
		fmt.Println("Usage: go run server.go HB_PORT DISSEM_PORT INTRO STORE_LOG_PATH [UDP_FAIL_RATE]")
		return
	}

	// Start UI thread
	var uiChanFrom = make(chan string)
	go uiThread(uiChanFrom)

	logChan = make(chan string)
	go logThread(logChan)

	// Start goroutine control
	var leaveFlag = false
	var gossipChanFrom = make(chan string)
	var heartbeatChanFrom = make(chan string)
	for !leaveFlag {
		select {
		case cmd, _ := <-uiChanFrom:

			switch cmd {
			case "join":

				initServer()

				// change to in join state
				atomic.StoreInt32(&leave, 0)

				// run gossip thread before bootstrapping to receive whole membership list
				go gossipThread(gossipChanFrom)

				// run heartbeat thread
				go heartbeatThread(heartbeatChanFrom)

				// Introducer does not need to bootstrap
				bootstrapFlag, _ := strconv.Atoi(os.Args[3])
				if bootstrapFlag != 1 {
					// for normal peers use vm01 to join
					bootstrapping(primIntroServerAddr)
				} else {
					// introducer use secondary intro server address to join
					bootstrapping(secIntroServerAddr)
				}

				break
			case "exit":
				// exit is also leaving with that the main thread dies
				leaveFlag = true
				fallthrough
			case "leave":

				// change to leave state
				atomic.StoreInt32(&leave, 1)

				// wait for them to die
				for cnt := 0; cnt < 2; {
					select {
					case x, _ := <-gossipChanFrom:
						if x == "RIP" {
							cnt++
						}
						break
					case x, _ := <-heartbeatChanFrom:
						if x == "RIP" {
							cnt++
						}
					}
				}

				go logToFile(time.Now().String() +
					"; Leaving the system, flushing membership list\n")

				break
			}
		case x, _ := <-gossipChanFrom:
			if x == "REJOIN" {
				// get suicide request
				// wait for them to die and then restart
				atomic.StoreInt32(&rejoin, 1)
				for cnt := 0; cnt < 2; {
					select {
					case x, _ := <-gossipChanFrom:
						if x == "RIP" {
							cnt++
						}
						break
					case x, _ := <-heartbeatChanFrom:
						if x == "RIP" {
							cnt++
						}
					}
				}

				go logToFile(time.Now().String() +
					"; Leaving the system to rejoin, flushing membership list\n")

				// Random wait to rejoin
				time.Sleep(time.Second * time.Duration(rand.Intn(rejoinPeriod)))

				// restart
				initServer()

				// change to in join state, rejoin is already 0
				atomic.StoreInt32(&leave, 0)

				go logToFile(time.Now().String() +
					"; Rejoining the system\n")

				// run gossip thread before bootstrapping to receive whole membership list
				go gossipThread(gossipChanFrom)

				// run heartbeat thread
				go heartbeatThread(heartbeatChanFrom)

				// Introducer does not need to bootstrap
				bootstrapFlag, _ := strconv.Atoi(os.Args[3])
				if bootstrapFlag != 1 {
					// for normal peers use vm01 to join
					bootstrapping(primIntroServerAddr)
				} else {
					// introducer use secondary intro server address to join
					bootstrapping(secIntroServerAddr)
				}

			} else if x == "RIP" {
				// gossipThread dead due to error, kill heartbeatThread and remain in leave state
				atomic.StoreInt32(&leave, 1)
				<-heartbeatChanFrom
			}
			break
		case x, _ := <-heartbeatChanFrom:
			if x == "RIP" {
				// gossipThread dead due to error, kill heartbeatThread and remain in leave state
				atomic.StoreInt32(&leave, 1)
				<-gossipChanFrom
			}
		}
	}
}
