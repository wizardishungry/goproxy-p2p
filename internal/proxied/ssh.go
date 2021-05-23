package proxied

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

func New() *Proxied {
	p := &Proxied{
		proxies:   make(map[*ssh.Client]*net.TCPListener),
		updates:   make(chan map[string]string, 1),
		goproxies: make(map[string]string),
	}
	p.updates <- p.goproxies
	return p
}

type Proxied struct {
	sync.Mutex
	proxies   map[*ssh.Client]*net.TCPListener
	goproxies map[string]string
	updates   chan map[string]string
}

func (p *Proxied) CallbackAdd() func(ctx context.Context, sshClient *ssh.Client) {
	return func(ctx context.Context, sshClient *ssh.Client) {
		log.Info().Str("addr", sshClient.RemoteAddr().String()).Msg("connected")

		rp := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "255.255.255.255:666"})
		rp.Transport = &http.Transport{
			Dial: sshClient.Dial,
		}
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		if err != nil {
			log.Error().Err(err).Msg("can't listen")
			return
		}
		u := fmt.Sprintf("http://127.0.0.1:%d/", listener.Addr().(*net.TCPAddr).Port)
		rk := sshClient.RemoteAddr().String()
		go func() { http.Serve(listener, rp) }()
		p.Mutex.Lock()
		defer p.Mutex.Unlock()
		p.stop(sshClient)
		p.goproxies[rk] = u
		p.proxies[sshClient] = listener
		log.Info().Int("numProxies", len(p.goproxies)).Msg("updating proxies")
		p.updates <- p.goproxies
	}
}

func (p *Proxied) CallbackRemove() func(ctx context.Context, sshClient *ssh.Client) {
	return func(ctx context.Context, sshClient *ssh.Client) {
		p.Mutex.Lock()
		defer p.Mutex.Unlock()
		p.stop(sshClient)
		p.updates <- p.goproxies
	}
}

func (p *Proxied) stop(sshClient *ssh.Client) {
	rk := sshClient.RemoteAddr().String()
	delete(p.goproxies, rk)

	l, ok := p.proxies[sshClient]
	if !ok {
		return
	}
	l.Close()
	delete(p.proxies, sshClient)
}

func (p *Proxied) Updates() <-chan map[string]string {
	return p.updates // race! TODO
}
