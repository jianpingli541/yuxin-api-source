package controller

import (
	"net"
	"testing"
)

func TestValidateOutboundBaseURLRejectsInternal(t *testing.T) {
	bad := []string{
		"http://127.0.0.1:9090",
		"http://localhost:3000",
		"http://169.254.169.254/latest/meta-data",
		"http://192.168.1.1",
		"http://10.0.0.5",
	}
	for _, u := range bad {
		if err := validateOutboundBaseURL(u); err == nil {
			t.Fatalf("expected rejection for %s", u)
		}
	}
}

func TestIsPrivateOrLocalIP(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1":       true,
		"10.1.2.3":        true,
		"172.16.0.1":      true,
		"172.31.255.255":  true,
		"172.32.0.1":      false,
		"192.168.0.1":     true,
		"169.254.169.254": true,
		"100.64.0.1":      true,
		"8.8.8.8":         false,
		"1.1.1.1":         false,
	}
	for s, want := range cases {
		ip := net.ParseIP(s)
		if got := isPrivateOrLocalIP(ip); got != want {
			t.Fatalf("isPrivateOrLocalIP(%s) = %v, want %v", s, got, want)
		}
	}
}
