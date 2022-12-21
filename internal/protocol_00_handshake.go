package internal

import (
	"fmt"
	"log"
	"strings"
)

type HandshakePacket struct {
	ProtocolVersion uint   `packet:"uvarint"`
	ServerAddress   string `packet:"string"`
	ServerPort      int    `packet:"int16"`
	RequestedState  uint   `packet:"uvarint"`
}

func CreateHandshakeProtocol() ProtocolHandler {
	handlers := make(map[byte]PacketHandler)
	handlers[0x00] = AutoPacketHandler(handleHandshake)
	return MapProtocolHandler(handlers)
}

func handleHandshake(client *Client, packet *HandshakePacket) error {
	log.Println("Answering handshake")

	index := strings.Index(packet.ServerAddress, "\000")
	if index != -1 {
		packet.ServerAddress = packet.ServerAddress[:index]
	}

	client.ProtocolVersion = packet.ProtocolVersion
	client.VirtualHost = fmt.Sprintf("%s:%d", packet.ServerAddress, packet.ServerPort)
	return client.SwitchProtocol(int(packet.RequestedState))
}
