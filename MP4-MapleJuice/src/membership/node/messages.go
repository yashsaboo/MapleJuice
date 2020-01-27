package node

// Contains data structurs for each message format. They are all based on a common Message format.
import (
	"net"
)

// Message is the base message struct
type Message struct {
	NodeID     uint64
	T          uint64
	RemoteAddr *net.UDPAddr
	Hostname   string
	Orig       string
	UDPPort    int
}

// JoinMessage format
type JoinMessage struct {
	*Message
	TCPPort int
}

// IntroMessage format
type IntroMessage struct {
	*Message
}

// MemberMessage format
type MemberMessage struct {
	*Message
	TCPPort int
}

// HeartMessage format
type HeartMessage struct {
	*Message
}

// LeaveMessage format
type LeaveMessage struct {
	*Message
}

// FailMessage format
type FailMessage struct {
	*Message
}

// FileMessage format
type FileMessage struct {
	*Message
	LocalName string
	SDFSName  string
	Updated   int64
	Hash      string
}
