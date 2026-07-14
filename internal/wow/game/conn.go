package game

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/WarhoopAll/wowchat/internal/config"
	"github.com/WarhoopAll/wowchat/internal/logger"
	"github.com/WarhoopAll/wowchat/internal/wow/crypto"
	"github.com/WarhoopAll/wowchat/internal/wow/protocol"
)

type incomingPacket struct {
	Opcode  int
	Payload []byte
}

type Conn struct {
	conn    net.Conn
	crypt   *crypto.HeaderCrypt
	writeMu sync.Mutex

	packets chan incomingPacket
	done    chan struct{}
	readErr error
}

func dialConn(cfg *config.Config, host string, port int) (*Conn, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	c, err := cfg.DialTCP(addr, cfg.Proxy.RealmConnect)
	if err != nil {
		return nil, err
	}
	return &Conn{
		conn:    c,
		crypt:   crypto.NewHeaderCrypt(),
		packets: make(chan incomingPacket, 64),
		done:    make(chan struct{}),
	}, nil
}

func (c *Conn) HeaderCrypt() *crypto.HeaderCrypt { return c.crypt }

func (c *Conn) Packets() <-chan incomingPacket { return c.packets }

func (c *Conn) Done() <-chan struct{} { return c.done }

func (c *Conn) ReadErr() error { return c.readErr }

func (c *Conn) Close() error { return c.conn.Close() }

func (c *Conn) startReadLoop() {
	go c.readLoop()
}

func (c *Conn) readLoop() {
	defer close(c.done)
	defer close(c.packets)
	var leftover []byte

	for {
		if len(leftover) < 4 {
			more, err := readSome(c.conn)
			if err != nil {
				c.readErr = err
				return
			}
			leftover = append(leftover, more...)
			continue
		}

		opcode, size, headerLen, err := c.parseHeader(leftover)
		if err == errNeedMore {
			more, rerr := readSome(c.conn)
			if rerr != nil {
				c.readErr = rerr
				return
			}
			leftover = append(leftover, more...)
			continue
		}
		if err != nil {
			c.readErr = err
			return
		}

		for len(leftover) < headerLen+size {
			more, rerr := readSome(c.conn)
			if rerr != nil {
				c.readErr = rerr
				return
			}
			leftover = append(leftover, more...)
		}

		body := leftover[headerLen : headerLen+size]
		payload := make([]byte, len(body))
		copy(payload, body)
		leftover = leftover[headerLen+size:]

		select {
		case c.packets <- incomingPacket{Opcode: opcode, Payload: payload}:
		case <-c.done:
			return
		}
	}
}

func (c *Conn) parseHeader(buf []byte) (opcode, size, headerLen int, err error) {
	if !c.crypt.Initialized() {
		if len(buf) < 4 {
			return 0, 0, 0, errNeedMore
		}
		s := int(uint16(buf[0])<<8 | uint16(buf[1]))
		s -= 2
		op := int(uint16(buf[3])<<8 | uint16(buf[2]))
		return op, s, 4, nil
	}
	hdr := c.crypt.Decrypt(buf[:4])
	if hdr[0]&0x80 != 0 {
		if len(buf) < 5 {
			return 0, 0, 0, errNeedMore
		}
		extra := c.crypt.Decrypt(buf[4:5])
		b4 := extra[0]
		size = int(uint32(hdr[0]&0x7F)<<16 | uint32(hdr[1])<<8 | uint32(hdr[2]))
		size -= 2
		opcode = int(uint16(b4)<<8 | uint16(hdr[3]))
		return opcode, size, 5, nil
	}
	size = int(uint32(hdr[0])<<8 | uint32(hdr[1]))
	size -= 2
	opcode = int(uint16(hdr[3])<<8 | uint16(hdr[2]))
	return opcode, size, 4, nil
}

var errNeedMore = errors.New("need more bytes")

func readSome(c net.Conn) ([]byte, error) {
	buf := make([]byte, 4096)
	n, err := c.Read(buf)
	if n > 0 {
		out := buf[:n]
		if err == io.EOF {
			return out, io.ErrUnexpectedEOF
		}
		return out, err
	}
	if err == nil {
		err = io.ErrUnexpectedEOF
	}
	return nil, err
}

func (c *Conn) Write(opcode int, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	unencrypted := opcode == OpCMSGAuthChallenge
	headerSize := 4
	if !unencrypted {
		headerSize = 6
	}

	bodyLen := len(payload)
	hdr := []byte{
		byte((bodyLen + headerSize - 2) >> 8),
		byte(bodyLen + headerSize - 2),
		byte(opcode),
		byte(opcode >> 8),
	}
	if !unencrypted {
		hdr = append(hdr, 0x00, 0x00)
		hdr = c.crypt.Encrypt(hdr)
	}

	out := make([]byte, 0, len(hdr)+len(payload))
	out = append(out, hdr...)
	out = append(out, payload...)
	logger.WoW().Debug("SEND", "opcode", fmt.Sprintf("%04X", opcode), "body", bodyLen)
	_, err := c.conn.Write(out)
	return err
}

func (c *Conn) WriteEmpty(opcode int) error { return c.Write(opcode, nil) }

func (c *Conn) WriteShortPayload(opcode int, w *protocol.PacketWriter) error {
	return c.Write(opcode, w.Bytes())
}
