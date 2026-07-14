package crypto

import (
	"bytes"
	"testing"
)

func TestRC4KnownVector(t *testing.T) {
	c := NewRC4([]byte("Key"))
	got := c.Crypt([]byte("Plaintext"))
	want := []byte{0xBB, 0xF3, 0x16, 0xE8, 0xD9, 0x40, 0xAF, 0x0A, 0xD3}
	if !bytes.Equal(got, want) {
		t.Errorf("RC4(\"Key\") = % x, want % x", got, want)
	}
}

func TestRC4WikiVector(t *testing.T) {
	c := NewRC4([]byte("Wiki"))
	got := c.Crypt([]byte("pedia"))
	want := []byte{0x10, 0x21, 0xBF, 0x04, 0x20}
	if !bytes.Equal(got, want) {
		t.Errorf("RC4(\"Wiki\") = % x, want % x", got, want)
	}
}

func TestRC4StatefulStream(t *testing.T) {
	full := NewRC4([]byte("Key"))
	a := full.Crypt([]byte("Plaintext"))

	split := NewRC4([]byte("Key"))
	p1 := split.Crypt([]byte("Plai"))
	p2 := split.Crypt([]byte("ntext"))
	combined := append(p1, p2...)

	if !bytes.Equal(combined, a) {
		t.Errorf("split stream != combined stream\n combined % x\n single   % x", combined, a)
	}
}

func TestRC4DiscardAdvances(t *testing.T) {
	c := NewRC4([]byte("Key"))
	c.Discard(1024)
	out := c.Crypt([]byte("Plaintext"))
	plain := NewRC4([]byte("Key")).Crypt([]byte("Plaintext"))
	if bytes.Equal(out, plain) {
		t.Error("Discard did not advance the keystream")
	}
}

func TestSHA1Known(t *testing.T) {
	got := SHA1([]byte("abc"))
	want := []byte{0xA9, 0x99, 0x3E, 0x36, 0x47, 0x06, 0x81, 0x6A, 0xBA, 0x3E,
		0x25, 0x71, 0x78, 0x50, 0xC2, 0x6C, 0x9C, 0xD0, 0xD8, 0x9D}
	if !bytes.Equal(got, want) {
		t.Errorf("SHA1(abc) = % x", got)
	}
}

func TestSHA1Concat(t *testing.T) {
	a := SHA1([]byte("foo"), []byte("bar"))
	b := SHA1([]byte("foobar"))
	if !bytes.Equal(a, b) {
		t.Error("SHA1 of parts != SHA1 of concat")
	}
}

func TestHMACSHA1RFC(t *testing.T) {
	key := bytes.Repeat([]byte{0x0b}, 20)
	got := HMACSHA1(key, []byte("Hi There"))
	want := []byte{0xb6, 0x17, 0x31, 0x86, 0x55, 0x05, 0x72, 0x64, 0xe2, 0x8b,
		0xc0, 0xb6, 0xfb, 0x37, 0x8c, 0x8e, 0xf1, 0x46, 0xbe, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("HMAC-SHA1 = % x", got)
	}
}
