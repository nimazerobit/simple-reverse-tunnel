package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
)

func hkdf(secret, salt, info []byte, length int) []byte {
	m := hmac.New(sha256.New, salt)
	m.Write(secret)
	prk := m.Sum(nil)

	var okm []byte
	var t []byte
	for i := byte(1); len(okm) < length; i++ {
		m := hmac.New(sha256.New, prk)
		m.Write(t)
		m.Write(info)
		m.Write([]byte{i})
		t = m.Sum(nil)
		okm = append(okm, t...)
	}
	return okm[:length]
}

func hmacSHA(secret string, data []byte) []byte {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(data)
	return m.Sum(nil)
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}
