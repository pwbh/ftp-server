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

		handleCall(call, conn)
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

func handleCall(c *call, conn net.Conn) {
	switch c.c {
	case "USER":
		data := fmt.Sprintf("331 User '%s' OK. Password required\n", c.args[0])
		conn.Write([]byte(data))

	case "LIST":
		files, err := os.ReadDir("/")

		if err != nil {
			conn.Write([]byte("550 Requested action not taken.\n"))
			break
		}

		buf := make([]byte, 0, 4096)

		for _, file := range files {
			info, err := file.Info()

			if err != nil {
				continue
			}

			data := fmt.Sprintf("%s\t %d\t %v\n", file.Name(), info.Size(), file.IsDir())
			buf = append(buf, data...)
		}

		conn.Write(buf)

	case "TYPE":
		conn.Write([]byte("200 Mode is accepted\n"))

	case "PASS":
		conn.Write([]byte("230 OK. Current restricted directory is /\n"))

	case "STOR":
		conn.Write([]byte("125 transfer starting\n"))

	case "PASV":
		port := rand.Intn(MAX_PORT-MIN_PORT) + MIN_PORT
		fmt.Println(port)
		p, k := calculatePort(port)

		err := initiateStream("file transfer", port, handleFileTransfer)

		if err != nil {
			conn.Write([]byte("425 Can't open data connection\n"))
			break
		}

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

func initiateStream(streamName string, port int, handler func(conn net.Conn) error) error {
	address := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		fmt.Printf("[%s] can't start a tcp stream\n", streamName)
		return err
	}

	go func() {
		defer func() {
			fmt.Printf("[%s] closing stream\n", streamName)
			listener.Close()
		}()

		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("[%s] error while accepting a TCP connection: %s\n", streamName, err)
		}

		if err := handler(conn); err != nil {
			fmt.Printf("[%s] error while streaming: %s\n", streamName, err)
		}
	}()

	return nil
}

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
