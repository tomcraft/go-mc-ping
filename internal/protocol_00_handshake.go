package internal

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

func CreateHandshakeProtocol() func(packetId byte) PacketHandler {
	handlers := make(map[byte]PacketHandler)
	handlers[0x00] = AutoHandler(handleHandshake)
	return func(packetId byte) PacketHandler {
		return handlers[packetId]
	}
}

func handleHandshake(client *Client, packet *HandshakePacket) error {
	log.Println("Answering handshake")

	index := strings.Index(packet.ServerAddress, "\000")
	if index != -1 {
		packet.ServerAddress = packet.ServerAddress[:index]
	}

	client.ProtocolVersion = packet.ProtocolVersion
	client.VirtualHost = fmt.Sprintf("%s:%d", packet.ServerAddress, packet.ServerPort)
	return client.SwitchProtocol(packet.RequestedState)
}
