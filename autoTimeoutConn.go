package netutils

import (
	"net"
	"time"
)

type autoTimeoutConn struct {
	net.Conn
	timeout time.Duration
}

func (c *autoTimeoutConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	c.Conn.SetReadDeadline(time.Now().Add(c.timeout))
	return
}

func (c *autoTimeoutConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
	return
}

func NewAutoTimeoutConn(conn net.Conn, timeout time.Duration) net.Conn {
	return &autoTimeoutConn{
		conn,
		timeout,
	}
}
