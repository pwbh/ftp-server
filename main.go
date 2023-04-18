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

	// Data stream
	port := rand.Intn(MAX_PORT-MIN_PORT) + MIN_PORT
	address := fmt.Sprintf("localhost:%d", port)
	fmt.Printf("Data stream address: %s\n", address)
	dataStream, err := net.Listen("tcp", address)

	if err != nil {
		fmt.Println("Data stream error: ", err)
	}

	defer dataStream.Close()

	for {
		n, err := conn.Read(tmp)

		if err != nil {
			if err != io.EOF {
				fmt.Println("read error: ", err)
			}
			break
		}

		call, err := parseCall(string(tmp[:n]))

		if err != nil {
			fmt.Println("call parse error: ", err)
			break
		}

		println(call.c)

		handleCall(call, conn, dataStream, port)
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

func handleCall(c *call, conn net.Conn, dataStream net.Listener, port int) {
	switch c.c {
	case "USER":
		data := fmt.Sprintf("331 User '%s' OK. Password required\n", c.args[0])
		conn.Write([]byte(data))

	case "LIST":
		conn.Write([]byte("150 Opening ASCII mode data connection for file list.\n"))
		handleDataStream(c, dataStream, "LIST")
		conn.Write([]byte("226 Transfer complete.\n"))

	case "TYPE":
		conn.Write([]byte("200 Type set to I.\n"))

	case "PASS":
		conn.Write([]byte("230 OK. Current restricted directory is /\n"))

	case "STOR":
		conn.Write([]byte("125 Transfer starting\n"))
		handleDataStream(c, dataStream, "STOR")
		conn.Write([]byte("226 Transfer complete.\n"))

	case "PASV":
		p, k := calculatePort(port)
		data := fmt.Sprintf("227 Entering Passive Mode (127,0,0,1,%d,%d)\n", p, k)
		conn.Write([]byte(data))

	case "PUT":
		conn.Write([]byte("150 Accepted data connection\n"))

	case "QUIT":
		conn.Close()

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

func handleDataStream(c *call, dataStream net.Listener, streamType string) error {
	conn, err := dataStream.Accept()

	if err != nil {
		fmt.Printf("[%s] when trying to accept a TCP connection from the client.", streamType)
		return err
	}

	defer conn.Close()

	switch streamType {
	case "LIST":
		handleList(conn)

	case "STOR":
		handleFileTransfer(conn, c.args[0])

	// need to add more cases

	default:
		conn.Write([]byte("500 Unrecognized.\n"))
		return fmt.Errorf("unrecognized comman was sent to the server")
	}

	return nil
}

func handleList(conn net.Conn) error {
	dir, err := os.Getwd()

	if err != nil {
		conn.Write([]byte("550 Requested action not taken.\n"))
		return err

	}

	files, err := os.ReadDir(dir)

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

		data := fmt.Sprintf("%v\t%s\t %d\t %v\n", info.Mode(), file.Name(), info.Size(), file.IsDir())

		buf = append(buf, data...)
	}

	conn.Write(buf)

	return nil
}

func handleFileTransfer(conn net.Conn, path string) error {
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

	if err := os.WriteFile(path, buf[:total], WRITE_MODE); err != nil {
		return err
	}

	return nil
}
