package internal

import (
	"_tomcraft/go-mc-ping/internal/types"
	"encoding/base64"
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

type StatusResponse struct {
	Version     StatusVersion       `json:"version"`
	Players     StatusPlayers       `json:"players"`
	Description types.ChatComponent `json:"description"`
	Favicon     string              `json:"favicon"`
}

type EmptyPacket struct{}

type StatusResponsePacket struct {
	Response StatusResponse `packet:"json"`
}

type PingPacket struct {
	PingTime int64 `packet:"int64"`
}

var favicon = readFavicon("assets/eduard.png")

func readFavicon(file string) string {
	image, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)
}

func CreateStatusProtocol() ProtocolHandler {
	handlers := make(map[byte]PacketHandler)
	handlers[0x00] = AutoPacketHandler(handleStatusRequest)
	handlers[0x01] = AutoPacketHandler(handlePingRequest)
	return MapProtocolHandler(handlers)
}

func handleStatusRequest(client *Client, _ *EmptyPacket) error {
	log.Println("Answering to status request")
	response := StatusResponse{
		Version: StatusVersion{
			"GoProtocol",
			int(client.ProtocolVersion),
		},
		Players:     StatusPlayers{Max: -1, Online: 6969},
		Description: types.ChatComponent{Text: "§cCoucou §bles §anoobs"},
		Favicon:     favicon,
	}
	return client.SendPacket(0x00, StatusResponsePacket{response})
}

func handlePingRequest(client *Client, packet *PingPacket) error {
	log.Println("Answering to ping request")

	if err := client.SendPacket(0x01, packet); err != nil {
		return err
	}

	return client.CloseConnection()
}
