package proto

import (
	"context"
	"testing"

	"jonwillia.ms/goproxy-p2p/internal/util"
)

func TestNewServer(t *testing.T) {
	gmc := util.Gomodcacher()

	s, err := NewServer(context.Background(), gmc)
	if err != nil {
		t.Fatalf("NewServer %v", err)
	}
	_ = s
}
