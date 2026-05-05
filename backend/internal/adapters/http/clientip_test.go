package http

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPResolverWithoutTrustedProxy(t *testing.T) {
	resolve := newClientIPResolver("")
	req := httptest.NewRequest("GET", "/api/health", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")

	ip := resolve(req)
	if ip != "10.0.0.2" {
		t.Fatalf("expected direct remote ip, got %s", ip)
	}
}

func TestClientIPResolverUsesForwardedForFromTrustedProxy(t *testing.T) {
	resolve := newClientIPResolver("10.0.0.0/8")
	req := httptest.NewRequest("GET", "/api/health", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.2")

	ip := resolve(req)
	if ip != "203.0.113.50" {
		t.Fatalf("expected forwarded client ip, got %s", ip)
	}
}

func TestClientIPResolverIgnoresForwardedForFromUntrustedProxy(t *testing.T) {
	resolve := newClientIPResolver("10.0.0.0/8")
	req := httptest.NewRequest("GET", "/api/health", nil)
	req.RemoteAddr = "198.51.100.10:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	ip := resolve(req)
	if ip != "198.51.100.10" {
		t.Fatalf("expected remote ip for untrusted proxy, got %s", ip)
	}
}

func TestClientIPResolverFallsBackOnInvalidCIDR(t *testing.T) {
	cases := []struct {
		name string
		cidr string
	}{
		{"garbage", "not-a-cidr"},
		{"bad prefix length", "10.0.0.0/99"},
		{"missing prefix", "10.0.0.0"},
		{"only prefix", "/8"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resolve := newClientIPResolver(c.cidr)
			req := httptest.NewRequest("GET", "/api/health", nil)
			req.RemoteAddr = "10.0.0.2:1234"
			req.Header.Set("X-Forwarded-For", "203.0.113.50")

			ip := resolve(req)
			if ip != "10.0.0.2" {
				t.Fatalf("invalid CIDR %q must disable XFF parsing, got %s", c.cidr, ip)
			}
		})
	}
}
