package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type PacketHandler func(client *Client, reader ByteArrayReader) error

type Client struct {
	Connection                net.Conn
	ConnectionReader          *bufio.Reader
	ConnectionActive          bool
	ReadBuffer                bytes.Buffer
	SendBuffer                bytes.Buffer
	SendLock                  sync.Mutex
	ProtocolVersion           int
	ProtocolState             int
	ProtocolHandler           func(packetId byte) PacketHandler
	VirtualHost               string
	Identity                  *Identity
	LastKeepAlive             int
	LastSentKeepAliveTime     int64
	LastReceivedKeepAliveTime int64
}

type Identity struct {
	Uuid     string
	Username string
}

func (client *Client) closeConnection() error {
	if client.ConnectionActive {
		client.ConnectionActive = false
		if err := client.Connection.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) processClient() {
	err := client.switchProtocol(0)
	if err != nil {
		log.Println("error switching protocol:", err)
		if err := client.closeConnection(); err != nil {
			log.Println("error while closing connection:", err)
		}
		return
	}
	go func() {
		for client.ConnectionActive {
			if err := client.processPacket(); err != nil {
				log.Println("error processing packets, closing connection:", err)
				if err := client.closeConnection(); err != nil {
					log.Println("error while closing connection:", err)
					break
				}
			}
		}
	}()
	for client.ConnectionActive {
		if err := client.tick(); err != nil {
			log.Println("error ticking client, closing connection:", err)
			if err := client.closeConnection(); err != nil {
				log.Println("error while closing connection:", err)
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (client *Client) switchProtocol(newProtocol int) error {
	if newProtocol <= client.ProtocolState && client.ProtocolState > 0 {
		return errors.New("invalid new protocol: unable to enter a previous state")
	}
	log.Printf("switching to protocol %d\n", newProtocol)

	client.ProtocolState = newProtocol
	switch newProtocol {
	case 0:
		client.ProtocolHandler = createHandshakeProtocol()
	case 1:
		client.ProtocolHandler = createStatusProtocol()
	case 2:
		client.ProtocolHandler = createLoginProtocol()
	case 3:
		client.ProtocolHandler = createPlayProtocol()
	default:
		return errors.New(fmt.Sprintf("invalid new protocol: %d", newProtocol))
	}
	return nil
}

func (client *Client) processPacket() error {
	frameBytes, err := readByteArray(client.ConnectionReader)
	if err != nil {
		return err
	}

	client.ReadBuffer.Reset()

	if _, err = client.ReadBuffer.Write(frameBytes); err != nil {
		return err
	}

	packetId, err := readVarInt(&client.ReadBuffer)
	if err != nil {
		return err
	}

	handler := client.ProtocolHandler(byte(packetId))

	if handler == nil {
		return client.disconnect(ChatComponent{Text: fmt.Sprintf("Unknown packet id %d", byte(packetId))})
	}

	return handler(client, &client.ReadBuffer)
}

func (client *Client) tick() error {
	if client.ProtocolState == 3 {
		now := time.Now().UnixMilli()
		if now-client.LastSentKeepAliveTime >= 1000 {
			if err := client.keepAlive(); err != nil {
				return err
			}
		} else if now-client.LastReceivedKeepAliveTime >= 20000 {
			return client.disconnect(ChatComponent{Text: "Timed-out :("})
		}
		if (now/1000)%2 == 0 {
			if err := client.sendMessage(ChatComponent{"Coucou bb"}, 2); err != nil {
				return err
			}
		}
	}
	return nil
}

func (client *Client) sendPacket(id byte, packet any) error {
	return client.sendRawPacket(id, func(writer ByteArrayWriter) error {
		return serializePacket(writer, packet)
	})
}

func (client *Client) sendRawPacket(id byte, handler func(writer ByteArrayWriter) error) error {
	packetBuffer := bytes.Buffer{}

	if err := writeVarInt(&packetBuffer, int(id)); err != nil {
		return err
	}
	if err := handler(&packetBuffer); err != nil {
		return err
	}

	client.SendLock.Lock()
	defer client.SendLock.Unlock()
	client.SendBuffer.Reset()
	if err := writeByteArray(&client.SendBuffer, packetBuffer.Bytes()); err != nil {
		return err
	}
	_, err := client.SendBuffer.WriteTo(client.Connection)
	return err
}

func (client *Client) keepAlive() error {
	client.LastSentKeepAliveTime = time.Now().UnixMilli()
	client.LastKeepAlive = int(time.Now().Unix())
	return client.sendPacket(0x00, KeepAlivePacket{client.LastKeepAlive})
}

func (client *Client) join() error {
	log.Println("sending join sequence")
	client.LastReceivedKeepAliveTime = time.Now().UnixMilli() - 5000
	client.LastSentKeepAliveTime = client.LastReceivedKeepAliveTime
	if err := client.sendPacket(0x01, JoinGamePacket{
		EntityId:         1,
		GameMode:         3,
		Dimension:        0,
		Difficulty:       0,
		MaxPlayers:       16,
		LevelType:        "flat",
		ReducedDebugInfo: false,
	}); err != nil {
		return err
	}
	if err := client.sendPacket(0x3F, CustomPayloadPacket{"MC|Brand", []byte("GoTOTO")}); err != nil {
		return err
	}
	if err := client.sendPacket(0x08, PlayerPosLookPacket{Y: 100}); err != nil {
		return err
	}
	return nil
}

func (client *Client) sendMessage(message ChatComponent, position byte) error {
	return client.sendPacket(0x02, OutgoingChatPacket{message, position})
}

func (client *Client) disconnect(message ChatComponent) error {
	log.Println("disconnecting client for reason: " + message.Text)
	if client.ProtocolState == 0 || client.ProtocolState == 1 {
		return client.closeConnection()
	}
	packetId := byte(0x00)
	if client.ProtocolState == 3 {
		packetId = byte(0x40)
	}
	if err := client.sendPacket(packetId, DisconnectPacket{message}); err != nil {
		return err
	}
	return client.closeConnection()
}
