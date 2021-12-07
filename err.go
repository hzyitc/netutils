package netutils

import (
	"errors"
	"net"
)

var (
	ErrClosed    = net.ErrClosed
	ErrTimeout   = errors.New("timeout")
	ErrConnected = errors.New("already had a connection")
)
