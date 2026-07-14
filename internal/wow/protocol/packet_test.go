package protocol

import (
	"bytes"
	"testing"
)

func TestStringToIntWoW(t *testing.T) {
	if got := StringToInt("WoW"); got != 0x576F57 {
		t.Errorf("StringToInt(WoW) = %#x, want 0x576F57", got)
	}
	if got := StringToInt("x86"); got != 0x783836 {
		t.Errorf("StringToInt(x86) = %#x, want 0x783836", got)
	}
	if got := StringToInt("OSX"); got != 0x4F5358 {
		t.Errorf("StringToInt(OSX) = %#x, want 0x4F5358", got)
	}
	if got := StringToInt("Win"); got != 0x57696E {
		t.Errorf("StringToInt(Win) = %#x, want 0x57696E", got)
	}
	if got := StringToInt("enUS"); got != 0x656E5553 {
		t.Errorf("StringToInt(enUS) = %#x, want 0x656E5553", got)
	}
}

func TestIntBEAndLE(t *testing.T) {
	v := int32(0x12345678)
	if got := IntBE(v); !bytes.Equal(got, []byte{0x12, 0x34, 0x56, 0x78}) {
		t.Errorf("IntBE = %x", got)
	}
	if got := IntLE(v); !bytes.Equal(got, []byte{0x78, 0x56, 0x34, 0x12}) {
		t.Errorf("IntLE = %x", got)
	}
}

func TestReaderShortLEIntLELongLE(t *testing.T) {
	w := NewPacketWriter()
	w.WriteShortLE(0x0102)
	w.WriteIntLE(0x03040506)
	w.WriteLongLE(0x0708090A0B0C0D0E)

	r := NewPacketReader(w.Bytes())
	if v, _ := r.ReadShortLE(); v != 0x0102 {
		t.Errorf("shortLE = %#x", v)
	}
	if v, _ := r.ReadIntLE(); v != 0x03040506 {
		t.Errorf("intLE = %#x", v)
	}
	if v, _ := r.ReadLongLE(); v != 0x0708090A0B0C0D0E {
		t.Errorf("longLE = %#x", v)
	}
}

func TestReaderShortAndIntBE(t *testing.T) {
	w := NewPacketWriter()
	w.WriteShortBE(0x0102)
	w.WriteIntBE(0x03040506)
	r := NewPacketReader(w.Bytes())
	if v, _ := r.ReadShort(); v != 0x0102 {
		t.Errorf("shortBE = %#x", v)
	}
	if v, _ := r.ReadInt(); v != 0x03040506 {
		t.Errorf("intBE = %#x", v)
	}
}

func TestReadStringNULTerminated(t *testing.T) {
	r := NewPacketReader([]byte("abc\x00extra"))
	s, _ := r.ReadString()
	if s != "abc" {
		t.Errorf("got %q want abc", s)
	}
	if r.Remaining() != 5 {
		t.Errorf("remaining = %d want 5", r.Remaining())
	}
}

func TestReadStringNoNUL(t *testing.T) {
	r := NewPacketReader([]byte("abc"))
	s, _ := r.ReadString()
	if s != "abc" {
		t.Errorf("got %q want abc", s)
	}
	if r.Remaining() != 0 {
		t.Errorf("remaining = %d want 0", r.Remaining())
	}
}

func TestReadCStringUTF8Cyrillic(t *testing.T) {
	src := []byte("Поиск\x00")
	r := NewPacketReader(src)
	s, _ := r.ReadString()
	if s != "Поиск" {
		t.Errorf("got %q want Поиск", s)
	}
}

func TestSkipString(t *testing.T) {
	r := NewPacketReader([]byte("hello\x00world\x00"))
	r.SkipString()
	if s, _ := r.ReadString(); s != "world" {
		t.Errorf("after skip got %q want world", s)
	}
}

func TestSkipAndShortBuffer(t *testing.T) {
	r := NewPacketReader([]byte{0x01, 0x02})
	if err := r.Skip(3); err != ErrShortBuffer {
		t.Errorf("Skip past end: err = %v, want ErrShortBuffer", err)
	}
}

func TestHexDump(t *testing.T) {
	got := HexDump([]byte{0x01, 0x02, 0xAB})
	want := "01 02 AB"
	if got != want {
		t.Errorf("HexDump = %q want %q", got, want)
	}
}

func TestWriteCString(t *testing.T) {
	w := NewPacketWriter()
	w.WriteCString("hi")
	if got := w.Bytes(); !bytes.Equal(got, []byte{'h', 'i', 0}) {
		t.Errorf("WriteCString = %x", got)
	}
}

func TestAuthIsSuccess(t *testing.T) {
	if !AuthIsSuccess(0x00) || !AuthIsSuccess(0x0E) {
		t.Error("0x00 and 0x0E should be success")
	}
	if AuthIsSuccess(0x05) {
		t.Error("0x05 should not be success")
	}
}
