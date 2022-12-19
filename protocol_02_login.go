package main

import (
	"crypto/md5"
	"fmt"
	"log"
)

type DisconnectPacket struct {
	Text ChatComponent `packet:"component"`
}

type LoginStartPacket struct {
	Username string `packet:"string"`
}

type LoginFinishPacket struct {
	Uuid     string `packet:"string"`
	Username string `packet:"string"`
}

func createLoginProtocol() func(packetId uint8) PacketHandler {
	handlers := make(map[uint8]PacketHandler)
	handlers[0x00] = wrapHandler(handleLoginStart)
	return func(packetId uint8) PacketHandler {
		return handlers[packetId]
	}
}

func createOfflineUuid(username string) string {
	digest := md5.Sum([]byte("OfflinePlayer:" + username))
	digest[6] &= 0x0F /* clear version        */
	digest[6] |= 0x30 /* set to version 3     */
	digest[8] &= 0x3F /* clear variant        */
	digest[8] |= 0x80 /* set to IETF variant  */
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		digest[0:4],
		digest[4:6],
		digest[6:8],
		digest[8:10],
		digest[10:16],
	)
}

func handleLoginStart(client *Client, packet LoginStartPacket) error {
	log.Println("answering to login start")

	client.identity = &Identity{createOfflineUuid(packet.Username), packet.Username}
	if err := client.sendPacket(0x02, LoginFinishPacket{client.identity.uuid, client.identity.username}); err != nil {
		return err
	}

	if err := client.switchProtocol(3); err != nil {
		return err
	}

	return client.join()
}
