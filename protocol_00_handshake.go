package main

import (
	"fmt"
	"log"
	"strings"
)

type HandshakePacket struct {
	ProtocolVersion int    `packet:"varint"`
	ServerAddress   string `packet:"string"`
	ServerPort      int    `packet:"int16"`
	RequestedState  int    `packet:"varint"`
}

func createHandshakeProtocol() func(packetId uint8) PacketHandler {
	handlers := make(map[uint8]PacketHandler)
	handlers[0x00] = wrapHandler(handleHandshake)
	return func(packetId uint8) PacketHandler {
		return handlers[packetId]
	}
}

func handleHandshake(client *Client, packet HandshakePacket) error {
	log.Println("answering handshake")

	index := strings.Index(packet.ServerAddress, "\000")
	if index != -1 {
		packet.ServerAddress = packet.ServerAddress[:index]
	}

	client.protocolVersion = packet.ProtocolVersion
	client.virtualHost = fmt.Sprintf("%s:%d", packet.ServerAddress, packet.ServerPort)
	return client.switchProtocol(packet.RequestedState)
}
