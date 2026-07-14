package game

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

func (s *Session) startMaintenance(ctx context.Context) {
	go s.runPing(ctx)
	go s.runKeepAlive(ctx)
}

func (s *Session) runPing(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	id := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		latency := rand.IntN(50) + 90
		w := protocol.NewPacketWriter()
		w.WriteIntLE(int32(id))
		w.WriteIntLE(int32(latency))
		id++
		if err := s.conn.Write(OpCMSGPing, w.Bytes()); err != nil {
			logger.WoW().Error("ping", "err", err)
			return
		}
	}
}

func (s *Session) runKeepAlive(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(15 * time.Second):
	}
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if err := s.conn.WriteEmpty(OpCMSGKeepAlive); err != nil {
			logger.WoW().Error("keep-alive", "err", err)
			return
		}
	}
}

func (s *Session) handleTimeSyncReq(r *protocol.PacketReader) error {
	counter := r.MustIntLE()
	uptime := int32(time.Since(s.connectTime).Seconds())
	w := protocol.NewPacketWriter()
	w.WriteIntLE(counter)
	w.WriteIntLE(uptime)
	return s.conn.Write(OpCMSGTimeSyncResp, w.Bytes())
}
