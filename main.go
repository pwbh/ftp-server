package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
)

const WRITE_MODE = fs.FileMode(0666)
const MIN_PORT = 8001
const MAX_PORT = 65535

const LIST = 3

type command string

type call struct {
	c    command
	args []string
}

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")

	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("error while accepting a TCP connection %s", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	tmp := make([]byte, 4096)

	conn.Write([]byte("220 FTP Server ready.\n"))

	var currentStream net.Listener

	for {
		n, err := conn.Read(tmp)

		if err != nil {
			if err != io.EOF {
				fmt.Println("read error: ", err)
			}
			break
		}

		fmt.Print(string(tmp))

		call, err := parseCall(string(tmp[:n]))

		if err != nil {
			fmt.Println("call parse error: ", err)
			break
		}

		handleCall(call, conn, &currentStream)
	}

}

func parseCall(cmd string) (*call, error) {
	items := strings.Split(cmd, " ")

	args := make([]string, len(items[1:]))

	for i, item := range items[1:] {
		args[i] = strings.TrimSpace(item)
	}

	c := command(strings.TrimSpace(items[0]))

	return &call{c: c, args: args}, nil

}

func handleCall(c *call, conn net.Conn, currentStream *net.Listener) {
	switch c.c {
	case "USER":
		data := fmt.Sprintf("331 User '%s' OK. Password required\n", c.args[0])
		conn.Write([]byte(data))

	case "LIST":
		handleStreamReal(currentStream, "LIST")

	case "TYPE":
		conn.Write([]byte("200 Mode is accepted\n"))

	case "PASS":
		conn.Write([]byte("230 OK. Current restricted directory is /\n"))

	case "STOR":
		conn.Write([]byte("125 transfer starting\n"))

	case "PASV":
		port := rand.Intn(MAX_PORT-MIN_PORT) + MIN_PORT

		address := fmt.Sprintf("localhost:%d", port)

		listener, err := net.Listen("tcp", address)

		if err != nil {
			conn.Write([]byte("425 Can't open data connection\n"))
			break
		}

		*currentStream = listener

		p, k := calculatePort(port)

		data := fmt.Sprintf("227 Entering Passive Mode (127,0,0,1,%d,%d)\n", p, k)

		conn.Write([]byte(data))

	case "PUT":
		conn.Write([]byte("150 Accepted data connection\n"))

	default:
		data := fmt.Sprintf("502 Command not implemented (%s)\n", c.c)
		conn.Write([]byte(data))
	}
}

func calculatePort(desiredPort int) (byte, byte) {
	k := desiredPort % 256
	p := (desiredPort - k) / 256
	return byte(p), byte(k)
}

func handleStreamReal(currentStream *net.Listener, streamType string) {
	defer func() {
		fmt.Printf("[%s] closing stream\n", streamType)
		(*currentStream).Close()
	}()

	conn, err := (*currentStream).Accept()

	if err != nil {
		fmt.Printf("[%s] error while accepting a TCP connection: %s\n", streamType, err)
	}

	if err := handleStream(streamType, conn); err != nil {
		fmt.Printf("[%s] error while streaming: %s\n", streamType, err)
	}
}

func handleStream(streamType string, conn net.Conn) error {
	switch streamType {
	case "LIST":
		return handleList(conn)

		// need to add more cases

	default:
		conn.Write([]byte("500 Unrecognized."))
		return nil
	}
}

// func initiateStream(user net.Conn, streamType *int, port int) error {
// 	address := fmt.Sprintf("localhost:%d", port)
//
// 	fmt.Printf("STREAM TYPE %d", streamType)
//
// 	listener, err := net.Listen("tcp", address)
//
// 	if err != nil {
// 		fmt.Printf("[%d] can't start a tcp stream\n", *streamType)
// 		return err
// 	}
//
// 	defer func() {
// 		fmt.Printf("[%d] closing stream\n", *streamType)
// 		listener.Close()
// 	}()
//
// 	p, k := calculatePort(port)
//
// 	data := fmt.Sprintf("227 Entering Passive Mode (127,0,0,1,%d,%d)\n", p, k)
//
// 	user.Write([]byte(data))
//
// 	conn, err := listener.Accept()
//
// 	if err != nil {
// 		fmt.Printf("[%d] error while accepting a TCP connection: %s\n", *streamType, err)
// 	}
//
// 	if err := handleStream(streamType, conn); err != nil {
// 		fmt.Printf("[%d] error while streaming: %s\n", *streamType, err)
// 	}
//
// 	return nil
// }

func handleFileTransfer(conn net.Conn) error {
	defer conn.Close()

	buf := make([]byte, 0, 4096) // file container
	packet := make([]byte, 8192)

	var total int

	for {
		n, err := conn.Read(packet)

		if err != nil && err != io.EOF {
			fmt.Println("read error: ", err)
			return err
		}

		buf = append(buf, packet[:n]...)

		total += n

		if n != len(packet) {
			fmt.Println("Done receiving bytes.")
			break
		}
	}

	fmt.Printf("Total bytes received: %d. Writing file.\n", total)

	if err := os.WriteFile("newfile.png", buf[:total], WRITE_MODE); err != nil {
		return err
	}

	return nil
}

func handleList(conn net.Conn) error {
	files, err := os.ReadDir("/")

	if err != nil {
		conn.Write([]byte("550 Requested action not taken.\n"))
		return err
	}

	buf := make([]byte, 0, 4096)

	for _, file := range files {
		info, err := file.Info()

		if err != nil {
			fmt.Printf("Error reading file %s\n", err)
			continue
		}

		data := fmt.Sprintf("%s\t %d\t %v\n", file.Name(), info.Size(), file.IsDir())
		buf = append(buf, data...)
	}

	fmt.Println(string(buf))

	conn.Write(buf)

	return nil
}
