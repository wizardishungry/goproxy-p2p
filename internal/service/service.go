package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"jonwillia.ms/goproxy-p2p/internal/agent"
	"jonwillia.ms/goproxy-p2p/internal/localserver"
	"jonwillia.ms/goproxy-p2p/internal/proto"
	"jonwillia.ms/goproxy-p2p/internal/proxied"
	"jonwillia.ms/goproxy-p2p/internal/util"
	"jonwillia.ms/weyoun"
	"jonwillia.ms/weyoun/pkg/handlers"
)

const (
	serviceName = "_goproxyp2p._tcp" // must be in this format
	username    = "goproxyp2p"
)

type Instance struct {
	Serve     bool
	LocalPort int // port for the localserver
	Password  string
}

func (i *Instance) GetConfig() *agent.Config {
	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", "127.0.0.1", i.LocalPort),
	}
	v := url.Values{}
	v.Add("pass", i.Password)
	u.RawQuery = v.Encode()
	return &agent.Config{
		GOPROXY: u.String(),
		// PID: ,
	}
}

func (i *Instance) Start(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	px := proxied.New()

	// this is an proxy for the local user
	localAddr, err := localserver.ListenAndServe(ctx, i.LocalPort, i.Password, px.Updates())
	if err != nil {
		return fmt.Errorf("localserver.ListenAndServe: %w", err)
	}
	log.Info().Msgf("local server %v", localAddr)
	i.LocalPort = localAddr.Port // in case we pass 0

	myServerID := ""
	_ = myServerID
	if i.Serve {
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
		myServerID = ws.GetID()
		keys, err := ws.GetAuthorizedKeys()
		if err != nil {
			log.Error().Err(err).Msg("GetAuthorizedKeys")
		}

		for _, key := range keys {
			logKeys(key).
				Msg("accepting")
		}
		log.Info().Str("id", ws.GetID()).
			Msg("announcing caching service to to network")
	}

	wc := weyoun.NewClient(serviceName, px.CallbackAdd(), px.CallbackRemove(), []string{
		myServerID, // TODO make matching localhost configurable
	})
	err = wc.Run(ctx)
	if err != nil {
		return fmt.Errorf("wc.Run: %w", err)
	}

	keys, err := wc.ListPublic()
	if err != nil {
		log.Error().Err(err).Msg("ListPublic")
	}
	for _, key := range keys {
		logKeys(key).
			Msg("identity available")
	}

	return nil
}

func logKeys(key ssh.PublicKey) *zerolog.Event {
	keyStr := string(ssh.MarshalAuthorizedKey(key))
	fp := ssh.FingerprintLegacyMD5(key)
	keyStr = keyStr[:len(keyStr)-1]
	return log.Debug().
		Str("authorized_key", keyStr).
		Str("fingerprint", fp)
}
