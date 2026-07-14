package crypto

import "strings"

type SRPClient struct {
	k *BigNumber
	a *BigNumber
	A *BigNumber
	M *BigNumber
	S *BigNumber
	K *BigNumber
}

func NewSRPClient(privateA *BigNumber) (*SRPClient, error) {
	if privateA == nil {
		var err error
		privateA, err = RandBig(19)
		if err != nil {
			return nil, err
		}
	}
	return &SRPClient{
		k: NewBigFromInt(3),
		a: privateA,
	}, nil
}

func (s *SRPClient) Step1(accountUpper []byte, password string, B, g, N, salt *BigNumber) {
	passwordUpper := strings.ToUpper(password)

	s.A = g.ModPow(s.a, N)

	u := NewBigFromBytes(SHA1(s.A.AsByteArray(32, true), B.AsByteArray(32, true)), true)

	user := make([]byte, 0, len(accountUpper)+1+len(passwordUpper))
	user = append(user, accountUpper...)
	user = append(user, ':')
	user = append(user, passwordUpper...)
	p := SHA1(user)

	x := NewBigFromBytes(SHA1(salt.AsByteArray(32, true), p), true)

	gx := g.ModPow(x, N)
	kgx := s.k.Mul(gx)
	base := B.Sub(kgx)
	exp := s.a.Add(u.Mul(x))
	s.S = base.ModPow(exp, N)

	t := s.S.AsByteArray(32, true)
	t1 := make([]byte, 16)
	t2 := make([]byte, 16)
	for i := 0; i < 16; i++ {
		t1[i] = t[i*2]
		t2[i] = t[i*2+1]
	}
	vK := make([]byte, 40)
	d1 := SHA1(t1)
	for i := 0; i < 20; i++ {
		vK[i*2] = d1[i]
	}
	d2 := SHA1(t2)
	for i := 0; i < 20; i++ {
		vK[i*2+1] = d2[i]
	}
	s.K = NewBigFromBytes(vK, true)

	hash := SHA1(N.AsByteArray(32, true))
	digest := SHA1(g.AsByteArray(1, true))
	for i := 0; i < 20; i++ {
		hash[i] ^= digest[i]
	}

	t4 := SHA1(accountUpper)
	s.M = NewBigFromBytes(SHA1(
		hash,
		t4,
		salt.AsByteArray(32, true),
		s.A.AsByteArray(32, true),
		B.AsByteArray(32, true),
		s.K.AsByteArray(40, true),
	), false)
}

func (s *SRPClient) GenerateHashLogonProof() []byte {
	return SHA1(
		s.A.AsByteArray(32, true),
		s.M.AsByteArray(20, false),
		s.K.AsByteArray(40, true),
	)
}

func (s *SRPClient) SessionKey() []byte {
	return s.K.AsByteArray(40, true)
}

func (s *SRPClient) ClientProof() []byte {
	return s.M.AsByteArray(20, false)
}

func (s *SRPClient) ClientPublic() []byte {
	return s.A.AsByteArray(32, true)
}
