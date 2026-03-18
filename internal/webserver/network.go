package webserver

import (
	"net"
)

// NetworkInterface represents a network interface on the machine that can
// be used to bind the HTTPS web server.
type NetworkInterface struct {
	Name        string
	IP          string
	IsTailscale bool
}

// tailscaleCIDR is the CGNAT range used by Tailscale (100.64.0.0/10).
var tailscaleCIDR *net.IPNet

func init() {
	_, tailscaleCIDR, _ = net.ParseCIDR("100.64.0.0/10")
}

// IsTailscaleIP reports whether ip falls within the Tailscale CGNAT range (100.64.0.0/10).
func IsTailscaleIP(ip net.IP) bool {
	return tailscaleCIDR.Contains(ip)
}

// ListInterfaces returns all active non-loopback network interfaces with IPv4 addresses,
// excluding link-local unicast addresses. Each entry includes whether the IP is in the
// Tailscale CGNAT range (100.64.0.0/10).
func ListInterfaces() ([]NetworkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInterface, 0)

	for _, iface := range ifaces {
		// Skip interfaces that are down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}

			// IPv4 only — VPN interfaces are always IPv4, keeps the dropdown clean
			if ip.To4() == nil {
				continue
			}

			result = append(result, NetworkInterface{
				Name:        iface.Name,
				IP:          ip.String(),
				IsTailscale: IsTailscaleIP(ip),
			})
		}
	}

	return result, nil
}
