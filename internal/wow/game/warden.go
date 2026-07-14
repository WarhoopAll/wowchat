package game

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strconv"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/crypto"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

const (
	wardenSMSGModuleUse        byte = 0x00
	wardenSMSGModuleCache      byte = 0x01
	wardenSMSGCheatChecksReq   byte = 0x02
	wardenSMSGModuleInitialize byte = 0x03
	wardenSMSGMemChecksReq     byte = 0x04
	wardenSMSGHashRequest      byte = 0x05
)

const (
	wardenCMSGModuleOK          byte = 0x01
	wardenCMSGCheatChecksResult byte = 0x02
	wardenCMSGHashResult        byte = 0x04
)

type wardenHandler struct {
	sessionKey  []byte
	clientCrypt *crypto.RC4
	serverCrypt *crypto.RC4
	moduleCrypt *crypto.RC4
	moduleName  string
	moduleSeed  []byte
	moduleLen   int
	moduleBuf   *bytes.Buffer
}

func newWardenHandler(sessionKey []byte) *wardenHandler {
	r := crypto.NewSHA1Randx(sessionKey)
	clientKey := r.Generate(16)
	serverKey := r.Generate(16)
	return &wardenHandler{
		sessionKey:  sessionKey,
		clientCrypt: crypto.NewRC4(clientKey),
		serverCrypt: crypto.NewRC4(serverKey),
		moduleSeed:  make([]byte, 16),
	}
}

