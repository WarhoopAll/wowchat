package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

type DiscordRelay interface {
	SendFromWoW(from *string, message string, wowType byte, channel *string)
	WantsWoW(wowType byte, channel *string) bool
}

type Session struct {
	cfg          *config.Config
	realmID      int
	realmName    string
	sessionKey   []byte
	conn         *Conn
	selfCharID   uint64
	hasCharID    bool
	languageID   byte
	inWorld      bool
	warden       *wardenHandler
	connectTime  time.Time
	joinChannels []string
	joined       bool
	relay        DiscordRelay
	rosterMu     sync.Mutex
	playerNames  map[uint64]string
	pendingChats map[uint64][]pendingChat
}

type pendingChat struct {
	tp      byte
	message string
	channel *string
}

func NewSession(cfg *config.Config, realmID int, realmName string, sessionKey []byte) *Session {
	return &Session{
		cfg:          cfg,
		realmID:      realmID,
		realmName:    realmName,
		sessionKey:   sessionKey,
		joinChannels: cfg.Channels,
		playerNames:  map[uint64]string{},
		pendingChats: map[uint64][]pendingChat{},
	}
}

func (s *Session) SetRelay(r DiscordRelay) { s.relay = r }

func (s *Session) Run(ctx context.Context, host string, port int) error {
	c, err := dialConn(s.cfg, host, port)
	if err != nil {
		return fmt.Errorf("dial game: %w", err)
	}
	s.conn = c
	logger.WoW().Info("connected to game server", "host", host, "port", port)
	c.startReadLoop()

	maintCtx, cancelMaint := context.WithCancel(ctx)
	defer cancelMaint()

	for {
		select {
		case <-ctx.Done():
			c.Close()
			return ctx.Err()
		case <-c.Done():
			if c.readErr != nil {
				return fmt.Errorf("game connection closed: %w", c.readErr)
			}
			return errors.New("game connection closed")
		case pkt := <-c.Packets():
			if err := s.dispatch(maintCtx, pkt); err != nil {
				logger.WoW().Error("handler error", "opcode", fmt.Sprintf("%04X", pkt.Opcode), "err", err)
			}
		}
	}
}

func (s *Session) dispatch(ctx context.Context, pkt incomingPacket) error {
	r := protocol.NewPacketReader(pkt.Payload)
	switch pkt.Opcode {
	case OpSMSGAuthChallenge:
		return s.handleAuthChallenge(r)
	case OpSMSGAuthResponse:
		return s.handleAuthResponse(r)
	case OpSMSGCharEnum:
		return s.handleCharEnum(r)
	case OpSMSGLoginVerifyWorld:
		s.handleLoginVerifyWorld(ctx)
	case OpSMSGMessageChat, OpSMSGGMMessageChat:
		s.handleChat(pkt.Opcode, r)
	case OpSMSGNameQuery:
		s.handleNameQuery(r)
	case OpSMSGChannelNotify:
		s.handleChannelNotify(r)
	case OpSMSGNotification:
		logger.WoW().Info("notification", "msg", r.MustString())
	case OpSMSGTimeSyncReq:
		return s.handleTimeSyncReq(r)
	case OpSMSGWardenData:
		return s.handleWardenData(r)
	}
	return nil
}

func (s *Session) handleAuthResponse(r *protocol.PacketReader) error {
	code := r.MustByte()
	if code == AuthResponseOK {
		logger.WoW().Info("logged into world session")
		return s.conn.WriteEmpty(OpCMSGCharEnum)
	}
	if code == AuthResponseWaitQueue {
		if r.Remaining() >= 14 {
			r.Skip(10)
		}
		pos := r.MustIntLE()
		logger.WoW().Warn("queue enabled", "position", pos)
		return nil
	}
	return fmt.Errorf("game auth failed: code 0x%02X", code)
}
