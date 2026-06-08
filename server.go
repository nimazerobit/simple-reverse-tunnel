package main

import (
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

func runServer(publicAddr, tunnelAddr, secret string) error {
	log.Printf("[server] public listener: %s", publicAddr)
	log.Printf("[server] tunnel listener: %s", tunnelAddr)

	tunnelListener, err := net.Listen("tcp", tunnelAddr)
	if err != nil {
		return err
	}
	defer tunnelListener.Close()

	publicListener, err := net.Listen("tcp", publicAddr)
	if err != nil {
		return err
	}
	defer publicListener.Close()

	// channel to hold the active yamux session
	sessionCh := make(chan *yamux.Session, 1)

	// accept tunnel connections
	go func() {
		for {
			conn, err := tunnelListener.Accept()
			if err != nil {
				continue
			}

			setKeepAlive(conn)
			sConn, err := serverHandshake(conn, secret)
			if err != nil {
				log.Printf("[server] Tunnel handshake failed: %v", err)
				conn.Close()
				continue
			}
			log.Printf("[server] Tunnel connected from %s", conn.RemoteAddr())

			// setup yamux client
			session, err := yamux.Client(sConn, yamux.DefaultConfig())
			if err != nil {
				log.Printf("[server] Failed to setup yamux client: %v", err)
				sConn.Close()
				continue
			}

			// non-blocking replace of the active session
			select {
			case <-sessionCh:
			default:
			}
			sessionCh <- session
		}
	}()

	var currentSession *yamux.Session

	// accept public connections
	for {
		userConn, err := publicListener.Accept()
		if err != nil {
			continue
		}

		go func(uc net.Conn) {
			defer uc.Close()
			remote := uc.RemoteAddr()
			log.Printf("[server] Client connected from %s", remote)
			defer log.Printf("[server] Client disconnected from %s", remote)

			// wait for a valid open tunnel session
			for currentSession == nil || currentSession.IsClosed() {
				currentSession = <-sessionCh
			}

			// open a multiplexed stream through the tunnel
			stream, err := currentSession.Open()
			if err != nil {
				log.Printf("[server] Failed to open tunnel stream: %v", err)
				return
			}
			defer stream.Close()

			pipeBoth(uc, stream)
		}(userConn)
	}
}
