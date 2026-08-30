//go:build windows

package platform

import (
	"context"
	"os"
	"testing"
)

func TestWindowsPort(t *testing.T) {
	for _, tt := range []struct {
		in   uint32
		want int
	}{
		{0x5000, 80},
		{0x901f, 8080},
		{0xbb01, 443},
	} {
		if got := windowsPort(tt.in); got != tt.want {
			t.Errorf("windowsPort(%#x) = %d; want %d", tt.in, got, tt.want)
		}
	}
}

func TestWindowsAddresses(t *testing.T) {
	if got := windowsIPv4(0x0100007f); got != "127.0.0.1" {
		t.Errorf("windowsIPv4(loopback) = %q; want 127.0.0.1", got)
	}
	var loopback [16]byte
	loopback[15] = 1
	if got := windowsIPv6(loopback, 0); got != "::1" {
		t.Errorf("windowsIPv6(loopback) = %q; want ::1", got)
	}
	linkLocal := [16]byte{0xfe, 0x80}
	if got := windowsIPv6(linkLocal, 12); got != "fe80::%12" {
		t.Errorf("windowsIPv6(link-local) = %q; want fe80::%%12", got)
	}
}

func TestWindowsProviderSmoke(t *testing.T) {
	provider, err := newProvider(nil)
	if err != nil {
		t.Fatalf("newProvider: %v", err)
	}
	sockets, err := provider.ListListening(context.Background())
	if err != nil {
		t.Fatalf("ListListening: %v", err)
	}
	for _, socket := range sockets {
		if socket.Port < 1 || socket.Port > 65535 {
			t.Errorf("invalid socket port: %+v", socket)
		}
		switch socket.Protocol {
		case "tcp", "tcp6", "udp", "udp6":
		default:
			t.Errorf("invalid socket protocol: %+v", socket)
		}
	}
}

func TestWindowsCurrentProcessIsAlive(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatal("current process reported as not alive")
	}
}
