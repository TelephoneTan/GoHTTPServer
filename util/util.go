package util

import (
	"net"
)

func ShallowCloneSlice[T any](src []T) []T {
	return append(make([]T, 0), src...)
}

func IsPureASCII(s string) bool {
	for _, c := range s {
		if c < 0 || c > 127 {
			return false
		}
	}
	return true
}

func New[T any](t *T, init ...func(*T)) *T {
	if len(init) > 0 {
		init[0](t)
	}
	return t
}

func AddrToIPPort(addr net.Addr) (ip net.IP, port *uint16) {
	switch addr := addr.(type) {
	case *net.IPAddr:
		ip = append(ip, addr.IP...)
	case *net.IPNet:
		ip = append(ip, addr.IP...)
	case *net.TCPAddr:
		ip = append(ip, addr.IP...)
		p := uint16(addr.Port)
		port = &p
	case *net.UDPAddr:
		ip = append(ip, addr.IP...)
		p := uint16(addr.Port)
		port = &p
	}
	return ip, port
}
