package main

import (
	"io"
	"log"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening to TCP port 9000...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr()
	log.Printf("connected: %s", addr)

	n, err := io.Copy(conn, conn)
	if err != nil {
		log.Printf("Transfer error from %s: %v", addr, err)
	}
	log.Printf("Disconnected: %s after %d bytes", addr, n)
}
