package crypto

import (
	"bytes"
	"testing"
)

var (
	srpN, _ = NewBigFromHex("894B645E89E1535BBDAD5B8B290650530801B18EBFBF5E8F2436616F0CC8EFAA")
	srpG    = NewBigFromInt(7)
)

func TestSRPConstantsLength(t *testing.T) {
	if got := srpN.AsByteArray(32, false); len(got) != 32 {
		t.Errorf("N length = %d, want 32", len(got))
	}
}

func TestSRPSelfConsistent(t *testing.T) {
	accountUpper := []byte("TESTUSER")
	password := "p4ssw0rd"
	saltBytes := bytes.Repeat([]byte{0xAB}, 32)
	salt := NewBigFromBytes(saltBytes, true)

	user := append(append([]byte("TESTUSER"), ':'), []byte("P4SSW0RD")...)
	inner := SHA1(user)
	x := NewBigFromBytes(SHA1(salt.AsByteArray(32, true), inner), true)
	verifier := srpG.ModPow(x, srpN)

	k := NewBigFromInt(3)
	b := NewBigFromInt(2)
	Bsrv := k.Mul(verifier).Add(srpG.ModPow(b, srpN))

	privA, _ := NewBigFromHex("01FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	client, err := NewSRPClient(privA)
	if err != nil {
		t.Fatalf("NewSRPClient: %v", err)
	}
	client.Step1(accountUpper, password, Bsrv, srpG, srpN, salt)

	u := NewBigFromBytes(SHA1(client.A.AsByteArray(32, true), Bsrv.AsByteArray(32, true)), true)
	Sserver := client.A.Mul(verifier.ModPow(u, srpN)).ModPow(b, srpN)

	kt := Sserver.AsByteArray(32, true)
	kt1 := make([]byte, 16)
	kt2 := make([]byte, 16)
	for i := 0; i < 16; i++ {
		kt1[i] = kt[i*2]
		kt2[i] = kt[i*2+1]
	}
	kvBytes := make([]byte, 40)
	kd1 := SHA1(kt1)
	for i := 0; i < 20; i++ {
		kvBytes[i*2] = kd1[i]
	}
	kd2 := SHA1(kt2)
	for i := 0; i < 20; i++ {
		kvBytes[i*2+1] = kd2[i]
	}
	Kserver := NewBigFromBytes(kvBytes, true)

	hash := SHA1(srpN.AsByteArray(32, true))
	dig := SHA1(srpG.AsByteArray(1, true))
	for i := range hash {
		hash[i] ^= dig[i]
	}
	Mserver := SHA1(hash, SHA1(accountUpper),
		salt.AsByteArray(32, true),
		client.A.AsByteArray(32, true),
		Bsrv.AsByteArray(32, true),
		Kserver.AsByteArray(40, true))

	if !bytes.Equal(client.ClientProof(), Mserver) {
		t.Fatalf("client M != server M\n client % x\n server % x", client.ClientProof(), Mserver)
	}

	M1server := SHA1(client.A.AsByteArray(32, true), Mserver, Kserver.AsByteArray(40, true))
	if !bytes.Equal(client.GenerateHashLogonProof(), M1server) {
		t.Fatalf("logon proof mismatch\n client % x\n server % x",
			client.GenerateHashLogonProof(), M1server)
	}

	if !bytes.Equal(client.SessionKey(), Kserver.AsByteArray(40, true)) {
		t.Fatalf("session key mismatch\n client % x\n server % x",
			client.SessionKey(), Kserver.AsByteArray(40, true))
	}

	if len(client.ClientPublic()) != 32 {
		t.Errorf("A length = %d, want 32", len(client.ClientPublic()))
	}
}

func TestSRPDeterministic(t *testing.T) {
	accountUpper := []byte("U")
	password := "p"
	salt := NewBigFromBytes(bytes.Repeat([]byte{0x11}, 32), true)
	privA, _ := NewBigFromHex("AA")
	B := NewBigFromInt(123456)

	run := func() []byte {
		c, _ := NewSRPClient(privA)
		c.Step1(accountUpper, password, B, srpG, srpN, salt)
		out := append([]byte{}, c.ClientProof()...)
		return append(out, c.SessionKey()...)
	}
	first := run()
	second := run()
	if !bytes.Equal(first, second) {
		t.Fatal("SRP is non-deterministic for fixed inputs")
	}
}
