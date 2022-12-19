package main

import (
	"log"
	"strconv"
	"strings"
)

func createHandshakeProtocol() func(packetId uint8) PacketHandler {
	handlers := make(map[uint8]PacketHandler)
	handlers[0x00] = handleHandshake
	return func(packetId uint8) PacketHandler {
		return handlers[packetId]
	}
}

func handleHandshake(client *Client, reader ByteArrayReader) error {
	log.Println("answering handshake")
	var err error
	client.protocolVersion, err = readVarInt(reader)
	if err != nil {
		return err
	}
	client.virtualHost, err = readString(reader)
	if err != nil {
		return err
	}
	index := strings.Index(client.virtualHost, "\000")
	if index != -1 {
		client.virtualHost = client.virtualHost[:index]
	}
	port, err := readShort(reader)
	if err != nil {
		return err
	}
	client.virtualHost += ":" + strconv.Itoa(int(port))
	requestedProtocolState, err := readVarInt(reader)
	if err != nil {
		return err
	}
	return client.switchProtocol(requestedProtocolState)
}
