package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type PacketHandler func(client *Client, reader ByteArrayReader) error

type Client struct {
	connection                net.Conn
	connectionReader          *bufio.Reader
	connectionActive          bool
	readBuffer                bytes.Buffer
	sendBuffer                bytes.Buffer
	sendLock                  sync.Mutex
	protocolVersion           int
	protocolState             int
	protocolHandler           func(packetId uint8) PacketHandler
	virtualHost               string
	identity                  *Identity
	lastKeepAlive             int
	lastSentKeepAliveTime     int64
	lastReceivedKeepAliveTime int64
}

type Identity struct {
	uuid     string
	username string
}

func (client *Client) closeConnection() error {
	if client.connectionActive {
		client.connectionActive = false
		if err := client.connection.Close(); err != nil {
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
		for client.connectionActive {
			if err := client.processPacket(); err != nil {
				log.Println("error processing packets, closing connection:", err)
				if err := client.closeConnection(); err != nil {
					log.Println("error while closing connection:", err)
					break
				}
			}
		}
	}()
	for client.connectionActive {
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
	if newProtocol < 0 || newProtocol > 3 {
		return errors.New("invalid new protocol")
	} else if newProtocol <= client.protocolState && client.protocolState > 0 {
		return errors.New("invalid new protocol: unable to enter a previous state")
	}
	log.Printf("switching to protocol %d\n", newProtocol)

	client.protocolState = newProtocol
	switch newProtocol {
	case 0:
		client.protocolHandler = createHandshakeProtocol()
	case 1:
		client.protocolHandler = createStatusProtocol()
	case 2:
		client.protocolHandler = createLoginProtocol()
	case 3:
		client.protocolHandler = createPlayProtocol()
	}
	return nil
}

func (client *Client) processPacket() error {
	frameBytes, err := readByteArray(client.connectionReader)
	if err != nil {
		return err
	}

	client.readBuffer.Reset()

	if _, err = client.readBuffer.Write(frameBytes); err != nil {
		return err
	}

	packetId, err := readVarInt(&client.readBuffer)
	if err != nil {
		return err
	}

	handler := client.protocolHandler(uint8(packetId))

	if handler == nil {
		if err := client.disconnect(ChatComponent{Text: fmt.Sprintf("Unknown packet id %d", uint8(packetId))}); err != nil {
			return err
		} else {
			return nil
		}
	}

	if err := handler(client, &client.readBuffer); err != nil {
		return err
	} else {
		return nil
	}
}

func (client *Client) tick() error {
	if client.protocolState == 3 {
		now := time.Now().UnixMilli()
		if now-client.lastSentKeepAliveTime >= 1000 {
			if err := client.keepAlive(); err != nil {
				return err
			}
		} else if now-client.lastReceivedKeepAliveTime >= 20000 {
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

func (client *Client) sendPacket(id int8, handler func(writer ByteArrayWriter) error) error {
	packetBuffer := bytes.Buffer{}

	if err := writeVarInt(&packetBuffer, int(id)); err != nil {
		return err
	}
	if err := handler(&packetBuffer); err != nil {
		return err
	}

	client.sendLock.Lock()
	defer client.sendLock.Unlock()
	client.sendBuffer.Reset()
	if err := writeByteArray(&client.sendBuffer, packetBuffer.Bytes()); err != nil {
		return err
	}
	_, err := client.sendBuffer.WriteTo(client.connection)
	return err
}

func (client *Client) keepAlive() error {
	client.lastSentKeepAliveTime = time.Now().UnixMilli()
	client.lastKeepAlive = int(time.Now().Unix())
	return client.sendPacket(0x00, func(writer ByteArrayWriter) error {
		return writeVarInt(writer, client.lastKeepAlive)
	})
}

func (client *Client) join() error {
	log.Println("sending join sequence")
	client.lastReceivedKeepAliveTime = time.Now().UnixMilli() - 5000
	client.lastSentKeepAliveTime = client.lastReceivedKeepAliveTime
	if err := client.sendPacket(0x01, func(writer ByteArrayWriter) error {
		if err := writeInt(writer, 1); err != nil {
			return err
		}
		if err := writeUnsignedByte(writer, 3); err != nil {
			return err
		}
		if err := writeByte(writer, 0); err != nil {
			return err
		}
		if err := writeUnsignedByte(writer, 0); err != nil {
			return err
		}
		if err := writeUnsignedByte(writer, 255); err != nil {
			return err
		}
		if err := writeString(writer, "flat"); err != nil {
			return err
		}
		return writeBool(writer, false)
	}); err != nil {
		return err
	}
	if err := client.sendPacket(0x3F, func(writer ByteArrayWriter) error {
		if err := writeString(writer, "MC|Brand"); err != nil {
			return err
		}
		return writeString(writer, "GoTOTO")
	}); err != nil {
		return err
	}
	if err := client.sendPacket(0x08, func(writer ByteArrayWriter) error {
		pos := []float64{0.0, 100.0, 0}
		for _, val := range pos {
			if err := writeDouble(writer, val); err != nil {
				return err
			}
		}
		direction := []float32{0.0, 0.0}
		for _, val := range direction {
			if err := writeFloat(writer, val); err != nil {
				return err
			}
		}
		return writeByte(writer, 0)
	}); err != nil {
		return err
	}
	return nil
}

func (client *Client) sendMessage(message ChatComponent, position byte) error {
	return client.sendPacket(0x02, func(writer ByteArrayWriter) error {
		jsonBytes, err := json.Marshal(message)
		if err != nil {
			return err
		}
		if err := writeByteArray(writer, jsonBytes); err != nil {
			return err
		}
		return writeUnsignedByte(writer, position)
	})
}

func (client *Client) disconnect(message ChatComponent) error {
	log.Println("disconnecting client for reason: " + message.Text)
	if client.protocolState == 0 || client.protocolState == 1 {
		return client.closeConnection()
	}
	packetId := int8(0x00)
	if client.protocolState == 3 {
		packetId = int8(0x40)
	}
	if err := client.sendPacket(packetId, func(writer ByteArrayWriter) error {
		jsonBytes, err := json.Marshal(message)
		if err != nil {
			return err
		}
		return writeByteArray(writer, jsonBytes)
	}); err != nil {
		return err
	}
	return client.closeConnection()
}
