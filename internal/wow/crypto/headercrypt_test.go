package crypto

import (
	"bytes"
	"testing"
)

func TestHeaderCryptDirectionalStreamsAgree(t *testing.T) {
	session := bytes.Repeat([]byte{0x42}, 40)

	a := NewHeaderCrypt()
	b := NewHeaderCrypt()
	a.Init(session)
	b.Init(session)

	headers := [][]byte{
		{0x01, 0x02, 0x03, 0x04},
		{0x10, 0x20, 0x30, 0x40, 0x50},
		{0xAA, 0xBB, 0xCC, 0xDD},
	}
	for i, hdr := range headers {
		aEnc := a.Encrypt(hdr)
		bEnc := b.Encrypt(hdr)
		if !bytes.Equal(aEnc, bEnc) {
			t.Errorf("header %d: client streams disagree\n a % x\n b % x", i, aEnc, bEnc)
		}

		aDec := a.Decrypt(hdr)
		bDec := b.Decrypt(hdr)
		if !bytes.Equal(aDec, bDec) {
			t.Errorf("header %d: server streams disagree\n a % x\n b % x", i, aDec, bDec)
		}
	}
}

func TestHeaderCryptClientStreamReversibility(t *testing.T) {
	session := bytes.Repeat([]byte{0x42}, 40)

	sender := NewHeaderCrypt()
	sender.Init(session)

	matchKey := HMACSHA1(clientHMACSeed, session)
	matcher := NewRC4(matchKey)
	matcher.Discard(1024)

	in := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	enc := sender.Encrypt(in)
	rev := matcher.Crypt(enc)
	if !bytes.Equal(rev, in) {
		t.Errorf("client stream not reversible\n in  % x\n rev % x", in, rev)
	}

	serverMatchKey := HMACSHA1(serverHMACSeed, session)
	serverMatcher := NewRC4(serverMatchKey)
	serverMatcher.Discard(1024)
	in2 := []byte{0x10, 0x20, 0x30, 0x40}
	dec := sender.Decrypt(in2)
	rev2 := serverMatcher.Crypt(dec)
	if !bytes.Equal(rev2, in2) {
		t.Errorf("server stream not reversible\n in  % x\n rev % x", in2, rev2)
	}
}

func TestHeaderCryptUninitialized(t *testing.T) {
	h := NewHeaderCrypt()
	in := []byte{1, 2, 3, 4}
	out := h.Encrypt(in)
	if !bytes.Equal(out, in) {
		t.Errorf("pre-Init Encrypt modified data: % x", out)
	}
	out = h.Decrypt(in)
	if !bytes.Equal(out, in) {
		t.Errorf("pre-Init Decrypt modified data: % x", out)
	}
}

func TestHeaderCryptDirectionsDistinct(t *testing.T) {
	session := bytes.Repeat([]byte{0x07}, 40)
	h := NewHeaderCrypt()
	h.Init(session)
	in := []byte{0x01, 0x02, 0x03, 0x04}
	if bytes.Equal(h.Encrypt(in), h.Decrypt(in)) {
		t.Error("encrypt and decrypt streams are identical; expected distinct")
	}
}
