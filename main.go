package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
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

	shouldClose := strings.ToLower(headers["connection"]) == "close"

	if method == "GET" {
		if url == "/" {
			// return 200 ok response
			header := "Content-Type: text/plain\r\n"
			var close string
			if shouldClose {
				close += "Connection: close\r\n"
			}
			writeResponse(conn, 200, header, "Welcome Home...", close)
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

			var body []byte
			var err error
			var message string

			if hasGzip {
				body, message = gzipCompress(echoStr)
				if err != nil {
					fmt.Println("Compression error:", err)
					close := "Connection: close\r\n"
					writeResponse(conn, 500, "", "", close)
					return
				}
				header += "Content-Encoding: gzip\r\n"
			} else {
				writeResponse(conn, 200, header, echoStr, "")
			}
			var close string
			if shouldClose {
				close += "Connection: close\r\n"
			}
			fmt.Println(body)
			// writeRawResponse(conn, 200, header, body)
			writeResponse(conn, 200, header, message, close)
		} else if url == "/user-agent" {
			header := "Content-Type: text/plain\r\n"
			userAgent := headers["user-agent"]
			var close string
			if shouldClose {
				close += "Connection: close\r\n"
			}
			writeResponse(conn, 200, header, userAgent, close)
		} else if strings.HasPrefix(url, "/files/") {
			filename := strings.TrimPrefix(url, "/files/")
			serveFile(conn, filename)
		} else {
			// Return 404 Not Found response
			header := "Content-Type: text/plain\r\n"

			close := "Connection: close\r\n"

			writeResponse(conn, 404, header, "Not found", close)
		}
	} else if method == "POST" {
		if strings.HasPrefix(url, "/files/") {
			filename := strings.TrimPrefix(url, "/files/")
			contentLength := headers["content-length"]

			length, err := strconv.Atoi(contentLength)
			if err != nil {
				close := "Connection: close\r\n"
				writeResponse(conn, 400, "", "Invalid Content-Length", close)
				return
			}

			body := make([]byte, length)
			_, err = reader.Read(body)
			if err != nil {
				close := "Connection: close\r\n"
				writeResponse(conn, 400, "", "Fail to read body", close)
				return
			}

			err = writeFile(filename, body)
			if err != nil {
				close := "Connection: close\r\n"
				writeResponse(conn, 500, "", "Failed to write file", close)
				return
			}

			var close string
			if shouldClose {
				close += "Connection: close\r\n"
			}
			writeResponse(conn, 201, "", "", close)
		}
	}
}

func writeResponse(conn net.Conn, statusCode int, header string, body string, close string) {
	statusText := map[int]string{
		200: "OK",
		404: "Not found",
		201: "Created",
		500: "Internal Server Error",
	}[statusCode]

	// build response string
	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText) +
		header +
		fmt.Sprintf("Content-Length: %d\r\n", len(body)) + close +
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
		close := "Connection: close\r\n"
		writeResponse(conn, 404, "", "", close)
		return
	}

	header := "Content-Type: application/octet-stream\r\n"

	writeResponse(conn, 200, header, string(data), "")
}

func writeFile(filename string, data []byte) error {
	fullpath := filepath.Join(fileDirectory, filename)
	return os.WriteFile(fullpath, data, 0644)
}

func writeRawResponse(conn net.Conn, statusCode int, header string, body []byte) {
	statusText := map[int]string{
		200: "OK",
		404: "Not found",
		201: "Created",
		500: "Internal Server Error",
	}[statusCode]

	// build response string
	fmt.Printf("Sending raw response with length: %d\n", len(body))

	response := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText) +
		header +
		fmt.Sprintf("Content-Length: %d\r\n", len(body)) +
		"Connection: close\r\n"

	// write headers
	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Header write error:", err)
		return
	}

	// write raw body
	// _, err = io.Copy(conn, bytes.NewReader(body))
	// if err != nil {
	// 	fmt.Println("Body write error:", err)
	// }

	written := 0
	for written < len(body) {
		n, err := conn.Write(body[written:])
		if err != nil {
			fmt.Println("Body write error:", err)
			return
		}
		written += n
	}

}

func gzipCompress(input string) ([]byte, string) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, _ = gzipWriter.Write([]byte(input))
	// if err != nil {
	// 	return nil, err
	// }

	gzipWriter.Close()
	// if err != nil {
	// 	return nil, err
	// }

	gzipMessage := buf.String()

	return buf.Bytes(), gzipMessage
}
