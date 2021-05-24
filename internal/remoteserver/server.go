package remoteserver

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
	"jonwillia.ms/goproxy-p2p/internal/util"
)

func runCmd(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), err
}

// ListenAndServe returns a server that serves from the gomodcache for lan clients
func ListenAndServe(ctx context.Context) (*net.TCPAddr, error) {

	dir, err := os.MkdirTemp("", "goproxy-p2p-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	goBinEnv := map[string]string{}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		goBinEnv[parts[0]] = parts[1]
	}

	// goBinEnv["GOPROXY"] = "direct" // Do not fetch?
	// goBinEnv["GOVCS"] = "*:off"
	// goBinEnv["SSH_AUTH_SOCK"] = `/dev/null`

	newBinEnv := make([]string, 0, len(goBinEnv))
	for k, v := range goBinEnv {
		newBinEnv = append(newBinEnv,
			k+"="+v,
		)
	}

	gomodcache, err := runCmd("go", "env", "GOMODCACHE")
	if err != nil {
		return nil, err
	}
	dir = gomodcache + "/cache/download"
	log.Debug().Str("path", dir).Msg("path to cache")

	p := &goproxy.Goproxy{
		GoBinEnv: newBinEnv,
		// Cacher: goproxy.DirCacher(dir),
		Cacher:      &cacherLogger{goproxy.DirCacher(dir)},
		ErrorLogger: util.Logger(log.Logger, "remoteserver"),
	}

	server := http.Server{
		Handler: p,
	}
	addr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0, // random port
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
	addr.Port = l.Addr().(*net.TCPAddr).Port
	return addr, nil
}

type cacherLogger struct {
	cacher goproxy.Cacher
}

func (cl *cacherLogger) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	rc, err := cl.cacher.Get(ctx, name)
	log.Trace().Err(err).Str("name", name).Msg("Get")

	return rc, err
}

func (cl *cacherLogger) Set(ctx context.Context, name string, content io.ReadSeeker) error {
	err := cl.cacher.Set(ctx, name, content)
	log.Trace().Err(err).Str("name", name).Msg("Set")
	return err
}

var _ goproxy.Cacher = &cacherLogger{}
