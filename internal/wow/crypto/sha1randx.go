package crypto

type SHA1Randx struct {
	o1, o2, o0 []byte
	index      int
}

func NewSHA1Randx(sessionKey []byte) *SHA1Randx {
	half := len(sessionKey) / 2
	r := &SHA1Randx{
		o1: SHA1(sessionKey[:half]),
		o2: SHA1(sessionKey[half:]),
		o0: make([]byte, 20),
	}
	r.fill()
	return r
}

func (r *SHA1Randx) fill() {
	r.o0 = SHA1(r.o1, r.o0, r.o2)
	r.index = 0
}

func (r *SHA1Randx) Generate(n int) []byte {
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		if r.index >= len(r.o0) {
			r.fill()
		}
		out[i] = r.o0[r.index]
		r.index++
	}
	return out
}
