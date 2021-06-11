package proxied

import (
	"context"
	"sync"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"jonwillia.ms/goproxy-p2p/internal/proto"
)

// TODO this is more like remote caches

func New() *Proxied {
	p := &Proxied{
		updates: make(chan []goproxy.Cacher, 1),
		cachers: make(map[*ssh.Client]closeCacher),
	}
	p.updates <- p.update()
	return p
}

type Proxied struct {
	sync.Mutex
	cachers map[*ssh.Client]closeCacher
	updates chan []goproxy.Cacher
}

type closeCacher interface {
	goproxy.Cacher
	Close() error
}

func (p *Proxied) CallbackAdd() func(ctx context.Context, sshClient *ssh.Client) {
	return func(ctx context.Context, sshClient *ssh.Client) {
		log.Info().Str("addr", sshClient.RemoteAddr().String()).Msg("connected")

		channel, reqs, err := sshClient.OpenChannel("cacher", nil) // TODO constant

		if err != nil {
			log.Error().Err(err).Msg("sshClient.OpenChannel")
		}
		go ssh.DiscardRequests(reqs)

		var rpc closeCacher = proto.New(channel, sshClient.RemoteAddr())

		p.Mutex.Lock()
		defer p.Mutex.Unlock()

		p.stop(sshClient)
		p.cachers[sshClient] = rpc
		log.Info().Int("numProxies", len(p.cachers)).Msg("updating remote cachers")
		p.updates <- p.update()
	}
}

func (p *Proxied) CallbackRemove() func(ctx context.Context, sshClient *ssh.Client) {
	return func(ctx context.Context, sshClient *ssh.Client) {
		p.Mutex.Lock()
		defer p.Mutex.Unlock()
		p.stop(sshClient)
		p.updates <- p.update()
	}
}

func (p *Proxied) stop(sshClient *ssh.Client) {
	delete(p.cachers, sshClient)

	l, ok := p.cachers[sshClient]
	if !ok {
		return
	}
	err := l.Close()
	if err != nil {
		log.Error().Err(err).Msg("error removing remote cacher")
	}
	delete(p.cachers, sshClient)
}

func (p *Proxied) Updates() <-chan []goproxy.Cacher {
	return p.updates
}

func (p *Proxied) update() []goproxy.Cacher {
	ret := make([]goproxy.Cacher, 0, len(p.cachers))
	for _, c := range p.cachers {
		ret = append(ret, c)
	}
	return ret
}
