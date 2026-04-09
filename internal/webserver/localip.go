package webserver

import (
	"errors"
	"net"
)

// GetLANIP returns the best available private IPv4 address on this machine,
// suitable for binding the local-mode HTTPS server.
//
// Preference order (important for multi-interface machines with VPN + Wi-Fi):
//  1. Named preferred interfaces: en0 (macOS Wi-Fi), eth0 (Linux), wlan0 (Linux Wi-Fi)
//  2. Any interface that is up, non-loopback, and has a private IPv4
//
// Tailscale CGNAT addresses (100.64.0.0/10, i.e. ip[0]==100 && ip[1] in 64–127)
// are explicitly excluded so the local server never binds on the Tailscale
// interface — that would defeat the purpose of the fallback.
func GetLANIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// First pass: try well-known preferred interface names.
	for _, name := range []string{"en0", "eth0", "wlan0"} {
		if ip := ipFromInterface(ifaces, name); ip != "" {
			return ip, nil
		}
	}

	// Second pass: iterate all interfaces; skip loopback and down interfaces.
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if ip := firstPrivateIPv4(iface); ip != "" {
			return ip, nil
		}
	}

	return "", errors.New("no suitable LAN IP found")
}

// IsTailscaleIP reports whether ip falls in the Tailscale CGNAT range
// 100.64.0.0/10 (100.64.x.x through 100.127.x.x).
//
// Exported for testing. The plan calls for an unexported helper; we export it
// so tests in the _test package can validate the range boundary conditions
// without importing an internal symbol directly.
func IsTailscaleIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127
}

// isTailscaleIP is the unexported alias used internally.
func isTailscaleIP(ip net.IP) bool { return IsTailscaleIP(ip) }

// ipFromInterface searches ifaces for the interface named name and returns
// the first suitable private IPv4 address found on it.
func ipFromInterface(ifaces []net.Interface, name string) string {
	for _, iface := range ifaces {
		if iface.Name != name {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		return firstPrivateIPv4(iface)
	}
	return ""
}

// firstPrivateIPv4 returns the first private (RFC 1918) IPv4 address on iface,
// skipping loopback, link-local (169.254.x.x), and Tailscale CGNAT addresses.
func firstPrivateIPv4(iface net.Interface) string {
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		ip4 := ip.To4()
		if ip4 == nil {
			continue // IPv6 — skip
		}
		if ip4.IsLoopback() {
			continue
		}
		if ip4.IsLinkLocalUnicast() {
			continue // 169.254.x.x
		}
		if isTailscaleIP(ip4) {
			continue // 100.64.0.0/10
		}
		// Private RFC 1918 ranges:
		//   10.0.0.0/8       ip4[0] == 10
		//   172.16.0.0/12    ip4[0] == 172 && ip4[1] in [16,31]
		//   192.168.0.0/16   ip4[0] == 192 && ip4[1] == 168
		if ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) {
			return ip4.String()
		}
	}
	return ""
}
