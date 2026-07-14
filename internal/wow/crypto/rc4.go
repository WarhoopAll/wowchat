package crypto

type RC4 struct {
	sbox [256]int
	i, j int
}

func NewRC4(key []byte) *RC4 {
	r := &RC4{}
	for i := 0; i < 256; i++ {
		r.sbox[i] = i
	}
	j := 0
	for i := 0; i < 256; i++ {
		ki := int(int8(key[i%len(key)]))
		j = (j + r.sbox[i] + ki + 256) % 256
		r.sbox[i], r.sbox[j] = r.sbox[j], r.sbox[i]
	}
	return r
}

func (r *RC4) Crypt(msg []byte) []byte {
	out := make([]byte, len(msg))
	for n := 0; n < len(msg); n++ {
		r.i = (r.i + 1) % 256
		r.j = (r.j + r.sbox[r.i]) % 256
		r.sbox[r.i], r.sbox[r.j] = r.sbox[r.j], r.sbox[r.i]
		rand := r.sbox[(r.sbox[r.i]+r.sbox[r.j])%256]
		out[n] = byte(rand) ^ msg[n]
	}
	return out
}

func (r *RC4) Discard(n int) {
	zeros := make([]byte, n)
	r.Crypt(zeros)
}
