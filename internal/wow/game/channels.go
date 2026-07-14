package game

import (
	"context"
	"time"

	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

func (s *Session) handleLoginVerifyWorld(ctx context.Context) {
	if s.inWorld {
		return
	}
	s.inWorld = true
	s.connectTime = time.Now()
	logger.WoW().Info("entered the world", "realm", s.realmName)
	s.startMaintenance(ctx)
	s.joinConfiguredChannels()
}

func (s *Session) joinConfiguredChannels() {
	if s.joined {
		return
	}
	s.joined = true
	for _, name := range s.joinChannels {
		id := ChannelID(name)
		logger.WoW().Info("joining channel", "name", name, "id", id)
		w := protocol.NewPacketWriter()
		w.WriteIntLE(int32(id))
		w.PutByte(0)
		w.PutByte(1)
		w.WriteBytes([]byte(name))
		w.PutByte(0)
		w.PutByte(0)
		if err := s.conn.Write(OpCMSGJoinChannel, w.Bytes()); err != nil {
			logger.WoW().Error("join channel", "name", name, "err", err)
		}
	}
}

func (s *Session) handleChannelNotify(r *protocol.PacketReader) {
	id := r.MustByte()
	name := r.MustString()
	if id == ChatNotifyYouJoined {
		logger.WoW().Info("joined channel", "name", name)
	}
}
