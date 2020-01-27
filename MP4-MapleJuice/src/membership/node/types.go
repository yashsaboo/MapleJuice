package node

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// RPCNode is used for rpc calls
type RPCNode int

// OtherNode contains info on other nodes in the network
type OtherNode struct {
	NodeID        uint64
	UDPAddr       *net.UDPAddr
	Hostname      string
	TCPPort       int
	UDPPort       int
	LastHeartbeat uint64 //Epoch is easier to deal with
}

// ThisNode contains info on our node
type ThisNode struct {
	*OtherNode
	ctx  context.Context
	conn io.Reader
	port int

	Logger         *log.Logger
	Neighbors      *NeighborList
	Members        *MembershipList
	MessageCache   *RecentMessageCache // Add the message cache too
	Files          *GlobalFileList
	GetsInProgress *OngoingNeeds
	Heartbeats     []int
	Active         bool
}

// RecentMessageCache with list and lock
type RecentMessageCache struct {
	L              *sync.Mutex
	RecentMessages []RecentMessage
}

// FileEntry holds data for a file we know about
type FileEntry struct {
	LocalName string
	SDFSName  string
	Hostname  string
	TimeAdded int64
	Hash      string
}

// GlobalFileList holds data for a collection of files
type GlobalFileList struct {
	L     *sync.Mutex
	Files []FileEntry
}

// NeedFileResponses holds the responses from nodes when we have requested a file
// We wait until we have received a majority until doing the transfer from the most recent one
type NeedFileResponses struct {
	L         *sync.Mutex
	Responses []FileEntry
}

// RecentMessage format
type RecentMessage struct {
	Type         string
	OriginatorID uint64
	Timestamp    uint64 // Set when message is added to cache
}

// PastHaveMessage collected here
type PastHaveMessage struct {
	Filename  string
	Timestamp int64
	Hostname  string
}

// OngoingNeedLabel labels each ongoing need with a timeout
type OngoingNeedLabel struct {
	ID        uint64
	Timeout   time.Time
	LocalPath string
}

// OngoingNeeds holds ongoing Needs
type OngoingNeeds struct {
	L      *sync.Mutex
	Labels []OngoingNeedLabel
	Needs  [][]PastHaveMessage
}

// WriteVerification holds data for confirming write-write conflicts
type WriteVerification struct {
	Flag      bool
	L         *sync.Mutex
	Confirmed bool
}
