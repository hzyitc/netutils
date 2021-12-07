package netutils

import (
	"net"
	"sync"
	"time"
)

type udpBuf struct {
	addr net.Addr
	buf  []byte
}

type UDP struct {
	conn net.PacketConn

	closedChan chan interface{}
	readBuf    chan udpBuf

	readDeadline  time.Time
	writeDeadline time.Time

	lock           sync.Mutex
	connectedConns map[string]*UDPConn
}

const (
	packetSize = 65535
	bufferLen  = 9000
)

func (u *UDP) IsClosed() bool {
	select {
	case <-u.closedChan:
		return true
	default:
		return false
	}
}

func (u *UDP) newConnection(addr net.Addr) (*UDPConn, error) {
	u.lock.Lock()
	defer u.lock.Unlock()

	if u.IsClosed() {
		return nil, ErrClosed
	}

	_, found := u.connectedConns[addr.String()]
	if found {
		return nil, ErrConnected
	}

	conn := &UDPConn{
		udp: u,

		closedChan: make(chan interface{}),
		readBuf:    make(chan []byte, bufferLen),

		readDeadline:  time.Time{},
		writeDeadline: time.Time{},

		local:  u.LocalAddr(),
		remote: addr,
	}
	u.connectedConns[addr.String()] = conn

	return conn, nil
}

func (u *UDP) handleClosedConnection(c *UDPConn) {
	u.lock.Lock()
	delete(u.connectedConns, c.RemoteAddr().String())
	u.lock.Unlock()
}

func (u *UDP) DialUDP(addr *net.UDPAddr) (net.Conn, error) {
	return u.newConnection(addr)
}

func (u *UDP) Dial(addr string) (net.Conn, error) {
	na, err := net.ResolveUDPAddr(u.LocalAddr().Network(), addr)
	if err != nil {
		return nil, err
	}

	return u.DialUDP(na)
}

func (u *UDP) DialResolved(network string, address string, service string, defaultPort int) (net.Conn, error) {
	na, err := ResolveAddr(network, address, service, defaultPort)
	if err != nil {
		return nil, err
	}

	return u.DialUDP(na.ToUDPAddr())
}

func (u *UDP) readFrom() (p []byte, addr net.Addr, err error) {
	select {
	case <-u.closedChan:
		return nil, nil, ErrClosed
	case buf := <-u.readBuf:
		return buf.buf, buf.addr, nil
	case <-timeAfter(u.readDeadline):
		return nil, nil, ErrTimeout
	}
}

func (u *UDP) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	buf, addr, err := u.readFrom()
	if err != nil {
		n = 0
		return
	}

	n = copy(p, buf)
	return
}

func (u *UDP) Accept() (net.Conn, error) {
	buf, addr, err := u.readFrom()
	if err != nil {
		return nil, err
	}

	conn, err := u.newConnection(addr)
	if err != nil {
		return nil, err
	}

	conn.readBuf <- buf

	return conn, nil
}

func (u *UDP) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	select {
	case <-u.closedChan:
		return 0, ErrClosed
	default:
	}

	if !u.writeDeadline.IsZero() && time.Now().After(u.writeDeadline) {
		return 0, ErrTimeout
	}
	return u.WriteTo(p, addr)
}

func (u *UDP) Close() error {
	if !chanClose(u.closedChan) {
		return ErrClosed
	}

	u.conn.Close()

	u.lock.Lock()
	cs := make([]*UDPConn, 0, len(u.connectedConns))
	for _, conn := range u.connectedConns {
		cs = append(cs, conn)
	}
	u.lock.Unlock()

	for _, conn := range cs {
		conn.Close()
	}

	return nil
}

func (u *UDP) LocalAddr() net.Addr {
	return u.conn.LocalAddr()
}

func (u *UDP) Addr() net.Addr {
	return u.conn.LocalAddr()
}

func (u *UDP) SetDeadline(t time.Time) error {
	err1 := u.SetReadDeadline(t)
	err2 := u.SetWriteDeadline(t)

	if err1 != nil {
		return err1
	} else {
		return err2
	}
}

func (u *UDP) SetReadDeadline(t time.Time) error {
	select {
	case <-u.closedChan:
		return ErrClosed
	default:
	}

	u.readDeadline = t

	return nil
}

func (u *UDP) SetWriteDeadline(t time.Time) error {
	select {
	case <-u.closedChan:
		return ErrClosed
	default:
	}

	u.writeDeadline = t

	return nil
}

func (u *UDP) handle() {
	defer u.conn.Close()

	for {
		buf := make([]byte, packetSize)
		n, addr, err := u.conn.ReadFrom(buf)
		if err != nil {
			if u.IsClosed() {
				break
			}

			continue
		}

		u.lock.Lock()
		conn, found := u.connectedConns[addr.String()]
		u.lock.Unlock()

		if found {
			select {
			case conn.readBuf <- buf[:n]:
			default:
			}

			continue
		}

		select {
		case u.readBuf <- udpBuf{
			addr: addr,
			buf:  buf[:n],
		}:
		default:
		}
	}
}

func NewUDP(network string, address string) (*UDP, error) {
	conn, err := net.ListenPacket(network, address)
	if err != nil {
		return nil, err
	}

	u := &UDP{
		conn: conn,

		closedChan: make(chan interface{}),
		readBuf:    make(chan udpBuf, bufferLen),

		readDeadline:  time.Time{},
		writeDeadline: time.Time{},

		lock:           sync.Mutex{},
		connectedConns: map[string]*UDPConn{},
	}
	go u.handle()

	return u, nil
}
