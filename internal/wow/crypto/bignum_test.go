package crypto

import (
	"bytes"
	"testing"
)

func TestAsByteArrayRightPad(t *testing.T) {
	n := NewBigFromInt(1)
	got := n.AsByteArray(4, true)
	want := []byte{0x01, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("AsByteArray(4,true) = % x, want % x", got, want)
	}
	got = n.AsByteArray(4, false)
	want = []byte{0x01, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("AsByteArray(4,false) = % x, want % x", got, want)
	}
}

func TestAsByteArrayRoundTripReverse(t *testing.T) {
	le := make([]byte, 32)
	le[0] = 0xAB
	le[31] = 0x01
	n := NewBigFromBytes(le, true)
	back := n.AsByteArray(32, true)
	if !bytes.Equal(back, le) {
		t.Errorf("round-trip LE failed\n got % x\nwant % x", back, le)
	}
}

func TestNewBigFromBytesSignGuard(t *testing.T) {
	le := make([]byte, 32)
	le[31] = 0xFF
	n := NewBigFromBytes(le, true)
	if n.v.Sign() < 0 {
		t.Fatal("value should be positive")
	}
	back := n.AsByteArray(32, true)
	if !bytes.Equal(back, le) {
		t.Errorf("sign-guard round-trip failed\n got % x\nwant % x", back, le)
	}
}

func TestSubRightOperandAbs(t *testing.T) {
	a := NewBigFromInt(10)
	b := NewBigFromInt(3)
	if r := a.Sub(b); r.Int() != 7 {
		t.Errorf("Sub = %d want 7", r.Int())
	}

	small := NewBigFromInt(3)
	big := NewBigFromInt(10)
	r := small.Sub(big)
	if r.v.Sign() >= 0 {
		t.Errorf("Sub left operand must keep sign, got non-negative %s", r.HexString())
	}
}

func TestHexString(t *testing.T) {
	n := NewBigFromInt(255)
	if n.HexString() != "ff" {
		t.Errorf("HexString = %q want ff", n.HexString())
	}
}
