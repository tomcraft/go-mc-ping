package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
)

type StatusVersion struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

type StatusPlayers struct {
	Max    int `json:"max"`
	Online int `json:"online"`
}

type ChatComponent struct {
	Text string `json:"text"`
}

type StatusResponse struct {
	Version     StatusVersion `json:"version"`
	Players     StatusPlayers `json:"players"`
	Description ChatComponent `json:"description"`
	Favicon     string        `json:"favicon"`
}

var favicon = readFavicon("assets/eduard.png")

func readFavicon(file string) string {
	image, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)
}

func createStatusProtocol() func(packetId uint8) PacketHandler {
	handlers := make(map[uint8]PacketHandler)
	handlers[0x00] = handleStatusRequest
	handlers[0x01] = handlePingRequest
	return func(packetId uint8) PacketHandler {
		return handlers[packetId]
	}
}

func createStatusResponse(protocolVersion int, favicon string) StatusResponse {
	return StatusResponse{
		Version: StatusVersion{
			"GoProtocol",
			protocolVersion,
		},
		Players:     StatusPlayers{Max: -1, Online: 6969},
		Description: ChatComponent{Text: "§cCoucou §bles §anoobs"},
		Favicon:     favicon,
	}
}

func handleStatusRequest(client *Client, reader ByteArrayReader) error {
	log.Println("answering status request")
	return client.sendPacket(0x00, func(writer ByteArrayWriter) error {
		response := createStatusResponse(client.protocolVersion, favicon)
		if jsonBytes, err := json.Marshal(response); err != nil {
			return err
		} else {
			return writeByteArray(writer, jsonBytes)
		}
	})
}

func handlePingRequest(client *Client, reader ByteArrayReader) error {
	log.Println("answering ping request")

	pingTime, err := readLong(reader)
	if err != nil {
		return err
	}

	if err = client.sendPacket(0x01, func(writer ByteArrayWriter) error {
		return writeLong(writer, pingTime)
	}); err != nil {
		return err
	}

	return client.closeConnection()
}
