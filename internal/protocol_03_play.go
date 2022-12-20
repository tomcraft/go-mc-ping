package internal

import (
	"fmt"
	"time"
)

type KeepAlivePacket struct {
	RandomId int `packet:"varint"`
}

type JoinGamePacket struct {
	EntityId         int    `packet:"int"`
	GameMode         byte   `packet:"byte"`
	Dimension        int8   `packet:"int8"`
	Difficulty       byte   `packet:"byte"`
	MaxPlayers       byte   `packet:"byte"`
	LevelType        string `packet:"string"`
	ReducedDebugInfo bool   `packet:"bool"`
}

type PlayerPosLookPacket struct {
	X, Y, Z    float64 `packet:"float64"`
	Yaw, Pitch float32 `packet:"float"`
	Flags      byte    `packet:"byte"`
}

type IncomingChatPacket struct {
	Text string `packet:"string"`
}

type OutgoingChatPacket struct {
	Text     ChatComponent `packet:"component"`
	Position byte          `packet:"byte"`
}

type CustomPayloadPacket struct {
	Tag     string `packet:"string"`
	Payload []byte `packet:"byte_array"`
}

func CreatePlayProtocol() ProtocolHandler {
	handlers := make(map[byte]PacketHandler)
	handlers[0x00] = AutoPacketHandler(handleKeepAlive)
	handlers[0x01] = AutoPacketHandler(handleTextInput)
	return MapProtocolHandlerWithDefault(handlers, func(client *Client, reader ByteArrayReader) error {
		return nil
	})
}

func handleKeepAlive(client *Client, packet *KeepAlivePacket) error {
	if packet.RandomId == client.LastKeepAlive {
		client.LastReceivedKeepAliveTime = time.Now().UnixMilli()
	}
	return nil
}

func handleTextInput(client *Client, packet *IncomingChatPacket) error {
	return client.SendMessage(ChatComponent{Text: fmt.Sprintf("<%s> %s", client.Identity.Username, packet.Text)}, 0)
}
