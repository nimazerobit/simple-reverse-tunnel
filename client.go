package main

import (
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

func runClient(serverAddr, localAddr, secret string) error {
	backoff := time.Second
	for {
		err := clientMultiplex(serverAddr, localAddr, secret)
		if err != nil {
			log.Printf("[client] Tunnel disconnected or error: %v", err)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
		} else {
			backoff = time.Second
		}
	}
}

func clientMultiplex(serverAddr, localAddr, secret string) error {
	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	setKeepAlive(conn)

	sConn, err := clientHandshake(conn, secret)
	if err != nil {
		return err
	}

	log.Printf("[client] Connected to server %s", serverAddr)
	defer log.Printf("[client] Disconnected from server %s", serverAddr)

	// setup yamux server
	session, err := yamux.Server(sConn, yamux.DefaultConfig())
	if err != nil {
		return err
	}
	defer session.Close()

	for {
		stream, err := session.Accept()
		if err != nil {
			return err // break loop and trigger reconnect
		}

		go func(st net.Conn) {
			defer st.Close()
			localConn, err := net.Dial("tcp", localAddr)
			if err != nil {
				log.Printf("[client] Failed to connect to local target: %v", err)
				return
			}
			defer localConn.Close()

			pipeBoth(localConn, st)
		}(stream)
	}
}
