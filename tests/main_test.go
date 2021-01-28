package tests

import (
	"net"
	"testing"
)

func listenTCPWithRandomPort(t *testing.T) net.Listener {
	t.Helper()
	taddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to resolve TCP addr: %+v", err)
	}
	lis, err := net.Listen("tcp", taddr.String())
	if err != nil {
		t.Fatalf("failed to listen TCP on %v: %+v", taddr.String(), err)
	}
	return lis
}
