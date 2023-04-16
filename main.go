package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"strings"
)

const WRITE_MODE = fs.FileMode(0666)

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

	case "TYPE":
		if c.args[0] != "bin" {

		} else {
			conn.Write([]byte("200 Mode is accepted\n"))
		}

	case "PASS":
		conn.Write([]byte("230 OK. Current restricted directory is /\n"))

	case "STOR":
		conn.Write([]byte("125 transfer starting\n"))

	case "PASV":
		port := 22000
		p, k := calculatePort(port)

		err := initiateFileTransfer(port)

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

func initiateFileTransfer(port int) error {
	address := fmt.Sprintf("localhost:%d", port)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		fmt.Println("[file transfer] can't start a tcp stream for file transfer")
		return err
	}

	go func() {
		defer func() {
			fmt.Println("[file transfer] closing file transfer stream")
			listener.Close()
		}()

		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("[file transfer] error while accepting a TCP connection %s\n", err)
		}

		if err := handleFileTransfer(conn); err != nil {
			fmt.Printf("[file transfer] error while transferring file %s\n", err)
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
