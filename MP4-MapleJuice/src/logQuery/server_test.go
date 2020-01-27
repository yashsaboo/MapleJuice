package logQuery

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"testing"
	"time"

	"logQuery/rpc_export"
)

var testClient *rpc.Client

// This sets up a local RPC server to test with
func TestMain(m *testing.M) {
	node := new(rpc_export.Node)

	err := rpc.Register(node)
	if err != nil {
		log.Fatal("Error registering node", err)
	}

	rpc.HandleHTTP()

	fmt.Println("Successfully created RPC node")

	listener, err := net.Listen("tcp", ":4321")

	if err != nil {
		log.Fatal("Listener error", err)
	}
	log.Printf("serving rpc on port %d", 4321)
	go func() {
		http.Serve(listener, nil)

		fmt.Println("Successfully launched HTTP listener")
	}()

	time.Sleep(time.Second * 2) // Wait for everything to launch

	testClient, err = rpc.DialHTTP("tcp", "localhost:4321")

	fmt.Println("Successfully connected to test client")

	if err != nil {
		log.Fatal("error serving: ", err)
	}

	m.Run()
	listener.Close()
}
