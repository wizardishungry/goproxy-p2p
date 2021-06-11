package agent

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestConnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const path = "/tmp/whatever"
	agent, err := listen(ctx, path)
	if err != nil {
		t.Errorf("listen returned an error %v", err)
	}
	if agent == nil {
		t.Error("connectOrListen returned a nil agent")
	}
	agent.SetConfig(&Config{GOPROXY: "hi mom"})

	time.Sleep(time.Second)

	agent, config, err := connectOrListen(ctx, path, false)
	if err != nil {
		t.Errorf("connectOrListen returned an error %v", err)
	}
	if agent != nil {
		t.Error("connectOrListen returned an agent")
	}
	if config == nil {
		t.Error("connectOrListen returned nil config")
	}
	fmt.Println(config)
}
