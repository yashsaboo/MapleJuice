package rpc_export

import (
	"log"
)

var kill_signal *chan bool

var GrepID int

var UL *log.Logger

type Node int

func SetKill(k *chan bool) {
	kill_signal = k
}

func SetLogger(l *log.Logger) {
	UL = l
}

func SetNodeID(i int) {
	GrepID = i
}
