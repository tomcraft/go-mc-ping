package main

import (
	"fmt"
	"time"
)

type KeepAlivePacket struct {
	RandomId int `packet:"varint"`
}

type JoinGamePacket struct {
	EntityId         int    `packet:"int"`
	GameMode         byte   `packet:"unsigned_byte"`
	Dimension        int8   `packet:"byte"`
	Difficulty       byte   `packet:"unsigned_byte"`
	MaxPlayers       byte   `packet:"unsigned_byte"`
	LevelType        string `packet:"string"`
	ReducedDebugInfo bool   `packet:"bool"`
}

type PlayerPosLookPacket struct {
	X, Y, Z    float64 `packet:"float64"`
	Yaw, Pitch float32 `packet:"float"`
	Flags      byte    `packet:"unsigned_byte"`
}

type IncomingChatPacket struct {
	Text string `packet:"string"`
}

type OutgoingChatPacket struct {
	Text     ChatComponent `packet:"component"`
	Position byte          `packet:"unsigned_byte"`
}

type CustomPayloadPacket struct {
	Tag     string `packet:"string"`
	Payload []byte `packet:"byte_array"`
}

func createPlayProtocol() func(packetId uint8) PacketHandler {
	handlers := make(map[uint8]PacketHandler)
	handlers[0x00] = wrapHandler(handleKeepAlive)
	handlers[0x01] = wrapHandler(handleTextInput)
	defaultHandler := func(client *Client, reader ByteArrayReader) error {
		return nil
	}
	return func(packetId uint8) PacketHandler {
		handler, ok := handlers[packetId]
		if ok {
			return handler
		}
		return defaultHandler
	}
}

func handleKeepAlive(client *Client, packet KeepAlivePacket) error {
	if packet.RandomId == client.lastKeepAlive {
		client.lastReceivedKeepAliveTime = time.Now().UnixMilli()
	}
	return nil
}

func handleTextInput(client *Client, packet IncomingChatPacket) error {
	return client.sendMessage(ChatComponent{Text: fmt.Sprintf("<%s> %s", client.identity.username, packet.Text)}, 0)
}
