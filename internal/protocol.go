package internal

import (
	"_tomcraft/go-mc-ping/internal/codec"
	"reflect"
)

var DefinedProtocols []ProtocolHandler

type PacketHandler func(client *Client, reader codec.ByteArrayReader) error
type ProtocolHandler func(packetId byte) PacketHandler

func AutoPacketHandler[T any](mappedFunction func(client *Client, packet *T) error) PacketHandler {
	t := reflect.TypeOf(mappedFunction).In(1).Elem()
	return func(client *Client, reader codec.ByteArrayReader) error {
		ptr, err := codec.DeserializePacket[T](reader, t)
		if err != nil {
			return err
		}
		return mappedFunction(client, ptr)
	}
}

func MapProtocolHandler(handlers map[byte]PacketHandler) ProtocolHandler {
	return func(packetId byte) PacketHandler {
		return handlers[packetId]
	}
}

func MapProtocolHandlerWithDefault(handlers map[byte]PacketHandler, defaultHandler PacketHandler) ProtocolHandler {
	return func(packetId byte) PacketHandler {
		handler, ok := handlers[packetId]
		if ok {
			return handler
		}
		return defaultHandler
	}
}

func init() {
	DefinedProtocols = []ProtocolHandler{
		CreateHandshakeProtocol(),
		CreateStatusProtocol(),
		CreateLoginProtocol(),
		CreatePlayProtocol(),
	}
}
