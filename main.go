package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
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
	reader := bufio.NewReader(conn)

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	// split the request into parts
	parts := strings.Split(strings.TrimSpace(requestLine), " ")
	if len(parts) < 3 {
		fmt.Println("Invalid request line:", parts)
		return
	}

	method := parts[0]
	url := parts[1]

	var response string

	if method == "GET" && url == "/" {
		// return 200 ok response
		body := "Welcome....."
		response = "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain\r\n" +
			fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
			"Connection: close\r\n" +
			"\r\n" +
			body
	} else {
		// Return 404 Not Found response
		body := "Not Found"
		response = "HTTP/1.1 404 Not Found\r\n" +
			"Content-Type: text/plain\r\n" +
			fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
			"Connection: close\r\n" +
			"\r\n" +
			body
	}

	conn.Write([]byte(response))
}
