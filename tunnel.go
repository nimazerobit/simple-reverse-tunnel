package main

import (
	"crypto/hmac"
	"fmt"
	"io"
	"net"
	"sync"
)

func setKeepAlive(conn net.Conn) {
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(defaultTTL)
	}
}

func pipeBoth(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	pipe := func(dst, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		defer src.Close()

		bufPtr := bufPool.Get().(*[]byte)
		defer bufPool.Put(bufPtr)

		_, _ = io.CopyBuffer(dst, src, *bufPtr)
	}

	go pipe(a, b)
	go pipe(b, a)
	wg.Wait()
}

func clientHandshake(conn net.Conn, secret string) (*SecureConn, error) {
	setKeepAlive(conn)

	salt, err := generateRandomBytes(saltSize)
	if err != nil {
		return nil, err
	}

	clientPrefix, err := generateRandomBytes(nonceSize)
	if err != nil {
		return nil, err
	}
	serverPrefix, err := generateRandomBytes(nonceSize)
	if err != nil {
		return nil, err
	}

	helloData := append([]byte(magic), salt...)
	helloData = append(helloData, clientPrefix...)
	helloData = append(helloData, serverPrefix...)
	auth := hmacSHA(secret, helloData)

	if _, err := conn.Write(append(helloData, auth...)); err != nil {
		return nil, err
	}

	expectedServerAuth := hmacSHA(secret, append([]byte("server-ok"), salt...))
	serverAuth := make([]byte, 32)
	if _, err := io.ReadFull(conn, serverAuth); err != nil {
		return nil, err
	}

	if !hmac.Equal(serverAuth, expectedServerAuth) {
		return nil, fmt.Errorf("server authentication failed")
	}

	key := hkdf([]byte(secret), salt, []byte("tunnel-key"), 32)
	return newSecureConn(conn, key, serverPrefix, clientPrefix)
}

func serverHandshake(conn net.Conn, secret string) (*SecureConn, error) {
	setKeepAlive(conn)

	headerLen := len(magic) + saltSize + (nonceSize * 2) + 32
	packet := make([]byte, headerLen)

	if _, err := io.ReadFull(conn, packet); err != nil {
		return nil, err
	}

	if string(packet[:len(magic)]) != magic {
		return nil, fmt.Errorf("bad magic")
	}

	idx := len(magic)
	salt := packet[idx : idx+saltSize]
	idx += saltSize
	clientPrefix := packet[idx : idx+nonceSize]
	idx += nonceSize
	serverPrefix := packet[idx : idx+nonceSize]
	idx += nonceSize
	clientAuth := packet[idx:]

	helloData := packet[:idx]
	if !hmac.Equal(clientAuth, hmacSHA(secret, helloData)) {
		return nil, fmt.Errorf("client authentication failed")
	}

	serverAuth := hmacSHA(secret, append([]byte("server-ok"), salt...))
	if _, err := conn.Write(serverAuth); err != nil {
		return nil, err
	}

	key := hkdf([]byte(secret), salt, []byte("tunnel-key"), 32)
	return newSecureConn(conn, key, clientPrefix, serverPrefix)
}
