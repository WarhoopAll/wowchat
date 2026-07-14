package protocol

import (
	"errors"
)

var ErrShortBuffer = errors.New("protocol: short buffer")

func IntBE(v int32) []byte {
	u := uint32(v)
	return []byte{byte(u >> 24), byte(u >> 16), byte(u >> 8), byte(u)}
}

func IntLE(v int32) []byte {
	u := uint32(v)
	return []byte{byte(u), byte(u >> 8), byte(u >> 16), byte(u >> 24)}
}

func stringToInt(s string) int32 {
	var u uint64
	for i := 0; i < len(s); i++ {
		u |= uint64(s[i]) << (uint(len(s)-1-i) * 8)
	}
	return int32(u)
}

func StringToInt(s string) int { return int(stringToInt(s)) }

func HexDump(b []byte) string {
	const hex = "0123456789ABCDEF"
	if len(b) == 0 {
		return ""
	}
	out := make([]byte, 0, len(b)*3)
	for _, x := range b {
		out = append(out, hex[x>>4], hex[x&0x0F], ' ')
	}
	return string(out[:len(out)-1])
}
