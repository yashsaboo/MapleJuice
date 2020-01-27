package main


import (
	"net"
	"fmt"
	"bufio"
	"os"
)


func sendPacket(ip string, message string) {
	conn, _ := net.Dial("tcp", "127.0.0.1:8090")
	
	fmt.Fprintf(conn, message + "\n")
}



func main () {
	conn, _ := net.Dial("tcp", "127.0.0.1:8090") //make a connection to the local server

	reader := bufio.NewReader(os.Stdin) //reader from console
	fmt.Print("Type to send (client, server): ")
	text, _ := reader.ReadString('\n') //actually read the input


	fmt.Println(text)

	var outMessage string

	if text == "client\n" {
		outMessage = "{\"PacketType\":\"client_request\",\"Command\":\"put\",\"FileName\":\"test/test.txt\"}"
	} else {
		outMessage = "{\"PacketType\":\"server_request\",\"Command\":\"write\",\"FileName\":\"test/test.txt\"}"
	}
    
    fmt.Fprintf(conn, outMessage + "\n") // send to socket
    message, _ := bufio.NewReader(conn).ReadString('\n')
    fmt.Print("Message from server: " + message)
}

