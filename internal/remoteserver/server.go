package remoteserver

import (
	"bytes"
	"context"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/goproxy/goproxy"
	"github.com/rs/zerolog/log"
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
// TODO add auth
func ListenAndServe(ctx context.Context) (*net.TCPAddr, error) {

	goBinEnv := map[string]string{}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		goBinEnv[parts[0]] = parts[1]
	}

	goBinEnv["GOPROXY"] = "off" // Do not fetch

	newBinEnv := make([]string, 0, len(goBinEnv))
	for k, v := range goBinEnv {
		newBinEnv = append(newBinEnv,
			k+"="+v,
		)
	}

	gomodcache, err := runCmd("go", "env", "GOMODCACHE")
	if err != nil {
		return nil, fmt.Errorf("couldn't figure out GOMODCACHE: %w", err)
	}
	log.Info().Str("gomodcache", gomodcache).Msg("Found cache")

	stdLogger := stdlog.New(log.Logger, "remoteserver", 0)

	p := &goproxy.Goproxy{
		Cacher:      goproxy.DirCacher(gomodcache),
		GoBinEnv:    newBinEnv,
		ErrorLogger: stdLogger,
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
