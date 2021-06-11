package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"syscall"

	"github.com/rs/zerolog/log"
)

// Agent is for keeping a single instance of this thing running
// like ssh-agent
type Agent struct {
	v atomic.Value
}

func (a *Agent) SetConfig(c *Config) {
	a.v.Store(c)
}

type Config struct {
	GOPROXY string // append
	PID     int
}

func (c *Config) ShellEval() {
	if c == nil || c.GOPROXY == "" {
		fmt.Println("echo problem setting up GOPROXY")
		return
	}
	fmt.Printf("export GOPROXY=%s\n", c.GOPROXY)
}

// ConnectOrNew returns a connection to the domain socket
func ConnectOrNew(ctx context.Context, mustConnect bool) (agent *Agent, existing *Config, err error) {
	path, err := os.UserConfigDir()
	if err != nil {
		err = fmt.Errorf("os.UserConfigDir: %w", err)
		return
	}
	path += "/goproxy-p2p.sock"
	return connectOrListen(ctx, path, mustConnect)
}

func connectOrListen(ctx context.Context, path string, mustConnect bool) (agent *Agent, existing *Config, err error) {
	existing, err = connect(ctx, path)
	if err != nil && mustConnect {
		return
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.ECONNREFUSED:
			_ = os.Remove(path)
			fallthrough
		case syscall.ENOENT:
			agent, err = listen(ctx, path)
		}
	}
	return
}

func connect(ctx context.Context, path string) (*Config, error) {
	cn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("net.Dial: %w", err)
	}
	dec := json.NewDecoder(cn)
	var c = &Config{}
	err = dec.Decode(c)
	return c, err
}

func listen(ctx context.Context, path string) (*Agent, error) {
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("net.Listen: %w", err)
	}
	go func() {
		<-ctx.Done()
		ln.Close()
		log.Trace().Err(err).Msgf("domain socket agent exiting")
	}()
	a := Agent{}
	go func() {
		for {
			cn, err := ln.AcceptUnix()
			if err != nil {
				log.Trace().Err(err).Msgf("domain socket agent exiting")
				continue
			}
			go func() {
				defer cn.Close()
				enc := json.NewEncoder(cn)
				var c *Config
				c = a.v.Load().(*Config)
				err := enc.Encode(c)
				if err != nil {
					log.Error().Err(err).Msgf("domain socket agent problem with json encoder")
				}
			}()
		}
	}()
	return &a, nil
}
