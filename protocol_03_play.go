package main

import (
	"fmt"
	"time"
)

func createPlayProtocol() func(packetId uint8) PacketHandler {
	handlers := make(map[uint8]PacketHandler)
	handlers[0x00] = handleKeepAlive
	handlers[0x01] = handleTextInput
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

func handleKeepAlive(client *Client, reader ByteArrayReader) error {
	randomId, err := readVarInt(reader)
	if err != nil {
		return err
	}
	if randomId == client.lastKeepAlive {
		client.lastReceivedKeepAliveTime = time.Now().UnixMilli()
	}
	return nil
}

func handleTextInput(client *Client, reader ByteArrayReader) error {
	textInput, err := readString(reader)
	if err != nil {
		return err
	}
	return client.sendMessage(ChatComponent{Text: fmt.Sprintf("<%s> %s", client.identity.username, textInput)}, 0)
}
