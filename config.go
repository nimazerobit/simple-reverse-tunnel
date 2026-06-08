package main

import (
	"sync"
	"time"
)

const (
	magic      = "GTUN2"
	maxFrame   = 64 * 1024
	defaultTTL = 30 * time.Second
	nonceSize  = 12
	saltSize   = 16
)

var bufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024)
		return &buf
	},
}
