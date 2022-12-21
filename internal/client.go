package internal

import (
	"_tomcraft/go-mc-ping/internal/codec"
	"_tomcraft/go-mc-ping/internal/types"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Client struct {
	Connection                net.Conn
	ConnectionReader          *bufio.Reader
	ConnectionActive          bool
	ReadBuffer                bytes.Buffer
	SendBuffer                bytes.Buffer
	SendLock                  sync.Mutex
	ProtocolVersion           uint
	ProtocolState             int
	ProtocolHandler           ProtocolHandler
	VirtualHost               string
	Identity                  *types.Identity
	LastKeepAlive             uint
	LastSentKeepAliveTime     int64
	LastReceivedKeepAliveTime int64
}

func (client *Client) CloseConnection() error {
	if client.ConnectionActive {
		client.ConnectionActive = false
		if err := client.Connection.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) ProcessClient() {
	client.ConnectionReader = bufio.NewReader(client.Connection)
	client.ConnectionActive = true
	client.ProtocolState = -1
	if err := client.SwitchProtocol(0); err != nil {
		log.Println("error switching protocol, closing connection:", err)
		if err := client.CloseConnection(); err != nil {
			log.Println("error while closing connection:", err)
			return
		}
	}
	go func() {
		for client.ConnectionActive {
			if err := client.processPacket(); err != nil {
				log.Println("error processing packets, closing connection:", err)
				if err := client.CloseConnection(); err != nil {
					log.Println("error while closing connection:", err)
					break
				}
			}
		}
	}()
	for client.ConnectionActive {
		if err := client.tick(); err != nil {
			log.Println("error ticking client, closing connection:", err)
			if err := client.CloseConnection(); err != nil {
				log.Println("error while closing connection:", err)
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (client *Client) SwitchProtocol(newProtocol int) error {
	switch newProtocol {
	case 0, 1, 2, 3:
		if newProtocol <= client.ProtocolState {
			return errors.New("invalid new protocol: unable to enter a previous state")
		}
		log.Printf("switching to protocol %d\n", newProtocol)
		client.ProtocolState = newProtocol
		client.ProtocolHandler = DefinedProtocols[newProtocol]
	default:
		return fmt.Errorf("invalid new protocol: %d", newProtocol)
	}
	return nil
}

func (client *Client) processPacket() error {
	frameBytes, err := codec.ReadByteArray(client.ConnectionReader)
	if err != nil {
		return err
	}

	client.ReadBuffer.Reset()

	if _, err = client.ReadBuffer.Write(frameBytes); err != nil {
		return err
	}

	packetId, err := codec.ReadUnsignedVarInt(&client.ReadBuffer)
	if err != nil {
		return err
	}

	handler := client.ProtocolHandler(byte(packetId))

	if handler == nil {
		return client.Disconnect(types.ChatComponent{Text: fmt.Sprintf("Unknown packet id %d", byte(packetId))})
	}

	return handler(client, &client.ReadBuffer)
}

func (client *Client) tick() error {
	if client.ProtocolState == 3 {
		now := time.Now().UnixMilli()
		if now-client.LastSentKeepAliveTime >= 1000 {
			if err := client.KeepAlive(); err != nil {
				return err
			}
		} else if now-client.LastReceivedKeepAliveTime >= 20000 {
			return client.Disconnect(types.ChatComponent{Text: "Timed-out :("})
		}
		if (now/1000)%2 == 0 {
			if err := client.SendMessage(types.ChatComponent{Text: "Coucou bb"}, 2); err != nil {
				return err
			}
		}
	}
	return nil
}

func (client *Client) SendPacket(id byte, packet any) error {
	return client.SendRawPacket(id, func(writer codec.ByteArrayWriter) error {
		return codec.SerializePacket(writer, packet)
	})
}

func (client *Client) SendRawPacket(id byte, handler func(writer codec.ByteArrayWriter) error) error {
	packetBuffer := bytes.Buffer{}

	if err := codec.WriteUnsignedVarInt(&packetBuffer, uint(id)); err != nil {
		return err
	}
	if err := handler(&packetBuffer); err != nil {
		return err
	}

	client.SendLock.Lock()
	defer client.SendLock.Unlock()
	client.SendBuffer.Reset()
	if err := codec.WriteByteArray(&client.SendBuffer, packetBuffer.Bytes()); err != nil {
		return err
	}
	_, err := client.SendBuffer.WriteTo(client.Connection)
	return err
}

func (client *Client) KeepAlive() error {
	client.LastSentKeepAliveTime = time.Now().UnixMilli()
	client.LastKeepAlive = uint(time.Now().Unix())
	return client.SendPacket(0x00, KeepAlivePacket{client.LastKeepAlive})
}

func (client *Client) Join() error {
	log.Println("sending join sequence")
	client.LastReceivedKeepAliveTime = time.Now().UnixMilli() - 5000
	client.LastSentKeepAliveTime = client.LastReceivedKeepAliveTime
	if err := client.SendPacket(0x01, JoinGamePacket{
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
	if err := client.SendPacket(0x3F, CustomPayloadPacket{"MC|Brand", []byte("GoTOTO")}); err != nil {
		return err
	}
	if err := client.SendPacket(0x08, PlayerPosLookPacket{Y: 100}); err != nil {
		return err
	}
	return nil
}

func (client *Client) SendMessage(message types.ChatComponent, position byte) error {
	return client.SendPacket(0x02, OutgoingChatPacket{message, position})
}

func (client *Client) Disconnect(message types.ChatComponent) error {
	log.Println("disconnecting client for reason: " + message.Text)
	if client.ProtocolState == 0 || client.ProtocolState == 1 {
		return client.CloseConnection()
	}
	packetId := byte(0x00)
	if client.ProtocolState == 3 {
		packetId = byte(0x40)
	}
	if err := client.SendPacket(packetId, DisconnectPacket{message}); err != nil {
		return err
	}
	return client.CloseConnection()
}
