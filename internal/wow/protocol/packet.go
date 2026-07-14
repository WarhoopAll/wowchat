package protocol

import (
	"errors"
	"math"
)

type PacketReader struct {
	buf []byte
	pos int
}

func NewPacketReader(b []byte) *PacketReader { return &PacketReader{buf: b} }

func (r *PacketReader) Remaining() int { return len(r.buf) - r.pos }

func (r *PacketReader) Bytes() []byte { return r.buf }

func (r *PacketReader) ReadByte() (byte, error) {
	if r.pos >= len(r.buf) {
		return 0, ErrShortBuffer
	}
	b := r.buf[r.pos]
	r.pos++
	return b, nil
}

func (r *PacketReader) MustByte() byte {
	b, err := r.ReadByte()
	if err != nil {
		panic(err)
	}
	return b
}

func (r *PacketReader) ReadBytes(n int) ([]byte, error) {
	if n < 0 || r.pos+n > len(r.buf) {
		return nil, ErrShortBuffer
	}
	out := make([]byte, n)
	copy(out, r.buf[r.pos:r.pos+n])
	r.pos += n
	return out, nil
}

func (r *PacketReader) MustBytes(n int) []byte {
	b, err := r.ReadBytes(n)
	if err != nil {
		panic(err)
	}
	return b
}

func (r *PacketReader) Skip(n int) error {
	if n < 0 || r.pos+n > len(r.buf) {
		return ErrShortBuffer
	}
	r.pos += n
	return nil
}

func (r *PacketReader) ReadShort() (int16, error) {
	b, err := r.ReadBytes(2)
	if err != nil {
		return 0, err
	}
	return int16(uint16(b[0])<<8 | uint16(b[1])), nil
}

func (r *PacketReader) ReadShortLE() (int16, error) {
	b, err := r.ReadBytes(2)
	if err != nil {
		return 0, err
	}
	return int16(uint16(b[1])<<8 | uint16(b[0])), nil
}

func (r *PacketReader) MustShortLE() int16 {
	v, err := r.ReadShortLE()
	if err != nil {
		panic(err)
	}
	return v
}

func (r *PacketReader) ReadUShortLE() (uint16, error) {
	v, err := r.ReadShortLE()
	return uint16(v), err
}

func (r *PacketReader) ReadInt() (int32, error) {
	b, err := r.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return int32(uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])), nil
}

func (r *PacketReader) MustInt() int32 {
	v, err := r.ReadInt()
	if err != nil {
		panic(err)
	}
	return v
}

func (r *PacketReader) ReadIntLE() (int32, error) {
	b, err := r.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return int32(uint32(b[3])<<24 | uint32(b[2])<<16 | uint32(b[1])<<8 | uint32(b[0])), nil
}

func (r *PacketReader) MustIntLE() int32 {
	v, err := r.ReadIntLE()
	if err != nil {
		panic(err)
	}
	return v
}

func (r *PacketReader) ReadLongLE() (int64, error) {
	b, err := r.ReadBytes(8)
	if err != nil {
		return 0, err
	}
	var u uint64
	for i := 0; i < 8; i++ {
		u |= uint64(b[i]) << (uint(i) * 8)
	}
	return int64(u), nil
}

func (r *PacketReader) MustLongLE() int64 {
	v, err := r.ReadLongLE()
	if err != nil {
		panic(err)
	}
	return v
}

func (r *PacketReader) ReadFloatLE() (float32, error) {
	u, err := r.ReadIntLE()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(uint32(u)), nil
}

func (r *PacketReader) ReadString() (string, error) {
	var end = len(r.buf)
	for i := r.pos; i < len(r.buf); i++ {
		if r.buf[i] == 0 {
			end = i
			break
		}
	}
	s := string(r.buf[r.pos:end])
	if end < len(r.buf) {
		r.pos = end + 1
	} else {
		r.pos = end
	}
	return s, nil
}

func (r *PacketReader) MustString() string {
	s, _ := r.ReadString()
	return s
}

func (r *PacketReader) SkipString() {
	for r.pos < len(r.buf) {
		b := r.buf[r.pos]
		r.pos++
		if b == 0 {
			return
		}
	}
}

func (r *PacketReader) ReadFixedString(n int) (string, error) {
	b, err := r.ReadBytes(n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type PacketWriter struct {
	buf []byte
}

func NewPacketWriter() *PacketWriter { return &PacketWriter{buf: make([]byte, 0, 64)} }

func (w *PacketWriter) Bytes() []byte { return w.buf }

func (w *PacketWriter) Len() int { return len(w.buf) }

func (w *PacketWriter) Reset() { w.buf = w.buf[:0] }

func (w *PacketWriter) PutByte(b byte) { w.buf = append(w.buf, b) }

func (w *PacketWriter) WriteBytes(b []byte) { w.buf = append(w.buf, b...) }

func (w *PacketWriter) WriteShortBE(v int) {
	w.buf = append(w.buf, byte(v>>8), byte(v))
}

func (w *PacketWriter) WriteShortLE(v int) {
	w.buf = append(w.buf, byte(v), byte(v>>8))
}

func (w *PacketWriter) WriteIntBE(v int32) {
	u := uint32(v)
	w.buf = append(w.buf, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
}

func (w *PacketWriter) WriteIntLE(v int32) {
	u := uint32(v)
	w.buf = append(w.buf, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
}

func (w *PacketWriter) WriteLongLE(v int64) {
	u := uint64(v)
	w.buf = append(w.buf,
		byte(u), byte(u>>8), byte(u>>16), byte(u>>24),
		byte(u>>32), byte(u>>40), byte(u>>48), byte(u>>56))
}

func (w *PacketWriter) WriteFloatLE(f float32) {
	w.WriteIntLE(int32(math.Float32bits(f)))
}

func (w *PacketWriter) WriteCString(s string) {
	w.buf = append(w.buf, s...)
	w.buf = append(w.buf, 0)
}

func (w *PacketWriter) WriteUTF8(s string) { w.buf = append(w.buf, s...) }

var _ = errors.New
