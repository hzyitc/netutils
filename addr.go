package netutils

import (
	"net"
	"strconv"
	"strings"
)

const (
	tooManyColons      = "too many colons in address"
	noSuchHost         = "no such host"
	missingPort        = "missing port in address"
	missingPortOrNoSRV = "missing port in address or no such SRV"
	unknownPort        = "unknown port"
)

type Addr struct {
	Type string
	IP   net.IP
	Port int
}

func (a *Addr) Network() string { return a.Type }

func (a *Addr) String() string {
	if a == nil {
		return "<nil>"
	}

	ip := ""
	if len(a.IP) != 0 {
		ip = a.IP.String()
	}
	return net.JoinHostPort(ip, strconv.Itoa(a.Port))
}

func (a *Addr) ToTCPAddr() *net.TCPAddr {
	return &net.TCPAddr{
		IP:   a.IP,
		Port: a.Port,
	}
}

func (a *Addr) ToUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   a.IP,
		Port: a.Port,
	}
}

func ParseIP(s string) net.IP {
	return net.ParseIP(s)
}

func IsVaildPort(port int) bool {
	return (0 <= port && port <= 65535)
}

func ParsePort(network string, s string) int {
	port, err := net.LookupPort(network, s)
	if err != nil {
		return -1
	}

	if !IsVaildPort(port) {
		return -1
	}

	return port
}

func resolveType(network string) string {
	switch network {
	case "udp", "udp4", "udp6":
		return "udp"
	case "tcp", "tcp4", "tcp6":
		return "tcp"
	default:
		return ""
	}
}

func ParseAddr(network string, s string) *Addr {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return nil
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}

	portnum := ParsePort(network, port)
	if portnum == -1 {
		return nil
	}

	return &Addr{
		Type: resolveType(network),
		IP:   ip,
		Port: portnum,
	}
}

func ResolveIP(address string) (net.IP, error) {
	ip := ParseIP(address)
	if ip != nil {
		return ip, nil
	}

	ips, err := net.LookupIP(address)
	if err != nil {
		return nil, err
	}

	return ips[0], nil
}

func ResolveAddr(network string, address string, service string, defaultPort int) (*Addr, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		if !strings.Contains(err.Error(), missingPort) {
			return nil, err
		}

		_, records, err := net.LookupSRV(service, network, address)
		if err == nil {
			for _, r := range records {
				ip, err := ResolveIP(r.Target)
				if err == nil {
					return &Addr{
						Type: resolveType(network),
						IP:   ip,
						Port: int(r.Port),
					}, nil
				}
			}
		}

		if IsVaildPort(defaultPort) {
			ip, err := ResolveIP(address)
			if err != nil {
				return nil, err
			}

			return &Addr{
				Type: resolveType(network),
				IP:   ip,
				Port: defaultPort,
			}, nil

		}

		return nil, &net.AddrError{
			Err:  missingPortOrNoSRV,
			Addr: address,
		}
	}

	ip, err := ResolveIP(host)
	if err != nil {
		return nil, err
	}

	portnum := ParsePort(network, port)
	if portnum == -1 {
		return nil, &net.AddrError{
			Err:  unknownPort,
			Addr: address,
		}
	}

	return &Addr{
		Type: resolveType(network),
		IP:   ip,
		Port: portnum,
	}, nil
}
