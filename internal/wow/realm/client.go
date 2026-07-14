package realm

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/crypto"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

type Result struct {
	Host       string
	Port       int
	RealmID    int
	RealmName  string
	SessionKey []byte
}

type Client struct {
	cfg    *config.Config
	conn   net.Conn
	srp    *crypto.SRPClient
	locale string
}

func New(cfg *config.Config, locale string) *Client {
	if locale == "" {
		locale = "enUS"
	}
	return &Client{cfg: cfg, locale: locale}
}

func (c *Client) Login() (*Result, error) {
	addr := net.JoinHostPort(c.cfg.WoW.Host, fmt.Sprintf("%d", c.cfg.WoW.Port))
	logger.Realm().Info("connecting to realm server", "addr", addr)
	conn, err := c.cfg.DialTCP(addr, c.cfg.Proxy.RealmConnect)
	if err != nil {
		return nil, fmt.Errorf("dial realm: %w", err)
	}
	c.conn = conn

	if err := c.sendChallenge(); err != nil {
		return nil, c.closeErr(err)
	}

	challenge, err := c.readChallenge()
	if err != nil {
		return nil, c.closeErr(err)
	}

	if err := c.handleChallenge(challenge); err != nil {
		return nil, c.closeErr(err)
	}

	proof, err := c.readProof()
	if err != nil {
		return nil, c.closeErr(err)
	}
	if err := c.handleProof(proof); err != nil {
		return nil, c.closeErr(err)
	}

	if err := c.sendRealmListRequest(); err != nil {
		return nil, c.closeErr(err)
	}

	realms, err := c.readRealmList()
	if err != nil {
		return nil, c.closeErr(err)
	}

	selected := c.selectRealm(realms)
	if selected == nil {
		var names []string
		for _, r := range realms {
			names = append(names, r.Name)
		}
		return nil, c.closeErr(fmt.Errorf("realm %q not found (%d realms available: %s)",
			c.cfg.WoW.Realm, len(realms), strings.Join(names, ", ")))
	}

	res := &Result{
		Host:       selected.Address,
		Port:       selected.Port,
		RealmID:    selected.RealmID,
		RealmName:  selected.Name,
		SessionKey: c.srp.SessionKey(),
	}
	c.conn.Close()
	logger.Realm().Info("logged into realm server", "realm", selected.Name, "host", selected.Address, "port", selected.Port)
	return res, nil
}

func (c *Client) closeErr(err error) error {
	if c.conn != nil {
		c.conn.Close()
	}
	return err
}

func (c *Client) sendChallenge() error {
	account := []byte(c.cfg.WoW.Account)
	w := protocol.NewPacketWriter()
	w.PutByte(8)
	w.WriteShortLE(30 + len(account))
	w.WriteIntLE(int32(protocol.StringToInt("WoW")))
	v := versionBytes(c.cfg.WoW.Version)
	w.PutByte(v[0])
	w.PutByte(v[1])
	w.PutByte(v[2])
	w.WriteShortLE(c.cfg.WoW.Build)
	w.WriteIntLE(int32(protocol.StringToInt("x86")))
	w.WriteIntLE(int32(protocol.StringToInt(platformStr(c.cfg.WoW.Platform))))
	w.WriteIntLE(int32(protocol.StringToInt(c.locale)))
	w.WriteIntLE(0)
	w.PutByte(127)
	w.PutByte(0)
	w.PutByte(0)
	w.PutByte(1)
	w.PutByte(byte(len(account)))
	w.WriteBytes(account)

	return c.writePacket(protocol.CmdAuthLogonChallenge, w.Bytes())
}

func versionBytes(v string) [3]byte {
	parts := strings.Split(v, ".")
	var out [3]byte
	for i := 0; i < 3 && i < len(parts); i++ {
		n := 0
		for _, ch := range parts[i] {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
		}
		out[i] = byte(n)
	}
	return out
}

func platformStr(p config.Platform) string {
	if p == config.PlatformWindows {
		return "Win"
	}
	return "OSX"
}

