package game

import (
	"errors"

	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

func (s *Session) SendChatMessage(tp byte, message string, channel *string) error {
	if !s.inWorld {
		return errors.New("not connected to WoW")
	}
	w := protocol.NewPacketWriter()
	w.WriteIntLE(int32(tp))
	w.WriteIntLE(int32(s.languageID))
	if channel != nil {
		w.WriteBytes([]byte(*channel))
		w.PutByte(0)
	}
	w.WriteBytes([]byte(message))
	w.PutByte(0)
	return s.conn.Write(OpCMSGMessageChat, w.Bytes())
}