func (s *Session) handleWardenData(r *protocol.PacketReader) error {
	if s.cfg.WoW.Platform != config.PlatformMac {
		return fmt.Errorf("warden on %s is not supported; use WOW_PLATFORM=Mac", s.cfg.WoW.Platform)
	}
	if s.warden == nil {
		s.warden = newWardenHandler(s.sessionKey)
		logger.WoW().Info("warden handling initialized")
	}

	id, resp, err := s.warden.handle(r.Bytes())
	if err != nil {
		return err
	}
	if resp != nil {
		if err := s.conn.Write(OpCMSGWardenData, resp); err != nil {
			return err
		}
		if id == wardenSMSGHashRequest {
			if err := s.conn.WriteEmpty(OpCMSGCharEnum); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *wardenHandler) handle(msg []byte) (byte, []byte, error) {
	decrypted := w.serverCrypt.Crypt(msg)
	dr := protocol.NewPacketReader(decrypted)
	id, _ := dr.ReadByte()
	logger.WoW().Debug("WARDEN recv", "id", fmt.Sprintf("0x%02X", id), "bytes", len(decrypted))

	var resp *protocol.PacketWriter
	switch id {
	case wardenSMSGModuleUse:
		resp = w.handleModuleUse(dr)
	case wardenSMSGModuleCache:
		resp = w.handleModuleCache(dr)
	case wardenSMSGCheatChecksReq:
		resp = w.handleCheatChecks(dr)
	case wardenSMSGHashRequest:
		resp = w.handleHashRequest(dr)
	case wardenSMSGModuleInitialize, wardenSMSGMemChecksReq:
		resp = nil
	default:
		logger.WoW().Debug("WARDEN unhandled", "id", fmt.Sprintf("0x%02X", id))
	}
	if resp == nil {
		return id, nil, nil
	}
	enc := w.clientCrypt.Crypt(resp.Bytes())
	return id, enc, nil
}

func (w *wardenHandler) handleModuleUse(r *protocol.PacketReader) *protocol.PacketWriter {
	nameBytes, _ := r.ReadBytes(16)
	w.moduleName = ""
	for _, b := range nameBytes {
		w.moduleName += strconv.FormatUint(uint64(b), 16)
	}
	seed, _ := r.ReadBytes(16)
	copy(w.moduleSeed, seed)
	w.moduleLen = int(r.MustIntLE())
	w.moduleCrypt = crypto.NewRC4(w.moduleSeed)
	return w.moduleOK()
}

func (w *wardenHandler) moduleOK() *protocol.PacketWriter {
	out := protocol.NewPacketWriter()
	out.PutByte(wardenCMSGModuleOK)
	return out
}

func (w *wardenHandler) handleModuleCache(r *protocol.PacketReader) *protocol.PacketWriter {
	length := int(r.MustShortLE())
	chunk, _ := r.ReadBytes(length)
	if w.moduleBuf == nil {
		w.moduleBuf = new(bytes.Buffer)
	}
	w.moduleBuf.Write(chunk)
	if w.moduleBuf.Len() < w.moduleLen {
		return nil
	}
	mod := w.moduleCrypt.Crypt(w.moduleBuf.Bytes())
	if len(mod) < 4 {
		return nil
	}
	decompressedLen := int(uint32(mod[0]) | uint32(mod[1])<<8 | uint32(mod[2])<<16 | uint32(mod[3])<<24)
	zr, err := zlib.NewReader(bytes.NewReader(mod[4:]))
	if err != nil {
		logger.WoW().Error("WARDEN module inflate", "err", err)
		return nil
	}
	if _, err := io.Copy(io.Discard, zr); err != nil {
		logger.WoW().Error("WARDEN module inflate read", "err", err)
	}
	zr.Close()
	_ = decompressedLen
	w.moduleBuf.Reset()
	return w.moduleOK()
}

func (w *wardenHandler) handleCheatChecks(r *protocol.PacketReader) *protocol.PacketWriter {
	strLen := int(r.MustByte())
	strArray, _ := r.ReadBytes(strLen)
	key := strArray
	for i, b := range key {
		if b == 0 {
			key = key[:i]
			break
		}
	}

	out := protocol.NewPacketWriter()
	out.PutByte(wardenCMSGCheatChecksResult)
	feedFace := uint32(0xFEEDFACE)
	seedBytes := []byte{byte(feedFace), byte(feedFace >> 8), byte(feedFace >> 16), byte(feedFace >> 24)}
	out.WriteBytes(crypto.SHA1(key, seedBytes))
	out.WriteBytes(crypto.MD5(key))
	return out
}

func (w *wardenHandler) handleHashRequest(r *protocol.PacketReader) *protocol.PacketWriter {
	clientKey := [4]uint32{
		uint32(r.MustIntLE()), uint32(r.MustIntLE()),
		uint32(r.MustIntLE()), uint32(r.MustIntLE()),
	}
	serverKey := [4]uint32{}

	serverKey[0] = clientKey[0]
	clientKey[0] ^= 0xDEADBEEF
	serverKey[1] = clientKey[1]
	clientKey[1] -= 0x35014542
	serverKey[2] = clientKey[2]
	clientKey[2] += 0x5313F22
	clientKey[3] *= 0x1337F00D
	serverKey[1] = serverKey[1] - 0x6A028A84
	serverKey[2] = serverKey[2] + 0xA627E44
	serverKey[3] = 0x1337F00D * clientKey[3]

	clientKeyBytes := make([]byte, 16)
	for i, k := range clientKey {
		clientKeyBytes[i*4+0] = byte(k)
		clientKeyBytes[i*4+1] = byte(k >> 8)
		clientKeyBytes[i*4+2] = byte(k >> 16)
		clientKeyBytes[i*4+3] = byte(k >> 24)
	}

	out := protocol.NewPacketWriter()
	out.PutByte(wardenCMSGHashResult)
	out.WriteBytes(crypto.SHA1(clientKeyBytes))

	serverKeyBytes := make([]byte, 16)
	for i, k := range serverKey {
		serverKeyBytes[i*4+0] = byte(k)
		serverKeyBytes[i*4+1] = byte(k >> 8)
		serverKeyBytes[i*4+2] = byte(k >> 16)
		serverKeyBytes[i*4+3] = byte(k >> 24)
	}
	w.clientCrypt = crypto.NewRC4(clientKeyBytes)
	w.serverCrypt = crypto.NewRC4(serverKeyBytes)
	return out
}
