package crypto

import (
	"crypto/rand"
	"math/big"
)

type BigNumber struct {
	v *big.Int
}

func NewBigFromInt(n int) *BigNumber { return &BigNumber{v: big.NewInt(int64(n))} }

func NewBigFromHex(s string) (*BigNumber, bool) {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, false
	}
	return &BigNumber{v: n}, true
}

func RandBig(amount int) (*BigNumber, error) {
	b := make([]byte, amount)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(b)
	return &BigNumber{v: n}, nil
}

func NewBigFromBytes(array []byte, reverse bool) *BigNumber {
	b := make([]byte, len(array))
	copy(b, array)
	if reverse {
		for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
			b[i], b[j] = b[j], b[i]
		}
	}
	if len(b) > 0 && b[0]&0x80 != 0 {
		guarded := make([]byte, len(b)+1)
		copy(guarded[1:], b)
		b = guarded
	}
	return &BigNumber{v: new(big.Int).SetBytes(b)}
}

func (n *BigNumber) Mul(b *BigNumber) *BigNumber {
	rhs := new(big.Int).Abs(b.v)
	return &BigNumber{v: new(big.Int).Mul(n.v, rhs)}
}

func (n *BigNumber) Sub(b *BigNumber) *BigNumber {
	rhs := new(big.Int).Abs(b.v)
	return &BigNumber{v: new(big.Int).Sub(n.v, rhs)}
}

func (n *BigNumber) Add(b *BigNumber) *BigNumber {
	rhs := new(big.Int).Abs(b.v)
	return &BigNumber{v: new(big.Int).Add(n.v, rhs)}
}

func (n *BigNumber) ModPow(exp, m *BigNumber) *BigNumber {
	base := new(big.Int).Abs(n.v)
	e := new(big.Int).Abs(exp.v)
	mm := new(big.Int).Abs(m.v)
	return &BigNumber{v: new(big.Int).Exp(base, e, mm)}
}

func (n *BigNumber) AsByteArray(reqSize int, reverse bool) []byte {
	be := n.v.Bytes()
	b := make([]byte, len(be))
	copy(b, be)

	if reverse {
		for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
			b[i], b[j] = b[j], b[i]
		}
	}

	if reqSize > len(b) {
		out := make([]byte, reqSize)
		copy(out, b)
		return out
	}
	return b
}

func (n *BigNumber) HexString() string {
	return n.v.Text(16)
}

func (n *BigNumber) Int() int64 { return n.v.Int64() }

func (n *BigNumber) Cmp(b *BigNumber) int { return n.v.Cmp(b.v) }
