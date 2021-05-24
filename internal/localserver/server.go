package localserver

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
	"jonwillia.ms/goproxy-p2p/internal/util"
)

// ListenAndServe returns a server that serves from the remoteServers
func ListenAndServe(ctx context.Context, port int, remoteServers <-chan map[string]string) (*net.TCPAddr, error) {
	e := &ephemeral{}
	e.setProxies(map[string]string{})

	server := http.Server{
		Handler: e,
	}
	addr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: port, // TODO alloc from command-line options
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() { server.Serve(l) }()
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()
	go func() {
		for {
			select {
			case newProxies := <-remoteServers:
				log.Info().Int("numProxies", len(newProxies)).Msg("receiving proxy list")
				e.setProxies(newProxies)
				log.Trace().Int("numProxies", len(newProxies)).Msg("received proxy list")

			case <-ctx.Done():
				return
			}
		}
	}()
	return addr, nil
}

type ephemeral struct {
	mutex   sync.Mutex
	goproxy *goproxy.Goproxy
}

var _ http.Handler = &ephemeral{}

func (e *ephemeral) setProxies(proxies map[string]string) {

	goBinEnv := map[string]string{}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		goBinEnv[parts[0]] = parts[1]
	}

	gp := ""

	for _, proxy := range proxies {
		if gp != "" {
			gp += "|"
		}
		gp += proxy
	}

	if gp == "" {
		gp = "off"
	}
	goBinEnv["GOPROXY"] = gp
	log.Info().Str("GOPROXY", gp).Msg("new GOPROXY for localserver")

	newBinEnv := make([]string, 0, len(goBinEnv))
	for k, v := range goBinEnv {
		newBinEnv = append(newBinEnv,
			k+"="+v,
		)
	}

	p := &goproxy.Goproxy{
		ErrorLogger: util.Logger(log.Logger, "localserver"),
		GoBinEnv:    newBinEnv,
		// TODO do a multi cacher here instead of a proxy hierarchy
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.goproxy = p
}

func (e *ephemeral) getProxy() *goproxy.Goproxy {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.goproxy
}

func (e *ephemeral) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	p := e.getProxy()
	log.Trace().Msg("serve http")
	p.ServeHTTP(resp, req)
}
