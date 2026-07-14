package game

import (
	"time"

	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

type chatMessage struct {
	guid    uint64
	tp      byte
	message string
	channel *string
}

func (s *Session) handleChat(opcode int, r *protocol.PacketReader) {
	tp := r.MustByte()
	lang := r.MustIntLE()
	if lang == -1 {
		return
	}

	senderGUID := uint64(r.MustLongLE())
	if tp != ChatMsgSystem && s.hasCharID && senderGUID == s.selfCharID {
		return
	}

	r.Skip(4)

	if opcode == OpSMSGGMMessageChat {
		r.Skip(4)
		r.SkipString()
	}

	var channel *string
	if tp == ChatMsgChannel {
		name := r.MustString()
		channel = &name
	}

	r.Skip(8)

	txtLen := r.MustIntLE()
	if int(txtLen)-1 > r.Remaining() || txtLen < 1 {
		logger.WoW().Debug("malformed text length", "len", txtLen, "remaining", r.Remaining())
		return
	}
	textBytes, _ := r.ReadBytes(int(txtLen - 1))
	r.Skip(1)
	if r.Remaining() > 0 {
		r.Skip(1)
	}

	s.logChat(tp, channel, senderGUID, string(textBytes))
	s.sendChatMessage(chatMessage{guid: senderGUID, tp: tp, message: string(textBytes), channel: channel})
}

func (s *Session) logChat(tp byte, channel *string, guid uint64, text string) {
	ty := ChatTypeName(tp)
	if channel != nil {
		logger.Chat().Info("recv", "type", ty, "channel", *channel, "guid", guid, "msg", text, "time", time.Now().Format("15:04:05"))
	} else {
		logger.Chat().Info("recv", "type", ty, "guid", guid, "msg", text, "time", time.Now().Format("15:04:05"))
	}
}

func (s *Session) sendChatMessage(cm chatMessage) {
	if s.relay == nil {
		return
	}
	if !s.relay.WantsWoW(cm.tp, cm.channel) {
		return
	}

	if cm.guid == 0 {
		s.relay.SendFromWoW(nil, cm.message, cm.tp, cm.channel)
		return
	}

	s.rosterMu.Lock()
	if name, ok := s.playerNames[cm.guid]; ok {
		s.rosterMu.Unlock()
		s.relay.SendFromWoW(&name, cm.message, cm.tp, cm.channel)
		return
	}
	_, alreadyQueued := s.pendingChats[cm.guid]
	s.pendingChats[cm.guid] = append(s.pendingChats[cm.guid], pendingChat{tp: cm.tp, message: cm.message, channel: cm.channel})
	s.rosterMu.Unlock()

	if !alreadyQueued {
		s.sendNameQuery(cm.guid)
	}
}

func (s *Session) sendNameQuery(guid uint64) {
	w := protocol.NewPacketWriter()
	w.WriteLongLE(int64(guid))
	if err := s.conn.Write(OpCMSGNameQuery, w.Bytes()); err != nil {
		logger.WoW().Error("name query", "err", err)
	}
}

func (s *Session) handleNameQuery(r *protocol.PacketReader) {
	guid := readPackedGUID(r)
	nameKnown := r.MustByte()
	name := "UNKNOWN"
	if nameKnown == 0 {
		name = r.MustString()
		r.SkipString()
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
	}

	s.rosterMu.Lock()
	queued := s.pendingChats[guid]
	delete(s.pendingChats, guid)
	if nameKnown == 0 {
		s.playerNames[guid] = name
	}
	s.rosterMu.Unlock()

	if s.relay == nil {
		return
	}
	for _, pc := range queued {
		n := name
		s.relay.SendFromWoW(&n, pc.message, pc.tp, pc.channel)
	}
}

func readPackedGUID(r *protocol.PacketReader) uint64 {
	mask := r.MustByte()
	var result uint64
	for i := 0; i < 8; i++ {
		if mask&(1<<uint(i)) != 0 {
			b := r.MustByte()
			result |= uint64(b) << (uint(i) * 8)
		}
	}
	return result
}
