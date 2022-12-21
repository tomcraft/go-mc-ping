package internal

import (
	"_tomcraft/go-mc-ping/internal/types"
	"crypto/md5"
	"fmt"
	"log"
)

type DisconnectPacket struct {
	Text types.ChatComponent `packet:"component"`
}

type LoginStartPacket struct {
	Username string `packet:"string"`
}

type LoginFinishPacket struct {
	Uuid     string `packet:"string"`
	Username string `packet:"string"`
}

func CreateLoginProtocol() ProtocolHandler {
	handlers := make(map[byte]PacketHandler)
	handlers[0x00] = AutoPacketHandler(handleLoginStart)
	return MapProtocolHandler(handlers)
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

func handleLoginStart(client *Client, packet *LoginStartPacket) error {
	log.Println("Answering to login start")

	client.Identity = &types.Identity{
		Uuid:     createOfflineUuid(packet.Username),
		Username: packet.Username,
	}
	if err := client.SendPacket(0x02, LoginFinishPacket{client.Identity.Uuid, client.Identity.Username}); err != nil {
		return err
	}

	if err := client.SwitchProtocol(3); err != nil {
		return err
	}

	return client.Join()
}
