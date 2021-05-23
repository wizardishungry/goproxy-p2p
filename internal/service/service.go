package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"jonwillia.ms/goproxy-p2p/internal/localserver"
	"jonwillia.ms/goproxy-p2p/internal/proxied"
	"jonwillia.ms/goproxy-p2p/internal/remoteserver"
	"jonwillia.ms/weyoun"
	"jonwillia.ms/weyoun/pkg/handlers"
)

type Instance struct {
	LocalPort int // port for the localserver
}

const serviceName = "_goproxyp2p._tcp" // ust be in this format

func (i *Instance) Start(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	px := proxied.New()

	// this is an agent for the local user
	localAddr, err := localserver.ListenAndServe(ctx, px.Updates())
	if err != nil {
		return fmt.Errorf("localserver.ListenAndServe: %w", err)
	}
	log.Info().Msgf("local server %v", localAddr)

	i.LocalPort = localAddr.Port // in case we pass 0

	remoteAddr, err := remoteserver.ListenAndServe(ctx)
	if err != nil {
		return fmt.Errorf("remoteserver.ListenAndServe: %w", err)
	}
	log.Info().Msgf("remote server %v", remoteAddr)

	ws := weyoun.NewServer(serviceName, func(ctx context.Context, channel ssh.Channel, msg handlers.ChannelOpenDirectMsg) {
		conn, err := net.Dial("tcp", remoteAddr.String())
		if err != nil {
			fmt.Println("dial error", err)
			return
		}
		defer conn.Close()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			io.Copy(channel, conn)
		}()
		go func() {
			defer wg.Done()
			io.Copy(conn, channel)
		}()
		wg.Wait()
	})

	err = ws.Run(ctx)
	if err != nil {
		return fmt.Errorf("ws.Run: %w", err)
	}

	wc := weyoun.NewClient(serviceName, px.CallbackAdd(), px.CallbackRemove())
	err = wc.Run(ctx)
	if err != nil {
		return fmt.Errorf("wc.Run: %w", err)
	}

	return nil
}