func (c *Client) readChallenge() (*protocol.PacketReader, error) {
	return c.readRealmPacket(protocol.CmdAuthLogonChallenge, func(id byte, r *protocol.PacketReader) (int, error) {
		if r.Remaining() < 2 {
			return 0, protocol.ErrShortBuffer
		}
		buf := r.Bytes()
		result := buf[1]
		if protocol.AuthIsSuccess(result) {
			needed := 2 + 115 + 1
			if r.Remaining() < needed {
				return 0, protocol.ErrShortBuffer
			}
			secFlags := buf[needed-1]
			secLen := 0
			if secFlags&0x01 != 0 {
				secLen += 20
			}
			if secFlags&0x02 != 0 {
				secLen += 12
			}
			if secFlags&0x04 != 0 {
				secLen += 1
			}
			return 118 + secLen, nil
		}
		return 2, nil
	})
}

func (c *Client) handleChallenge(r *protocol.PacketReader) error {
	r.MustByte()
	result := r.MustByte()
	if !protocol.AuthIsSuccess(result) {
		return fmt.Errorf("realm auth: %s", protocol.AuthMessage(result))
	}

	Bbytes := r.MustBytes(32)
	gLen := int(r.MustByte())
	gBytes := r.MustBytes(gLen)
	nLen := int(r.MustByte())
	nBytes := r.MustBytes(nLen)
	saltBytes := r.MustBytes(32)
	r.Skip(16)
	securityFlag := r.MustByte()

	var token string
	if securityFlag == 0x04 {
		tokLen := int(r.MustByte())
		tb := r.MustBytes(tokLen)
		token = string(tb)
		logger.Realm().Info("two-factor token required", "len", tokLen)
		_ = token
	} else if securityFlag != 0x00 {
		return fmt.Errorf("unsupported two-factor auth type 0x%02X (disable it or use another account)", securityFlag)
	}

	B := crypto.NewBigFromBytes(Bbytes, true)
	g := crypto.NewBigFromBytes(gBytes, true)
	N := crypto.NewBigFromBytes(nBytes, true)
	salt := crypto.NewBigFromBytes(saltBytes, true)

	srp, err := crypto.NewSRPClient(nil)
	if err != nil {
		return err
	}
	c.srp = srp
	srp.Step1([]byte(c.cfg.WoW.Account), c.cfg.WoW.Password, B, g, N, salt)

	A := srp.ClientPublic()
	M := srp.ClientProof()
	crcHash, ok := crcHashFor(c.cfg.WoW.Build, c.cfg.WoW.Platform)
	if !ok {
		logger.Realm().Warn("no CRC hash for build; sending zeros", "build", c.cfg.WoW.Build, "platform", c.cfg.WoW.Platform)
		crcHash = make([]byte, 20)
	}
	proofDigest := crypto.SHA1(A, crcHash)

	out := protocol.NewPacketWriter()
	out.WriteBytes(A)
	out.WriteBytes(M)
	out.WriteBytes(proofDigest)
	out.PutByte(0)
	out.PutByte(securityFlag)
	if securityFlag == 0x04 {
		tb := []byte(token)
		out.PutByte(byte(len(tb)))
		out.WriteBytes(tb)
	}
	logger.Realm().Debug("sending logon proof", "srp", "A+M+CRC")
	return c.writePacket(protocol.CmdAuthLogonProof, out.Bytes())
}

func (c *Client) readProof() (*protocol.PacketReader, error) {
	return c.readRealmPacket(protocol.CmdAuthLogonProof, func(id byte, r *protocol.PacketReader) (int, error) {
		if r.Remaining() < 1 {
			return 0, protocol.ErrShortBuffer
		}
		result := r.Bytes()[0]
		if protocol.AuthIsSuccess(result) {
			return 31, nil
		}
		if r.Remaining() == 0 {
			return 1, nil
		}
		return 3, nil
	})
}

func (c *Client) handleProof(r *protocol.PacketReader) error {
	result := r.MustByte()
	if !protocol.AuthIsSuccess(result) {
		msg := protocol.AuthMessage(result)
		if result == protocol.AuthFailUnknownAcct {
			return fmt.Errorf("realm auth: %s (will retry)", msg)
		}
		return fmt.Errorf("realm auth: %s", msg)
	}
	serverProof := r.MustBytes(20)
	expect := c.srp.GenerateHashLogonProof()
	if !bytesEqual(serverProof, expect) {
		return errors.New("logon proof generated by client and server differ")
	}
	logger.Realm().Info("logged into realm server", "target", c.cfg.WoW.Realm)
	return nil
}

