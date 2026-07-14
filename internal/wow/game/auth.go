package game

import (
	"fmt"
	"math/rand/v2"

	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/crypto"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

func (s *Session) handleAuthChallenge(r *protocol.PacketReader) error {
	account := []byte(s.cfg.WoW.Account)

	r.Skip(4)
	serverSeed := r.MustInt()
	clientSeed := rand.Int32()

	w := protocol.NewPacketWriter()
	w.WriteShortLE(0)
	w.WriteIntLE(int32(s.cfg.WoW.Build))
	w.WriteIntLE(0)
	w.WriteBytes(account)
	w.PutByte(0)
	w.WriteIntBE(0)
	w.WriteIntBE(clientSeed)
	w.WriteIntLE(0)
	w.WriteIntLE(0)
	w.WriteIntLE(int32(s.realmID))
	w.WriteLongLE(3)

	digest := crypto.SHA1(
		account,
		[]byte{0, 0, 0, 0},
		protocol.IntBE(clientSeed),
		protocol.IntBE(serverSeed),
		s.sessionKey,
	)
	w.WriteBytes(digest)
	w.WriteBytes(addonInfoWotLK)

	s.conn.HeaderCrypt().Init(s.sessionKey)

	return s.conn.Write(OpCMSGAuthChallenge, w.Bytes())
}

func (s *Session) handleCharEnum(r *protocol.PacketReader) error {
	want := lowercaseUTF8(s.cfg.WoW.Character)
	num := int(r.MustByte())
	for i := 0; i < num; i++ {
		guid := uint64(r.MustLongLE())
		name := r.MustString()
		race := r.MustByte()
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
		r.Skip(1)
		r.Skip(4)
		r.Skip(4)
		r.Skip(12)
		r.MustIntLE()

		if equalFoldUTF8(name, want) {
			s.selfCharID = guid
			s.hasCharID = true
			s.languageID = raceLanguage(race)
			logger.WoW().Info("logging in with character", "name", name)
			out := protocol.NewPacketWriter()
			out.WriteLongLE(int64(guid))
			return s.conn.Write(OpCMSGPlayerLogin, out.Bytes())
		}
		r.Skip(4)
		r.Skip(4)
		r.Skip(1)
		r.Skip(12)
		r.Skip(19 * 9)
		r.Skip(4 * 9)
	}
	return fmt.Errorf("character %q not found", s.cfg.WoW.Character)
}

func lowercaseUTF8(s string) []byte {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return b
}

func equalFoldUTF8(name string, want []byte) bool {
	return bytesEqual(lowercaseUTF8(name), want)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
