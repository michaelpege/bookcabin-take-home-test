package redis

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestClientUnreachableFailsPromptly(t *testing.T) {
	// Pick a closed loopback port by binding and immediately closing a listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen temp port: %v", err)
	}
	closedAddr := ln.Addr().String()
	_ = ln.Close()

	cache, pingErr := NewClient(closedAddr, WithOperationTimeout(50*time.Millisecond))
	defer func() { _ = cache.Close() }()

	if pingErr == nil {
		t.Fatalf("expected NewClient ping error on unreachable Redis address %s", closedAddr)
	}
	if _, err := cache.Get(context.Background(), "flight_search:CGK:DPS:2025-12-15:1:economy"); err == nil {
		t.Fatalf("expected Get error on unreachable Redis address %s", closedAddr)
	}
	if err := cache.Set(context.Background(), "k", []byte("v"), time.Minute); err == nil {
		t.Fatalf("expected Set error on unreachable Redis address %s", closedAddr)
	}
	if err := cache.Set(context.Background(), "k", []byte("v"), 0); err != nil {
		t.Fatalf("expected zero-TTL Set to be a no-op, got %v", err)
	}
}
