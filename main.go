package main

import (
	"bufio"
	"log"
	"net"
)

const (
	ServerHost = "0.0.0.0"
	ServerPort = "25565"
	ServerType = "tcp"
)

func main() {
	server, err := net.Listen(ServerType, ServerHost+":"+ServerPort)
	if err != nil {
		log.Fatal("Error listening: ", err)
	}
	defer func(server net.Listener) {
		if err := server.Close(); err != nil {
			log.Fatal("Error closing server: ", err)
		}
	}(server)
	log.Println("Listening on " + ServerHost + ":" + ServerPort)
	log.Println("Waiting for client...")
	for {
		connection, err := server.Accept()
		if err != nil {
			log.Println("Error accepting: ", err)
			continue
		}
		log.Println("client connected")
		client := Client{
			connection:       connection,
			connectionReader: bufio.NewReader(connection),
			connectionActive: true,
		}
		go client.processClient()
	}
}
