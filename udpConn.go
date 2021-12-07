package netutils

import (
	"net"
	"time"
)

type UDPConn struct {
	udp *UDP

	closedChan chan interface{}
	readBuf    chan []byte

	readDeadline  time.Time
	writeDeadline time.Time

	local  net.Addr
	remote net.Addr
}

func (c *UDPConn) Read(b []byte) (n int, err error) {
	select {
	case <-c.closedChan:
		return 0, ErrClosed
	case buf := <-c.readBuf:
		return copy(b, buf), nil
	case <-timeAfter(c.readDeadline):
		return 0, ErrTimeout
	}
}

func (c *UDPConn) Write(b []byte) (n int, err error) {
	select {
	case <-c.closedChan:
		return 0, ErrClosed
	default:
	}

	if timeIsPast(c.writeDeadline) {
		return 0, ErrTimeout
	}
	return c.udp.conn.WriteTo(b, c.remote)
}

func (c *UDPConn) Close() error {
	if !chanClose(c.closedChan) {
		return ErrClosed
	}

	c.udp.handleClosedConnection(c)
	return nil
}

func (c *UDPConn) LocalAddr() net.Addr {
	return c.local
}

func (c *UDPConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *UDPConn) SetDeadline(t time.Time) error {
	err1 := c.SetReadDeadline(t)
	err2 := c.SetWriteDeadline(t)

	if err1 != nil {
		return err1
	} else {
		return err2
	}
}

func (c *UDPConn) SetReadDeadline(t time.Time) error {
	select {
	case <-c.closedChan:
		return ErrClosed
	default:
	}

	c.readDeadline = t

	return nil
}

func (c *UDPConn) SetWriteDeadline(t time.Time) error {
	select {
	case <-c.closedChan:
		return ErrClosed
	default:
	}

	c.writeDeadline = t

	return nil
}
