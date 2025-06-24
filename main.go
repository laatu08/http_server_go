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

	if method == "GET" {
		if url == "/" {
			// return 200 ok response
			writeResponse(conn,200,"Welcome Home...")
		} else if strings.HasPrefix(url,"/echo/"){
			echoStr:=strings.TrimPrefix(url,"/echo/")
			writeResponse(conn,200,echoStr)
		} else {
			// Return 404 Not Found response
			writeResponse(conn,404,"Not found")
		}

	}
}


func writeResponse(conn net.Conn,statusCode int,body string){
	statusText:=map[int]string{
		200:"OK",
		404:"Not found",
	}[statusCode]

	// build response string
	response:=fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText) +
		"Content-Type: text/plain\r\n" +
		fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
		"Connection: close\r\n" +
		"\r\n" +
		body


	// write to client
	_,err:=conn.Write([]byte(response))
	if err!=nil{
		fmt.Println("Error writing response:",err)
	}
}
