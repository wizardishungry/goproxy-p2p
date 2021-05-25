package localserver

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
	"jonwillia.ms/goproxy-p2p/internal/util"
)

// ListenAndServe returns a server that serves from the remoteServers
func ListenAndServe(ctx context.Context, port int, remoteServers <-chan []goproxy.Cacher) (*net.TCPAddr, error) {
	e := &ephemeral{}
	e.setRemotes(nil)

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
				e.setRemotes(newProxies)
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

func (e *ephemeral) setRemotes(cachers []goproxy.Cacher) {
	log.Trace().Int("len", len(cachers)).Msg("setRemotes")
	// goBinEnv := map[string]string{}
	// for _, env := range os.Environ() {
	// 	parts := strings.SplitN(env, "=", 2)
	// 	if len(parts) != 2 {
	// 		continue
	// 	}
	// 	goBinEnv[parts[0]] = parts[1]
	// }

	// // goBinEnv["GOPROXY"] = "direct" // Do not fetch?
	// // goBinEnv["GOVCS"] = "*:off"
	// // goBinEnv["SSH_AUTH_SOCK"] = `/dev/null`

	// newBinEnv := make([]string, 0, len(goBinEnv))
	// for k, v := range goBinEnv {
	// 	newBinEnv = append(newBinEnv,
	// 		k+"="+v,
	// 	)
	// }
	myCacher := util.Gomodcacher()

	multiCacher := newMulticacher()
	multiCacher.Add(myCacher) // try me first
	multiCacher.Add(cachers...)

	p := &goproxy.Goproxy{
		// GoBinEnv: newBinEnv,
		Cacher: &cacherLogger{multiCacher},
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
	// TODO add auth so other local users can't hit your proxy
	p := e.getProxy()
	log.Trace().Msg("serve http")
	p.ServeHTTP(resp, req)
}

type cacherLogger struct {
	cacher goproxy.Cacher
}

func (cl *cacherLogger) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	rc, err := cl.cacher.Get(ctx, name)
	log.Trace().Err(err).Str("name", name).Bool("found", rc != nil).Msg("Get")

	return rc, err
}

func (cl *cacherLogger) Set(ctx context.Context, name string, content io.ReadSeeker) error {
	err := cl.cacher.Set(ctx, name, content)
	log.Trace().Err(err).Str("name", name).Msg("Set")
	return err
}

var _ goproxy.Cacher = &cacherLogger{}
