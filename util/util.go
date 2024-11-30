package util

import (
	"golang.org/x/net/idna"
	"net"
	"net/http"
	"strings"
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

func GetAllIPs() (ips []net.IP) {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			addresses, err := i.Addrs()
			if err == nil {
				for _, addr := range addresses {
					ip, _ := AddrToIPPort(addr)
					if ip != nil {
						ips = append(ips, ip)
					}
				}
			}
		}
	}
	return ips
}

func GetLocalhostIPs() (ips []net.IP) {
	allIPs := GetAllIPs()
	for _, ip := range allIPs {
		if ip.IsLoopback() {
			ips = append(ips, ip)
		}
	}
	return ips
}

func MatchCDNOriginHost(r *http.Request, getCDNOriginHosts func() []string) bool {
	var cdnOriginHosts []string
	if getCDNOriginHosts != nil {
		cdnOriginHosts = getCDNOriginHosts()
	}
	clientHost, _ := idna.ToASCII(r.Host)
	for _, host := range cdnOriginHosts {
		host, _ := idna.ToASCII(host)
		if strings.EqualFold(clientHost, host) {
			return true
		}
	}
	return false
}

func MatchSafeHTTPHeaderKey(r *http.Request, getSafeHTTPHeaderKeys func() []string) bool {
	var safeHTTPHeaderKeys []string
	if getSafeHTTPHeaderKeys != nil {
		safeHTTPHeaderKeys = getSafeHTTPHeaderKeys()
	}
	for _, h := range safeHTTPHeaderKeys {
		if r.Header.Get(h) != "" {
			return true
		}
	}
	return false
}
