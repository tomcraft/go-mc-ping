package main

import (
	"_tomcraft/go-mc-ping/internal"
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
		log.Println("Client connected")
		client := internal.Client{
			Connection: connection,
		}
		go client.ProcessClient()
	}
}
