package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	fmt.Println("Server started...")

	// start http server
	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Fail to bind at port 4221")
		os.Exit(1)
	}

	for {
		// Accept client connection
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// handle connection (3-way handshaking)
		handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read request
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading request in handle connection:", err)
		return
	}

	// Form a response
	response := "HTTP/1.1 200 OK\r\n" + "Content-Type: text/plain\r\n" + "Content-Length: 13\r\n" + "Connection: close\r\n" + "\r\n" + "Hello World!"

	// send response
	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}
