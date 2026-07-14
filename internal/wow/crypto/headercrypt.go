package crypto

type HeaderCrypt struct {
	clientCrypt *RC4
	serverCrypt *RC4
	initialized bool
}

var (
	serverHMACSeed = []byte{
		0xCC, 0x98, 0xAE, 0x04, 0xE8, 0x97, 0xEA, 0xCA,
		0x12, 0xDD, 0xC0, 0x93, 0x42, 0x91, 0x53, 0x57,
	}
	clientHMACSeed = []byte{
		0xC2, 0xB3, 0x72, 0x3C, 0xC6, 0xAE, 0xD9, 0xB5,
		0x34, 0x3C, 0x53, 0xEE, 0x2F, 0x43, 0x67, 0xCE,
	}
)

func NewHeaderCrypt() *HeaderCrypt { return &HeaderCrypt{} }

func (h *HeaderCrypt) Init(sessionKey []byte) {
	serverKey := HMACSHA1(serverHMACSeed, sessionKey)
	clientKey := HMACSHA1(clientHMACSeed, sessionKey)

	h.serverCrypt = NewRC4(serverKey)
	h.serverCrypt.Discard(1024)
	h.clientCrypt = NewRC4(clientKey)
	h.clientCrypt.Discard(1024)
	h.initialized = true
}

func (h *HeaderCrypt) Initialized() bool { return h.initialized }

func (h *HeaderCrypt) Decrypt(data []byte) []byte {
	if !h.initialized {
		out := make([]byte, len(data))
		copy(out, data)
		return out
	}
	return h.serverCrypt.Crypt(data)
}

func (h *HeaderCrypt) Encrypt(data []byte) []byte {
	if !h.initialized {
		out := make([]byte, len(data))
		copy(out, data)
		return out
	}
	return h.clientCrypt.Crypt(data)
}
