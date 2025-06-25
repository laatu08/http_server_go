package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var fileDirectory string

func main() {
	// parse --directory flag
	flag.StringVar(&fileDirectory, "directory", ".", "Directory to serve file from")
	flag.Parse()
	fmt.Println("Serving from directory:", fileDirectory)

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
		go handleConnection(conn)
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

	// Read header
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading headers:", err.Error())
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break // End of headers
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex != -1 {
			key := strings.TrimSpace(line[:colonIndex])
			value := strings.TrimSpace(line[colonIndex+1:])
			headers[strings.ToLower(key)] = value
		}
	}

	if method == "GET" {
		if url == "/" {
			// return 200 ok response
			header := "Content-Type: text/plain\r\n"
			writeResponse(conn, 200, header, "Welcome Home...")
		} else if strings.HasPrefix(url, "/echo/") {
			header := "Content-Type: text/plain\r\n"
			echoStr := strings.TrimPrefix(url, "/echo/")

			acceptEncoding := headers["accept-encoding"]
			hasGzip := false

			for _, encoding := range strings.Split(acceptEncoding, ",") {
				if strings.TrimSpace(encoding) == "gzip" {
					hasGzip = true
					break
				}
			}

			if hasGzip {
				header += "Content-Encoding: gzip\r\n"
			}

			writeResponse(conn, 200, header, echoStr)
		} else if url == "/user-agent" {
			header := "Content-Type: text/plain\r\n"
			userAgent := headers["user-agent"]
			writeResponse(conn, 200, header, userAgent)
		} else if strings.HasPrefix(url, "/files/") {
			filename := strings.TrimPrefix(url, "/files/")
			serveFile(conn, filename)
		} else {
			// Return 404 Not Found response
			header := "Content-Type: text/plain\r\n"
			writeResponse(conn, 404, header, "Not found")
		}
	} else if method == "POST" {
		if strings.HasPrefix(url, "/files/") {
			filename := strings.TrimPrefix(url, "/files/")
			contentLength := headers["content-length"]

			length, err := strconv.Atoi(contentLength)
			if err != nil {
				writeResponse(conn, 400, "", "Invalid Content-Length")
				return
			}

			body := make([]byte, length)
			_, err = reader.Read(body)
			if err != nil {
				writeResponse(conn, 400, "", "Fail to read body")
				return
			}

			err = writeFile(filename, body)
			if err != nil {
				writeResponse(conn, 500, "", "Failed to write file")
				return
			}

			writeResponse(conn, 201, "", "")
		}
	}
}

func writeResponse(conn net.Conn, statusCode int, header string, body string) {
	statusText := map[int]string{
		200: "OK",
		404: "Not found",
	}[statusCode]

	// build response string
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText) +
		header +
		fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
		"Connection: close\r\n" +
		"\r\n" +
		body

	// write to client
	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
}

func serveFile(conn net.Conn, filename string) {
	// construct full path
	fullPath := filepath.Join(fileDirectory, filename)

	// Read file contents
	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		writeResponse(conn, 404, "", "")
		return
	}

	header := "Content-Type: application/octet-stream\r\n"

	writeResponse(conn, 200, header, string(data))
}

func writeFile(filename string, data []byte) error {
	fullpath := filepath.Join(fileDirectory, filename)
	return os.WriteFile(fullpath, data, 0644)
}
