package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

type SecureConn struct {
	net.Conn
	aead       cipher.AEAD
	readPrefix []byte
	writePref  []byte

	readCtr  uint64
	writeCtr uint64

	readBuf []byte // slice-based buffer for zero-copy reads
	writeMu sync.Mutex
}

func newSecureConn(conn net.Conn, key, readPrefix, writePrefix []byte) (*SecureConn, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &SecureConn{
		Conn:       conn,
		aead:       aead,
		readPrefix: readPrefix,
		writePref:  writePrefix,
	}, nil
}

func (s *SecureConn) buildNonce(prefix []byte, ctr uint64) []byte {
	n := make([]byte, nonceSize)
	copy(n, prefix)
	// XOR the counter into the last 8 bytes of the nonce like TLS 1.3
	var ctrBytes [8]byte
	binary.BigEndian.PutUint64(ctrBytes[:], ctr)
	for i := 0; i < 8; i++ {
		n[nonceSize-8+i] ^= ctrBytes[i]
	}
	return n
}

func (s *SecureConn) Write(p []byte) (int, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	total := 0
	for len(p) > 0 {
		chunk := p
		if len(chunk) > maxFrame {
			chunk = p[:maxFrame]
		}

		n := s.buildNonce(s.writePref, s.writeCtr)
		s.writeCtr++

		encrypted := s.aead.Seal(nil, n, chunk, nil)

		var hdr [4]byte
		binary.BigEndian.PutUint32(hdr[:], uint32(len(encrypted)))

		if _, err := s.Conn.Write(hdr[:]); err != nil {
			return total, err
		}
		if _, err := s.Conn.Write(encrypted); err != nil {
			return total, err
		}

		total += len(chunk)
		p = p[len(chunk):]
	}
	return total, nil
}

func (s *SecureConn) Read(p []byte) (int, error) {
	// serve from buffer first to avoid memory allocations
	if len(s.readBuf) > 0 {
		n := copy(p, s.readBuf)
		s.readBuf = s.readBuf[n:]
		return n, nil
	}

	var hdr [4]byte
	if _, err := io.ReadFull(s.Conn, hdr[:]); err != nil {
		return 0, err
	}

	size := binary.BigEndian.Uint32(hdr[:])
	if size == 0 || size > maxFrame+1024 {
		return 0, fmt.Errorf("invalid encrypted frame size: %d", size)
	}

	encrypted := make([]byte, size)
	if _, err := io.ReadFull(s.Conn, encrypted); err != nil {
		return 0, err
	}

	n := s.buildNonce(s.readPrefix, s.readCtr)
	s.readCtr++

	plain, err := s.aead.Open(nil, n, encrypted, nil)
	if err != nil {
		return 0, err
	}

	// copy to user buffer and save remainder
	copied := copy(p, plain)
	if copied < len(plain) {
		s.readBuf = plain[copied:]
	}
	return copied, nil
}
