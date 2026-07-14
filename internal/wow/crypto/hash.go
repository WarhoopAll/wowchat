package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
)

func SHA1(parts ...[]byte) []byte {
	h := sha1.New()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

func HMACSHA1(key []byte, parts ...[]byte) []byte {
	h := hmac.New(sha1.New, key)
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

func MD5(parts ...[]byte) []byte {
	h := md5.New()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}
