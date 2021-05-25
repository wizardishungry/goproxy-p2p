package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"jonwillia.ms/goproxy-p2p/internal/localserver"
	"jonwillia.ms/goproxy-p2p/internal/proto"
	"jonwillia.ms/goproxy-p2p/internal/proxied"
	"jonwillia.ms/goproxy-p2p/internal/util"
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
	localAddr, err := localserver.ListenAndServe(ctx, i.LocalPort, px.Updates())
	if err != nil {
		return fmt.Errorf("localserver.ListenAndServe: %w", err)
	}
	log.Info().Msgf("local server %v", localAddr)

	i.LocalPort = localAddr.Port // in case we pass 0
	myCacher := util.Gomodcacher()
	ws := weyoun.NewServer(serviceName, handlers.Handlers{
		FreeForm: map[string]func(ctx context.Context, channel ssh.Channel, extra []byte){
			"cacher": func(ctx context.Context, channel ssh.Channel, extra []byte) { // TODO const
				s, err := proto.NewServer(ctx, myCacher)
				if err != nil {
					log.Error().Err(err).Msg("proto.NewServer")
					return
				}
				log.Info().Msg("Serving connection")
				s.ServeConn(channel)
			},
		},
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
