package internal

import (
	"reflect"
)

type ChatComponent struct {
	Text string `json:"text"`
}

type PacketHandler func(client *Client, reader ByteArrayReader) error

func AutoHandler[T any](mappedFunction func(client *Client, packet T) error) PacketHandler {
	t := reflect.TypeOf(mappedFunction).In(1)
	return func(client *Client, reader ByteArrayReader) error {
		pp, err := DeserializePacket(reader, t)
		if err != nil {
			return err
		}
		return mappedFunction(client, *(pp.Interface().(*T)))
	}
}
