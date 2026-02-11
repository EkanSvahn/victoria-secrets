package http

import (
	"net"
	"net/http"
	"strings"
)

func newClientIPResolver(trustedProxyCIDR string) func(*http.Request) string {
	_, trustedNet, err := net.ParseCIDR(strings.TrimSpace(trustedProxyCIDR))
	if err != nil {
		trustedNet = nil
	}

	return func(r *http.Request) string {
		direct := remoteIP(r)
		if trustedNet == nil {
			return direct
		}
		parsedDirect := net.ParseIP(direct)
		if parsedDirect == nil || !trustedNet.Contains(parsedDirect) {
			return direct
		}

		xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
		if xff == "" {
			return direct
		}
		parts := strings.Split(xff, ",")
		for _, p := range parts {
			candidate := strings.TrimSpace(p)
			ip := net.ParseIP(candidate)
			if ip != nil {
				return ip.String()
			}
		}
		return direct
	}
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil || host == "" {
		if r.RemoteAddr == "" {
			return "unknown"
		}
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