func (c *Client) sendRealmListRequest() error {
	w := protocol.NewPacketWriter()
	w.WriteIntLE(0)
	return c.writePacket(protocol.CmdRealmList, w.Bytes())
}

func (c *Client) readRealmList() ([]realmEntry, error) {
	r, err := c.readRealmPacket(protocol.CmdRealmList, func(id byte, pr *protocol.PacketReader) (int, error) {
		if pr.Remaining() < 2 {
			return 0, protocol.ErrShortBuffer
		}
		size := int(uint16(pr.Bytes()[0]) | uint16(pr.Bytes()[1])<<8)
		return 2 + size, nil
	})
	if err != nil {
		return nil, err
	}
	r.MustShortLE()
	return parseRealmListTBC(r), nil
}

type realmEntry struct {
	Name    string
	Address string
	Port    int
	RealmID int
}

func parseRealmListTBC(r *protocol.PacketReader) []realmEntry {
	r.MustIntLE()
	num := int(r.MustShortLE())
	out := make([]realmEntry, 0, num)
	for i := 0; i < num; i++ {
		r.Skip(1)
		r.Skip(1)
		realmFlags := r.MustByte()
		name := r.MustString()
		address := r.MustString()
		r.Skip(4)
		r.Skip(1)
		r.Skip(1)
		realmID := int(r.MustByte())
		if realmFlags&0x04 == 0x04 {
			r.Skip(5)
		}
		host, portStr := splitHostPort(address)
		port := portFromString(portStr)
		out = append(out, realmEntry{Name: name, Address: host, Port: port, RealmID: realmID})
	}
	return out
}

func (c *Client) selectRealm(realms []realmEntry) *realmEntry {
	for i := range realms {
		if strings.EqualFold(realms[i].Name, c.cfg.WoW.Realm) {
			return &realms[i]
		}
	}
	return nil
}

func (c *Client) writePacket(opcode byte, payload []byte) error {
	buf := make([]byte, 0, 1+len(payload))
	buf = append(buf, opcode)
	buf = append(buf, payload...)
	_, err := c.conn.Write(buf)
	return err
}

type sizeFunc func(id byte, r *protocol.PacketReader) (int, error)

func (c *Client) readRealmPacket(expected byte, sf sizeFunc) (*protocol.PacketReader, error) {
	var buf []byte
	for {
		if len(buf) == 0 {
			b := make([]byte, 1)
			if _, err := io.ReadFull(c.conn, b); err != nil {
				return nil, err
			}
			buf = append(buf, b...)
		}
		opcode := buf[0]
		if opcode != expected {
			return nil, fmt.Errorf("unexpected realm opcode 0x%02X (want 0x%02X)", opcode, expected)
		}
		pr := protocol.NewPacketReader(buf[1:])
		size, err := sf(opcode, pr)
		if err == protocol.ErrShortBuffer {
			more := make([]byte, 256)
			n, rerr := c.conn.Read(more)
			if rerr != nil {
				return nil, rerr
			}
			buf = append(buf, more[:n]...)
			continue
		}
		if err != nil {
			return nil, err
		}
		need := 1 + size
		for len(buf) < need {
			more := make([]byte, need-len(buf))
			if _, err := io.ReadFull(c.conn, more); err != nil {
				return nil, err
			}
			buf = append(buf, more...)
		}
		payload := buf[1 : 1+size]
		out := make([]byte, len(payload))
		copy(out, payload)
		return protocol.NewPacketReader(out), nil
	}
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

func splitHostPort(address string) (string, string) {
	address = strings.TrimSpace(address)
	idx := strings.LastIndex(address, ":")
	if idx < 0 {
		return address, "8085"
	}
	return strings.TrimSpace(address[:idx]), strings.TrimSpace(address[idx+1:])
}

func portFromString(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	return n & 0xFFFF
}
